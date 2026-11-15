package app

import (
	"context"
	"errors"
	"fmt"

	"pimed/internal/domain/session"
	"pimed/internal/domain/user"
)

// maxTokenLen bounds tokens accepted from clients (ours are 43 chars).
const maxTokenLen = 128

// Login verifies credentials and starts a session. Unknown users and wrong
// passwords both yield user.ErrInvalidCredentials after a hash verification.
func (s *Service) Login(ctx context.Context, username, password string) (AuthResult, error) {
	name, nameErr := user.NormalizeUsername(username)
	var u user.User
	encoded := s.dummyHash
	if nameErr == nil {
		found, err := s.users.ByUsername(ctx, name)
		switch {
		case err == nil:
			u, encoded = found, found.PasswordHash
		case !errors.Is(err, user.ErrNotFound):
			return AuthResult{}, fmt.Errorf("find user: %w", err)
		}
	}
	ok, err := s.hasher.Verify(ctx, password, encoded)
	if err != nil {
		return AuthResult{}, fmt.Errorf("verify password: %w", err)
	}
	if !ok || u.ID == 0 {
		return AuthResult{}, user.ErrInvalidCredentials
	}
	return s.startSession(ctx, u)
}

// Logout deletes the session identified by token, if any.
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" || len(token) > maxTokenLen {
		return nil
	}
	if err := s.sessions.Delete(ctx, session.HashToken(token)); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// CurrentUser resolves a session token to its user. Missing, unknown or
// expired sessions yield ErrUnauthenticated; expired ones are pruned.
func (s *Service) CurrentUser(ctx context.Context, token string) (user.User, error) {
	if token == "" || len(token) > maxTokenLen {
		return user.User{}, ErrUnauthenticated
	}
	hash := session.HashToken(token)
	sess, err := s.sessions.Get(ctx, hash)
	if errors.Is(err, session.ErrNotFound) {
		return user.User{}, ErrUnauthenticated
	}
	if err != nil {
		return user.User{}, fmt.Errorf("get session: %w", err)
	}
	if sess.Expired(s.now()) {
		if err := s.sessions.Delete(ctx, hash); err != nil {
			return user.User{}, fmt.Errorf("delete expired session: %w", err)
		}
		return user.User{}, ErrUnauthenticated
	}
	u, err := s.users.ByID(ctx, sess.UserID)
	if errors.Is(err, user.ErrNotFound) {
		return user.User{}, ErrUnauthenticated
	}
	if err != nil {
		return user.User{}, fmt.Errorf("get session user: %w", err)
	}
	return u, nil
}

// PruneSessions deletes all expired sessions.
func (s *Service) PruneSessions(ctx context.Context) (int64, error) {
	n, err := s.sessions.DeleteExpired(ctx, s.now())
	if err != nil {
		return 0, fmt.Errorf("prune sessions: %w", err)
	}
	return n, nil
}
