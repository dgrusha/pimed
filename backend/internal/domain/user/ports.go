package user

import "context"

// Repository persists users. Usernames passed in must already be normalized.
type Repository interface {
	// Count returns the number of users.
	Count(ctx context.Context) (int, error)
	// Create inserts a user. Returns ErrUsernameTaken on conflict.
	Create(ctx context.Context, u NewUser) (User, error)
	// CreateFirstAdmin atomically inserts u only if no users exist yet.
	// Returns ErrSetupCompleted otherwise. Must be race-safe.
	CreateFirstAdmin(ctx context.Context, u NewUser) (User, error)
	// ByID returns ErrNotFound if absent.
	ByID(ctx context.Context, id int64) (User, error)
	// ByUsername returns ErrNotFound if absent.
	ByUsername(ctx context.Context, username string) (User, error)
	// List returns all users ordered by id.
	List(ctx context.Context) ([]User, error)
}

// PasswordHasher hashes and verifies passwords. Both methods may block
// while the hasher is saturated; they give up when ctx is done and return
// ErrHasherBusy when too many callers are already waiting.
type PasswordHasher interface {
	Hash(ctx context.Context, password string) (string, error)
	// Verify reports whether password matches encoded. A malformed encoded
	// hash is an error.
	Verify(ctx context.Context, password, encoded string) (bool, error)
}
