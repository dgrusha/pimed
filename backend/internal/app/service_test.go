package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"pimed/internal/domain/user"
)

type env struct {
	svc      *Service
	users    *fakeUsers
	sessions *fakeSessions
	hasher   *fakeHasher
	clock    *fakeClock
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		users:    &fakeUsers{},
		sessions: newFakeSessions(),
		hasher:   &fakeHasher{},
		clock:    &fakeClock{t: time.Date(2026, 11, 14, 19, 40, 0, 0, time.UTC)},
	}
	svc, err := New(Deps{
		Users: e.users, Sessions: e.sessions, Hasher: e.hasher, Tokens: &seqTokens{},
		Now: e.clock.Now, SessionTTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	e.svc = svc
	return e
}

func (e *env) setupAdmin(t *testing.T) AuthResult {
	t.Helper()
	res, err := e.svc.Setup(context.Background(), "Admin", "password1")
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func isValidation(err error) bool {
	var ve *user.ValidationError
	return errors.As(err, &ve)
}

func TestSetup(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)

	needs, err := e.svc.NeedsSetup(ctx)
	if err != nil || !needs {
		t.Fatalf("NeedsSetup = %v, %v", needs, err)
	}

	tests := []struct {
		name, username, password string
		check                    func(error) bool
	}{
		{"bad username", "a", "password1", isValidation},
		{"bad password", "admin", "short", isValidation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := e.svc.Setup(ctx, tt.username, tt.password); !tt.check(err) {
				t.Fatalf("unexpected err %v", err)
			}
		})
	}

	res := e.setupAdmin(t)
	if res.User.Username != "admin" || res.User.Role != user.RoleAdmin || res.Token == "" {
		t.Fatalf("unexpected result %+v", res)
	}
	if !res.ExpiresAt.Equal(e.clock.Now().Add(time.Hour)) {
		t.Fatalf("ExpiresAt = %v", res.ExpiresAt)
	}
	if needs, _ := e.svc.NeedsSetup(ctx); needs {
		t.Fatal("NeedsSetup still true")
	}
	if _, err := e.svc.Setup(ctx, "other", "password1"); !errors.Is(err, user.ErrSetupCompleted) {
		t.Fatalf("second setup err = %v", err)
	}
	// A closed setup reports ErrSetupCompleted even for invalid input.
	if _, err := e.svc.Setup(ctx, "x", "short"); !errors.Is(err, user.ErrSetupCompleted) {
		t.Fatalf("invalid input after setup err = %v", err)
	}
}

func TestSetupConcurrent(t *testing.T) {
	e := newEnv(t)
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for range 20 {
		wg.Go(func() {
			if _, err := e.svc.Setup(context.Background(), "admin", "password1"); err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if ok != 1 || len(e.users.users) != 1 {
		t.Fatalf("successes = %d, users = %d", ok, len(e.users.users))
	}
}

func TestLogin(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	e.setupAdmin(t)

	tests := []struct {
		name, username, password string
		wantErr                  error
	}{
		{"ok", "admin", "password1", nil},
		{"case-insensitive username", "ADMIN", "password1", nil},
		{"wrong password", "admin", "nope-nope", user.ErrInvalidCredentials},
		{"unknown user", "ghost", "password1", user.ErrInvalidCredentials},
		{"malformed username", "!", "password1", user.ErrInvalidCredentials},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := e.hasher.verifies
			res, err := e.svc.Login(ctx, tt.username, tt.password)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if e.hasher.verifies != before+1 {
				t.Fatalf("expected exactly one hash verification, got %d", e.hasher.verifies-before)
			}
			if tt.wantErr == nil && res.User.Username != "admin" {
				t.Fatalf("user = %+v", res.User)
			}
		})
	}
}

func TestLoginHasherBusy(t *testing.T) {
	e := newEnv(t)
	e.setupAdmin(t)
	e.hasher.err = user.ErrHasherBusy
	if _, err := e.svc.Login(context.Background(), "admin", "password1"); !errors.Is(err, user.ErrHasherBusy) {
		t.Fatalf("err = %v", err)
	}
}

func TestCurrentUserAndLogout(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	res := e.setupAdmin(t)

	u, err := e.svc.CurrentUser(ctx, res.Token)
	if err != nil || u.ID != res.User.ID {
		t.Fatalf("CurrentUser = %+v, %v", u, err)
	}
	for _, tok := range []string{"", "bogus", string(make([]byte, 500))} {
		if _, err := e.svc.CurrentUser(ctx, tok); !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("token %q: err = %v", tok, err)
		}
	}
	if err := e.svc.Logout(ctx, res.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.CurrentUser(ctx, res.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("after logout err = %v", err)
	}
}

func TestSessionExpiry(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	res := e.setupAdmin(t)
	if _, err := e.svc.Login(ctx, "admin", "password1"); err != nil {
		t.Fatal(err)
	}

	e.clock.Advance(time.Hour)
	if _, err := e.svc.CurrentUser(ctx, res.Token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expired session err = %v", err)
	}
	if e.sessions.len() != 1 {
		t.Fatalf("expired session not deleted on lookup, have %d", e.sessions.len())
	}
	n, err := e.svc.PruneSessions(ctx)
	if err != nil || n != 1 || e.sessions.len() != 0 {
		t.Fatalf("PruneSessions = %d, %v (left %d)", n, err, e.sessions.len())
	}
}

func TestCreateAndListUsers(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	admin := e.setupAdmin(t).User
	plain := user.User{ID: 99, Username: "bob", Role: user.RoleUser}

	tests := []struct {
		name    string
		actor   user.User
		in      CreateUserInput
		check   func(error) bool
		wantRol user.Role
	}{
		{"default role", admin, CreateUserInput{"Alice", "password1", ""}, nil, user.RoleUser},
		{"explicit admin", admin, CreateUserInput{"carol", "password1", "admin"}, nil, user.RoleAdmin},
		{"duplicate case-insensitive", admin, CreateUserInput{"ALICE", "password1", ""},
			func(err error) bool { return errors.Is(err, user.ErrUsernameTaken) }, ""},
		{"bad role", admin, CreateUserInput{"dave", "password1", "root"}, isValidation, ""},
		{"bad password", admin, CreateUserInput{"dave", "short", ""}, isValidation, ""},
		{"non-admin", plain, CreateUserInput{"eve", "password1", ""},
			func(err error) bool { return errors.Is(err, ErrForbidden) }, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := e.svc.CreateUser(ctx, tt.actor, tt.in)
			if tt.check != nil {
				if !tt.check(err) {
					t.Fatalf("unexpected err %v", err)
				}
				return
			}
			if err != nil || u.Role != tt.wantRol {
				t.Fatalf("CreateUser = %+v, %v", u, err)
			}
		})
	}

	list, err := e.svc.ListUsers(ctx, admin)
	if err != nil || len(list) != 3 || list[1].Username != "alice" {
		t.Fatalf("ListUsers = %+v, %v", list, err)
	}
	if _, err := e.svc.ListUsers(ctx, plain); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-admin ListUsers err = %v", err)
	}
}
