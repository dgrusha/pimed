package user

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeUsername(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"alice", "alice", false},
		{"Alice", "alice", false},
		{"a.b-c_d9", "a.b-c_d9", false},
		{"abc", "abc", false},
		{strings.Repeat("x", 32), strings.Repeat("x", 32), false},
		{"ab", "", true},
		{strings.Repeat("x", 33), "", true},
		{"", "", true},
		{"al ice", "", true},
		{"alice!", "", true},
		{"ålice", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := NormalizeUsername(tt.in)
			if tt.wantErr {
				var ve *ValidationError
				if !errors.As(err, &ve) {
					t.Fatalf("want ValidationError, got %v", err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("got (%q, %v), want %q", got, err, tt.want)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"min", "12345678", false},
		{"max", strings.Repeat("p", 128), false},
		{"multibyte counted as chars", strings.Repeat("é", 8), false},
		{"too short", "1234567", true},
		{"too long", strings.Repeat("p", 129), true},
		{"empty", "", true},
		{"invalid utf8", "\xff\xfe12345678", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseRole(t *testing.T) {
	tests := []struct {
		in      string
		want    Role
		wantErr bool
	}{
		{"admin", RoleAdmin, false},
		{"user", RoleUser, false},
		{"", "", true},
		{"Admin", "", true},
		{"root", "", true},
	}
	for _, tt := range tests {
		got, err := ParseRole(tt.in)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("ParseRole(%q) = (%q, %v)", tt.in, got, err)
		}
	}
}
