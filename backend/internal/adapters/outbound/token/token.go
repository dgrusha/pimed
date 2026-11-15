// Package token implements session.TokenGenerator with crypto/rand.
package token

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// Size is the number of random bytes per token (256 bits).
const Size = 32

// Generator produces base64url (unpadded) encoded random tokens.
type Generator struct{}

// NewToken returns a fresh random token.
func (Generator) NewToken() (string, error) {
	b := make([]byte, Size)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("token: read random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
