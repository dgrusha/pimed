package app

import (
	"context"
	"fmt"

	"pimed/internal/domain/user"
)

// NeedsSetup reports whether no user exists yet.
func (s *Service) NeedsSetup(ctx context.Context) (bool, error) {
	n, err := s.users.Count(ctx)
	if err != nil {
		return false, fmt.Errorf("count users: %w", err)
	}
	return n == 0, nil
}

// Setup creates the first (admin) account and logs it in. It fails with
// user.ErrSetupCompleted once any user exists, even under concurrent calls.
func (s *Service) Setup(ctx context.Context, username, password string) (AuthResult, error) {
	// Cheap early exit, checked first so a closed setup always reports
	// ErrSetupCompleted (never a validation error) and never pays for a hash.
	// The repository re-checks atomically; that is the real guard.
	needs, err := s.NeedsSetup(ctx)
	if err != nil {
		return AuthResult{}, err
	}
	if !needs {
		return AuthResult{}, user.ErrSetupCompleted
	}
	name, err := user.NormalizeUsername(username)
	if err != nil {
		return AuthResult{}, err
	}
	if err := user.ValidatePassword(password); err != nil {
		return AuthResult{}, err
	}
	hash, err := s.hasher.Hash(ctx, password)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hash password: %w", err)
	}
	u, err := s.users.CreateFirstAdmin(ctx, user.NewUser{
		Username:     name,
		PasswordHash: hash,
		Role:         user.RoleAdmin,
		CreatedAt:    s.nowUTC(),
	})
	if err != nil {
		return AuthResult{}, fmt.Errorf("create first admin: %w", err)
	}
	return s.startSession(ctx, u)
}
