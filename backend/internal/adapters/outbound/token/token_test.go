package token

import (
	"encoding/base64"
	"testing"
)

func TestNewToken(t *testing.T) {
	var g Generator
	seen := map[string]bool{}
	for range 100 {
		tok, err := g.NewToken()
		if err != nil {
			t.Fatal(err)
		}
		b, err := base64.RawURLEncoding.DecodeString(tok)
		if err != nil || len(b) != Size {
			t.Fatalf("bad token %q: %v", tok, err)
		}
		if seen[tok] {
			t.Fatal("duplicate token")
		}
		seen[tok] = true
	}
}
