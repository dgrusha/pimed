// Package app contains the use cases. It depends only on domain ports.
package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"pimed/internal/domain/session"
	"pimed/internal/domain/user"
)

// Application errors (in addition to the domain errors).
var (
	ErrUnauthenticated = errors.New("authentication required")
	ErrForbidden       = errors.New("admin privileges required")
)

// dummyPassword is hashed once at startup so that logins for unknown users
// still pay the cost of one verification (blunts user enumeration by timing).
const dummyPassword = "pimed-dummy-password-for-timing"

// Deps are the collaborators of Service.
type Deps struct {
	Users      user.Repository
	Sessions   session.Repository
	Hasher     user.PasswordHasher
	Tokens     session.TokenGenerator
	Now        func() time.Time // defaults to time.Now
	SessionTTL time.Duration
}

// Service implements the authentication and user management use cases.
type Service struct {
	users      user.Repository
	sessions   session.Repository
	hasher     user.PasswordHasher
	tokens     session.TokenGenerator
	now        func() time.Time
	sessionTTL time.Duration
	dummyHash  string
}

// New builds a Service. It hashes a dummy password once, so it may take as
// long as one password hash.
func New(d Deps) (*Service, error) {
	if d.Users == nil || d.Sessions == nil || d.Hasher == nil || d.Tokens == nil {
		return nil, errors.New("app: missing dependency")
	}
	if d.SessionTTL <= 0 {
		return nil, errors.New("app: session TTL must be positive")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	dummy, err := d.Hasher.Hash(context.Background(), dummyPassword)
	if err != nil {
		return nil, fmt.Errorf("app: hash dummy password: %w", err)
	}
	return &Service{
		users:      d.Users,
		sessions:   d.Sessions,
		hasher:     d.Hasher,
		tokens:     d.Tokens,
		now:        d.Now,
		sessionTTL: d.SessionTTL,
		dummyHash:  dummy,
	}, nil
}

// AuthResult is returned by use cases that log a user in.
type AuthResult struct {
	User      user.User
	Token     string // raw session token; hand to the client, never log
	ExpiresAt time.Time
}

func (s *Service) nowUTC() time.Time { return s.now().UTC().Truncate(time.Second) }

// startSession creates a new session for u.
func (s *Service) startSession(ctx context.Context, u user.User) (AuthResult, error) {
	token, err := s.tokens.NewToken()
	if err != nil {
		return AuthResult{}, fmt.Errorf("generate session token: %w", err)
	}
	now := s.nowUTC()
	sess := session.Session{
		TokenHash: session.HashToken(token),
		UserID:    u.ID,
		CreatedAt: now,
		ExpiresAt: now.Add(s.sessionTTL),
	}
	if err := s.sessions.Create(ctx, sess); err != nil {
		return AuthResult{}, fmt.Errorf("create session: %w", err)
	}
	return AuthResult{User: u, Token: token, ExpiresAt: sess.ExpiresAt}, nil
}
