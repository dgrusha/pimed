package httpapi

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestLoginLimiter(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	l := NewLoginLimiter(3, 15*time.Minute, 100, clock)

	for i := range 3 {
		a, _, ok := l.Acquire("a")
		if !ok {
			t.Fatalf("blocked after %d failures", i)
		}
		a.Fail()
	}
	_, retry, ok := l.Acquire("a")
	if ok || retry != 15*time.Minute {
		t.Fatalf("Acquire = %v, %v", ok, retry)
	}
	b, _, ok := l.Acquire("b")
	if !ok {
		t.Fatal("other key blocked")
	}
	b.Release()

	// A blocked key blocks a combined attempt, which then reserves nothing.
	if _, _, ok := l.Acquire("c", "a"); ok {
		t.Fatal("combined attempt allowed")
	}
	if l.Len() != 1 {
		t.Fatalf("Len = %d, want 1", l.Len())
	}

	now = now.Add(15 * time.Minute)
	a, _, ok := l.Acquire("a")
	if !ok {
		t.Fatal("still blocked after window")
	}
	a.Release()
	a.Fail() // finishing twice is a no-op
	if l.Len() != 0 {
		t.Fatalf("Len = %d, want 0", l.Len())
	}
}

func TestLoginLimiterInFlight(t *testing.T) {
	l := NewLoginLimiter(3, time.Minute, 100, nil)
	var held []*Attempt
	for range 3 {
		a, _, ok := l.Acquire("k")
		if !ok {
			t.Fatal("refused below limit")
		}
		held = append(held, a)
	}
	if _, retry, ok := l.Acquire("k"); ok || retry != inFlightRetry {
		t.Fatalf("in-flight limit not enforced: %v, %v", ok, retry)
	}
	held[0].Release()
	a, _, ok := l.Acquire("k")
	if !ok {
		t.Fatal("released slot not reusable")
	}
	a.Fail()
	held[1].Fail()
	held[2].Fail()
	if _, retry, ok := l.Acquire("k"); ok || retry != time.Minute {
		t.Fatalf("Acquire = %v, %v", ok, retry)
	}
}

func TestLoginLimiterSucceedResetsOnlyGivenKeys(t *testing.T) {
	l := NewLoginLimiter(3, time.Minute, 100, nil)
	for range 2 {
		a, _, _ := l.Acquire("ip", "u:bob")
		a.Fail()
	}
	a, _, ok := l.Acquire("ip", "u:bob")
	if !ok {
		t.Fatal("refused below limit")
	}
	a.Succeed("u:bob")
	// ip still has 2 failures: one more is allowed, then it blocks.
	a, _, _ = l.Acquire("ip")
	a.Fail()
	if _, _, ok := l.Acquire("ip"); ok {
		t.Fatal("ip counter was reset by success")
	}
	if a, _, ok := l.Acquire("u:bob"); !ok {
		t.Fatal("username counter not reset")
	} else {
		a.Release()
	}
}

func TestLoginLimiterConcurrent(t *testing.T) {
	l := NewLoginLimiter(10, time.Minute, 100, nil)
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		allowed int
	)
	start := make(chan struct{})
	for range 100 {
		wg.Go(func() {
			<-start
			a, _, ok := l.Acquire("ip", "u:x")
			if !ok {
				return
			}
			mu.Lock()
			allowed++
			mu.Unlock()
			time.Sleep(time.Millisecond)
			a.Fail()
		})
	}
	close(start)
	wg.Wait()
	if allowed != 10 {
		t.Fatalf("allowed %d attempts, want 10", allowed)
	}
}

func TestLoginLimiterBoundedMemory(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := NewLoginLimiter(3, time.Minute, 10, func() time.Time { return now })
	fail := func(k string) {
		if a, _, ok := l.Acquire(k); ok {
			a.Fail()
		}
	}
	for i := range 50 {
		now = now.Add(time.Millisecond)
		fail(fmt.Sprintf("ip-%d", i))
	}
	if l.Len() != 10 {
		t.Fatalf("Len = %d, want 10", l.Len())
	}
	// Oldest entries were evicted; the newest survives.
	fail("ip-49")
	fail("ip-49")
	if _, _, ok := l.Acquire("ip-49"); ok {
		t.Fatal("newest entry was evicted")
	}
}
