package user

import "errors"

// Domain errors.
var (
	ErrNotFound           = errors.New("user not found")
	ErrUsernameTaken      = errors.New("username already taken")
	ErrSetupCompleted     = errors.New("setup already completed")
	ErrInvalidCredentials = errors.New("invalid username or password")
	// ErrHasherBusy is returned by a PasswordHasher that is overloaded; the
	// caller should retry later.
	ErrHasherBusy = errors.New("password hasher busy")
)

// ValidationError reports invalid user input. Message is safe to show to
// clients.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return e.Message }
