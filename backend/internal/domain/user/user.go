// Package user holds the User entity, its validation rules and the ports
// (repository, password hasher) the application layer depends on.
package user

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Role is the authorization level of a user.
type Role string

// Supported roles.
const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

// Username and password constraints.
const (
	UsernameMinLen = 3
	UsernameMaxLen = 32
	PasswordMinLen = 8
	PasswordMaxLen = 128
)

// User is a registered account.
type User struct {
	ID           int64
	Username     string // always lowercase
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
}

// IsAdmin reports whether the user has admin privileges.
func (u User) IsAdmin() bool { return u.Role == RoleAdmin }

// NewUser is the data needed to persist a new account.
type NewUser struct {
	Username     string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
}

// ParseRole converts s into a Role, rejecting unknown values.
func ParseRole(s string) (Role, error) {
	switch r := Role(s); r {
	case RoleAdmin, RoleUser:
		return r, nil
	default:
		return "", &ValidationError{Field: "role", Message: `role must be "admin" or "user"`}
	}
}

// NormalizeUsername validates a username and returns its canonical
// (lowercase) form.
func NormalizeUsername(s string) (string, error) {
	if n := len(s); n < UsernameMinLen || n > UsernameMaxLen {
		return "", &ValidationError{Field: "username", Message: "username must be 3-32 characters long"}
	}
	for i := 0; i < len(s); i++ {
		if !isUsernameChar(s[i]) {
			return "", &ValidationError{Field: "username", Message: "username may only contain letters, digits, '_', '.' and '-'"}
		}
	}
	return strings.ToLower(s), nil
}

func isUsernameChar(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	case c == '_', c == '.', c == '-':
		return true
	}
	return false
}

// ValidatePassword checks the password length (counted in characters).
func ValidatePassword(p string) error {
	if !utf8.ValidString(p) {
		return &ValidationError{Field: "password", Message: "password must be valid UTF-8"}
	}
	if n := utf8.RuneCountInString(p); n < PasswordMinLen || n > PasswordMaxLen {
		return &ValidationError{Field: "password", Message: "password must be 8-128 characters long"}
	}
	return nil
}
