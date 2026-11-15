// Package argon2id implements user.PasswordHasher with argon2id and PHC-style
// encoded hashes: $argon2id$v=19$m=<KiB>,t=<iter>,p=<threads>$<salt>$<hash>.
package argon2id

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"golang.org/x/crypto/argon2"

	"pimed/internal/domain/user"
)

// Params are the argon2id cost parameters.
type Params struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	SaltLen     uint32
	KeyLen      uint32
}

// DefaultParams follow RFC 9106's second recommended option (64 MiB, t=3,
// p=4): roughly 100-250 ms on a Raspberry Pi 5 and affordable on 2 GB RAM
// because concurrent hashing is bounded (see MaxConcurrent).
func DefaultParams() Params {
	return Params{MemoryKiB: 64 * 1024, Iterations: 3, Parallelism: 4, SaltLen: 16, KeyLen: 32}
}

// MaxConcurrent bounds simultaneous hash computations so memory use stays at
// most MaxConcurrent * MemoryKiB.
const MaxConcurrent = 2

// MaxWaiting bounds how many callers may queue for a computation slot; any
// further caller fails fast with user.ErrHasherBusy instead of piling up.
const MaxWaiting = 8

// Bounds accepted when decoding stored hashes.
const (
	maxMemoryKiB  = 1024 * 1024
	maxIterations = 32
	minSaltLen    = 8
	minKeyLen     = 16
)

// ErrMalformed is returned by Verify for unparsable encoded hashes.
var ErrMalformed = errors.New("argon2id: malformed hash")

// Hasher hashes passwords with argon2id.
type Hasher struct {
	p       Params
	sem     chan struct{}
	waiting atomic.Int32
}

// New returns a Hasher using p.
func New(p Params) *Hasher {
	return &Hasher{p: p, sem: make(chan struct{}, MaxConcurrent)}
}

// acquire takes a computation slot, waiting at most until ctx is done. It
// fails fast with user.ErrHasherBusy when MaxWaiting callers already wait.
func (h *Hasher) acquire(ctx context.Context) error {
	select {
	case h.sem <- struct{}{}:
		return nil
	default:
	}
	if h.waiting.Add(1) > MaxWaiting {
		h.waiting.Add(-1)
		return user.ErrHasherBusy
	}
	defer h.waiting.Add(-1)
	select {
	case h.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *Hasher) key(ctx context.Context, password string, salt []byte, p Params) ([]byte, error) {
	if err := h.acquire(ctx); err != nil {
		return nil, err
	}
	defer func() { <-h.sem }()
	return argon2.IDKey([]byte(password), salt, p.Iterations, p.MemoryKiB, p.Parallelism, p.KeyLen), nil
}

// Hash returns the encoded argon2id hash of password with a random salt.
func (h *Hasher) Hash(ctx context.Context, password string) (string, error) {
	salt := make([]byte, h.p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("argon2id: read salt: %w", err)
	}
	key, err := h.key(ctx, password, salt, h.p)
	if err != nil {
		return "", err
	}
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.p.MemoryKiB, h.p.Iterations, h.p.Parallelism,
		b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// Verify checks password against an encoded hash in constant time. The
// parameters embedded in encoded are used, so changing Params does not break
// existing hashes.
func (h *Hasher) Verify(ctx context.Context, password, encoded string) (bool, error) {
	p, salt, want, err := decode(encoded)
	if err != nil {
		return false, err
	}
	got, err := h.key(ctx, password, salt, p)
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

func decode(encoded string) (Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return Params{}, nil, nil, ErrMalformed
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return Params{}, nil, nil, fmt.Errorf("%w: unsupported version", ErrMalformed)
	}
	var p Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.MemoryKiB, &p.Iterations, &p.Parallelism); err != nil {
		return Params{}, nil, nil, fmt.Errorf("%w: params: %w", ErrMalformed, err)
	}
	if p.MemoryKiB == 0 || p.MemoryKiB > maxMemoryKiB || p.Iterations == 0 || p.Iterations > maxIterations || p.Parallelism == 0 {
		return Params{}, nil, nil, fmt.Errorf("%w: params out of range", ErrMalformed)
	}
	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil || len(salt) < minSaltLen {
		return Params{}, nil, nil, fmt.Errorf("%w: salt", ErrMalformed)
	}
	key, err := b64.DecodeString(parts[5])
	if err != nil || len(key) < minKeyLen {
		return Params{}, nil, nil, fmt.Errorf("%w: key", ErrMalformed)
	}
	p.SaltLen, p.KeyLen = uint32(len(salt)), uint32(len(key))
	return p, salt, key, nil
}
