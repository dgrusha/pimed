package argon2id

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"pimed/internal/domain/user"
)

var fast = Params{MemoryKiB: 64, Iterations: 1, Parallelism: 1, SaltLen: 16, KeyLen: 32}

func TestHashVerify(t *testing.T) {
	ctx := context.Background()
	h := New(fast)
	enc, err := h.Hash(ctx, "correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(enc, "$argon2id$v=19$m=64,t=1,p=1$") {
		t.Fatalf("unexpected encoding %q", enc)
	}
	enc2, _ := h.Hash(ctx, "correct horse")
	if enc == enc2 {
		t.Fatal("salt not random")
	}

	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{"match", "correct horse", true},
		{"mismatch", "correct horsE", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := h.Verify(ctx, tt.password, enc)
			if err != nil || ok != tt.want {
				t.Fatalf("Verify = %v, %v", ok, err)
			}
		})
	}

	// Hashes created with other params still verify.
	other := New(Params{MemoryKiB: 128, Iterations: 2, Parallelism: 2, SaltLen: 8, KeyLen: 16})
	if ok, err := other.Verify(ctx, "correct horse", enc); err != nil || !ok {
		t.Fatalf("cross-param Verify = %v, %v", ok, err)
	}
	short, err := other.Hash(ctx, "pw")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := h.Verify(ctx, "pw", short); err != nil || !ok {
		t.Fatalf("minimum salt/key Verify = %v, %v", ok, err)
	}
}

func TestVerifyMalformed(t *testing.T) {
	const (
		salt8  = "c2FsdHNhbHQ"            // 8 bytes
		salt7  = "c2FsdHNhbA"             // 7 bytes
		key16  = "AAAAAAAAAAAAAAAAAAAAAA" // 16 bytes
		key15  = "AAAAAAAAAAAAAAAAAAAA"   // 15 bytes
		params = "$argon2id$v=19$m=64,t=1,p=1$"
	)
	h := New(fast)
	// Sanity check: the well-formed variant parses (and simply mismatches).
	if ok, err := h.Verify(context.Background(), "x", params+salt8+"$"+key16); err != nil || ok {
		t.Fatalf("well-formed Verify = %v, %v", ok, err)
	}
	for _, enc := range []string{
		"",
		"plain",
		"$argon2i$v=19$m=64,t=1,p=1$" + salt8 + "$" + key16,
		"$argon2id$v=18$m=64,t=1,p=1$" + salt8 + "$" + key16,
		"$argon2id$v=19$m=x,t=1,p=1$" + salt8 + "$" + key16,
		"$argon2id$v=19$m=99999999,t=1,p=1$" + salt8 + "$" + key16,
		params + "!!$" + key16,
		params + salt8 + "$",
		params + salt7 + "$" + key16,
		params + salt8 + "$" + key15,
		params + "$" + key16,
	} {
		if _, err := h.Verify(context.Background(), "x", enc); !errors.Is(err, ErrMalformed) {
			t.Errorf("Verify(%q) err = %v", enc, err)
		}
	}
}

func TestQueueBound(t *testing.T) {
	h := New(fast)
	// Occupy every computation slot.
	for range MaxConcurrent {
		h.sem <- struct{}{}
	}

	// A cancelled caller gives up instead of waiting forever.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := h.Hash(ctx, "pw"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Hash with expired ctx err = %v", err)
	}

	// Fill the waiting queue, then the next caller fails fast.
	waitCtx, stop := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for range MaxWaiting {
		wg.Go(func() { _, _ = h.Hash(waitCtx, "pw") })
	}
	deadline := time.Now().Add(5 * time.Second)
	for h.waiting.Load() < MaxWaiting {
		if time.Now().After(deadline) {
			t.Fatalf("waiters = %d", h.waiting.Load())
		}
		time.Sleep(time.Millisecond)
	}
	if _, err := h.Verify(context.Background(), "pw", "$argon2id$v=19$m=64,t=1,p=1$c2FsdHNhbHQ$AAAAAAAAAAAAAAAAAAAAAA"); !errors.Is(err, user.ErrHasherBusy) {
		t.Fatalf("Verify with full queue err = %v", err)
	}
	stop()
	wg.Wait()
	if n := h.waiting.Load(); n != 0 {
		t.Fatalf("waiters after cancel = %d", n)
	}
}
