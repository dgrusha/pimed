package app

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"pimed/internal/domain/session"
	"pimed/internal/domain/user"
)

type fakeUsers struct {
	mu    sync.Mutex
	users []user.User
}

func (f *fakeUsers) Count(context.Context) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.users), nil
}

func (f *fakeUsers) insert(u user.NewUser) (user.User, error) {
	for _, e := range f.users {
		if e.Username == u.Username {
			return user.User{}, user.ErrUsernameTaken
		}
	}
	nu := user.User{ID: int64(len(f.users) + 1), Username: u.Username, PasswordHash: u.PasswordHash, Role: u.Role, CreatedAt: u.CreatedAt}
	f.users = append(f.users, nu)
	return nu, nil
}

func (f *fakeUsers) Create(_ context.Context, u user.NewUser) (user.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.insert(u)
}

func (f *fakeUsers) CreateFirstAdmin(_ context.Context, u user.NewUser) (user.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.users) > 0 {
		return user.User{}, user.ErrSetupCompleted
	}
	return f.insert(u)
}

func (f *fakeUsers) ByID(_ context.Context, id int64) (user.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return user.User{}, user.ErrNotFound
}

func (f *fakeUsers) ByUsername(_ context.Context, name string) (user.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.users {
		if u.Username == name {
			return u, nil
		}
	}
	return user.User{}, user.ErrNotFound
}

func (f *fakeUsers) List(context.Context) ([]user.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.users), nil
}

type fakeSessions struct {
	mu sync.Mutex
	m  map[string]session.Session
}

func newFakeSessions() *fakeSessions { return &fakeSessions{m: map[string]session.Session{}} }

func (f *fakeSessions) Create(_ context.Context, s session.Session) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.m[s.TokenHash] = s
	return nil
}

func (f *fakeSessions) Get(_ context.Context, h string) (session.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.m[h]
	if !ok {
		return session.Session{}, session.ErrNotFound
	}
	return s, nil
}

func (f *fakeSessions) Delete(_ context.Context, h string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.m, h)
	return nil
}

func (f *fakeSessions) DeleteExpired(_ context.Context, now time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for h, s := range f.m {
		if s.Expired(now) {
			delete(f.m, h)
			n++
		}
	}
	return n, nil
}

func (f *fakeSessions) len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.m)
}

// fakeHasher "hashes" by prefixing; counts Verify calls.
type fakeHasher struct {
	mu       sync.Mutex
	verifies int
	err      error // returned by Verify when set
}

func (h *fakeHasher) Hash(_ context.Context, p string) (string, error) { return "hash:" + p, nil }

func (h *fakeHasher) Verify(_ context.Context, p, encoded string) (bool, error) {
	h.mu.Lock()
	h.verifies++
	err := h.err
	h.mu.Unlock()
	if err != nil {
		return false, err
	}
	if !strings.HasPrefix(encoded, "hash:") {
		return false, fmt.Errorf("malformed hash")
	}
	return encoded == "hash:"+p, nil
}

type seqTokens struct {
	mu sync.Mutex
	n  int
}

func (t *seqTokens) NewToken() (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.n++
	return fmt.Sprintf("token-%d", t.n), nil
}

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}
