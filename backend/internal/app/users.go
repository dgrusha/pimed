package app

import (
	"context"
	"fmt"

	"pimed/internal/domain/user"
)

// CreateUserInput is the input of CreateUser. An empty Role means user.RoleUser.
type CreateUserInput struct {
	Username string
	Password string
	Role     string
}

// CreateUser creates an account. Only admins may call it.
func (s *Service) CreateUser(ctx context.Context, actor user.User, in CreateUserInput) (user.User, error) {
	if !actor.IsAdmin() {
		return user.User{}, ErrForbidden
	}
	name, err := user.NormalizeUsername(in.Username)
	if err != nil {
		return user.User{}, err
	}
	if err := user.ValidatePassword(in.Password); err != nil {
		return user.User{}, err
	}
	role := user.RoleUser
	if in.Role != "" {
		if role, err = user.ParseRole(in.Role); err != nil {
			return user.User{}, err
		}
	}
	hash, err := s.hasher.Hash(ctx, in.Password)
	if err != nil {
		return user.User{}, fmt.Errorf("hash password: %w", err)
	}
	u, err := s.users.Create(ctx, user.NewUser{
		Username:     name,
		PasswordHash: hash,
		Role:         role,
		CreatedAt:    s.nowUTC(),
	})
	if err != nil {
		return user.User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// ListUsers returns all users ordered by id. Only admins may call it.
func (s *Service) ListUsers(ctx context.Context, actor user.User) ([]user.User, error) {
	if !actor.IsAdmin() {
		return nil, ErrForbidden
	}
	users, err := s.users.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}
