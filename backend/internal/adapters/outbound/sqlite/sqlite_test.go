package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pimed/internal/domain/session"
	"pimed/internal/domain/user"
)

func openTest(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

var t0 = time.Date(2026, 11, 14, 19, 40, 0, 0, time.UTC)

func newUser(name string, role user.Role) user.NewUser {
	return user.NewUser{Username: name, PasswordHash: "h", Role: role, CreatedAt: t0}
}

func TestMigrateIdempotentAndPragmas(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "m.db")
	for range 2 {
		db, err := Open(ctx, path)
		if err != nil {
			t.Fatal(err)
		}
		var version int
		if err := db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil || version != 1 {
			t.Fatalf("version = %d, %v", version, err)
		}
		var mode string
		var fk int
		db.QueryRow(`PRAGMA journal_mode`).Scan(&mode)
		db.QueryRow(`PRAGMA foreign_keys`).Scan(&fk)
		if mode != "wal" || fk != 1 {
			t.Fatalf("journal_mode=%q foreign_keys=%d", mode, fk)
		}
		db.Close()
	}
}

func TestUserRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewUserRepository(openTest(t))

	if n, err := repo.Count(ctx); err != nil || n != 0 {
		t.Fatalf("Count = %d, %v", n, err)
	}
	admin, err := repo.CreateFirstAdmin(ctx, newUser("admin", user.RoleAdmin))
	if err != nil || admin.ID == 0 || !admin.CreatedAt.Equal(t0) || admin.Role != user.RoleAdmin {
		t.Fatalf("CreateFirstAdmin = %+v, %v", admin, err)
	}
	if _, err := repo.CreateFirstAdmin(ctx, newUser("admin2", user.RoleAdmin)); !errors.Is(err, user.ErrSetupCompleted) {
		t.Fatalf("second CreateFirstAdmin err = %v", err)
	}
	bob, err := repo.Create(ctx, newUser("bob", user.RoleUser))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Create(ctx, newUser("bob", user.RoleUser)); !errors.Is(err, user.ErrUsernameTaken) {
		t.Fatalf("duplicate err = %v", err)
	}
	if _, err := repo.Create(ctx, newUser("Bob", user.RoleUser)); err == nil {
		t.Fatal("non-lowercase username accepted by schema")
	}

	tests := []struct {
		name    string
		get     func() (user.User, error)
		wantID  int64
		wantErr error
	}{
		{"by id", func() (user.User, error) { return repo.ByID(ctx, bob.ID) }, bob.ID, nil},
		{"by name", func() (user.User, error) { return repo.ByUsername(ctx, "admin") }, admin.ID, nil},
		{"missing id", func() (user.User, error) { return repo.ByID(ctx, 999) }, 0, user.ErrNotFound},
		{"missing name", func() (user.User, error) { return repo.ByUsername(ctx, "nobody") }, 0, user.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := tt.get()
			if !errors.Is(err, tt.wantErr) || u.ID != tt.wantID {
				t.Fatalf("got %+v, %v", u, err)
			}
		})
	}

	list, err := repo.List(ctx)
	if err != nil || len(list) != 2 || list[0].ID != admin.ID || list[1].ID != bob.ID {
		t.Fatalf("List = %+v, %v", list, err)
	}
}

func TestCreateFirstAdminRace(t *testing.T) {
	ctx := context.Background()
	repo := NewUserRepository(openTest(t))

	var wg sync.WaitGroup
	var ok, completed atomic.Int32
	start := make(chan struct{})
	for i := range 20 {
		wg.Go(func() {
			<-start
			_, err := repo.CreateFirstAdmin(ctx, newUser(fmt.Sprintf("admin%d", i), user.RoleAdmin))
			switch {
			case err == nil:
				ok.Add(1)
			case errors.Is(err, user.ErrSetupCompleted):
				completed.Add(1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
	close(start)
	wg.Wait()
	if ok.Load() != 1 || completed.Load() != 19 {
		t.Fatalf("ok=%d completed=%d", ok.Load(), completed.Load())
	}
	if n, _ := repo.Count(ctx); n != 1 {
		t.Fatalf("users = %d", n)
	}
}

func TestSessionRepository(t *testing.T) {
	ctx := context.Background()
	db := openTest(t)
	users := NewUserRepository(db)
	repo := NewSessionRepository(db)

	u, err := users.Create(ctx, newUser("alice", user.RoleUser))
	if err != nil {
		t.Fatal(err)
	}
	live := session.Session{TokenHash: "live", UserID: u.ID, CreatedAt: t0, ExpiresAt: t0.Add(time.Hour)}
	dead := session.Session{TokenHash: "dead", UserID: u.ID, CreatedAt: t0, ExpiresAt: t0.Add(time.Minute)}
	for _, s := range []session.Session{live, dead} {
		if err := repo.Create(ctx, s); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.Create(ctx, session.Session{TokenHash: "orphan", UserID: 999, CreatedAt: t0, ExpiresAt: t0}); err == nil {
		t.Fatal("foreign key not enforced")
	}

	got, err := repo.Get(ctx, "live")
	if err != nil || got != live {
		t.Fatalf("Get = %+v, %v", got, err)
	}
	if _, err := repo.Get(ctx, "nope"); !errors.Is(err, session.ErrNotFound) {
		t.Fatalf("Get missing err = %v", err)
	}

	n, err := repo.DeleteExpired(ctx, t0.Add(30*time.Minute))
	if err != nil || n != 1 {
		t.Fatalf("DeleteExpired = %d, %v", n, err)
	}
	if err := repo.Delete(ctx, "live"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, "live"); err != nil {
		t.Fatalf("second delete: %v", err)
	}
	if _, err := repo.Get(ctx, "live"); !errors.Is(err, session.ErrNotFound) {
		t.Fatalf("after delete err = %v", err)
	}
}
