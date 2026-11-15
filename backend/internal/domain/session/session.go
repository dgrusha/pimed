// Package session holds the login Session entity and its ports.
package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

// ErrNotFound is returned when a session does not exist.
var ErrNotFound = errors.New("session not found")

// Session binds an opaque client token to a user. Only the SHA-256 hash of
// the token is stored.
type Session struct {
	TokenHash string
	UserID    int64
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Expired reports whether the session is no longer valid at now.
func (s Session) Expired(now time.Time) bool { return !now.Before(s.ExpiresAt) }

// HashToken returns the hex-encoded SHA-256 of a raw session token.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Repository persists sessions.
type Repository interface {
	Create(ctx context.Context, s Session) error
	// Get returns ErrNotFound if absent.
	Get(ctx context.Context, tokenHash string) (Session, error)
	// Delete is a no-op if the session does not exist.
	Delete(ctx context.Context, tokenHash string) error
	// DeleteExpired removes sessions expired at now and returns the count.
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

// TokenGenerator produces unguessable session tokens.
type TokenGenerator interface {
	NewToken() (string, error)
}
