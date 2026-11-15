package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"pimed/internal/domain/user"
)

// UserRepository implements user.Repository.
type UserRepository struct{ db *sql.DB }

// NewUserRepository returns a UserRepository backed by db.
func NewUserRepository(db *sql.DB) *UserRepository { return &UserRepository{db: db} }

var _ user.Repository = (*UserRepository)(nil)

const userColumns = `id, username, password_hash, role, created_at`

type rowScanner interface{ Scan(dest ...any) error }

func scanUser(s rowScanner) (user.User, error) {
	var (
		u       user.User
		role    string
		created int64
	)
	if err := s.Scan(&u.ID, &u.Username, &u.PasswordHash, &role, &created); err != nil {
		return user.User{}, err
	}
	u.Role, u.CreatedAt = user.Role(role), fromUnix(created)
	return u, nil
}

// Count implements user.Repository.
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return 0, fmt.Errorf("sqlite: count users: %w", err)
	}
	return n, nil
}

// Create implements user.Repository.
func (r *UserRepository) Create(ctx context.Context, u user.NewUser) (user.User, error) {
	row := r.db.QueryRowContext(ctx,
		`INSERT INTO users (username, password_hash, role, created_at) VALUES (?, ?, ?, ?)
		 RETURNING `+userColumns,
		u.Username, u.PasswordHash, string(u.Role), unix(u.CreatedAt))
	created, err := scanUser(row)
	if isUniqueViolation(err) {
		return user.User{}, user.ErrUsernameTaken
	}
	if err != nil {
		return user.User{}, fmt.Errorf("sqlite: insert user: %w", err)
	}
	return created, nil
}

// CreateFirstAdmin implements user.Repository. The emptiness check and the
// insert are a single statement, which SQLite executes atomically under its
// write lock, so concurrent callers cannot both succeed.
func (r *UserRepository) CreateFirstAdmin(ctx context.Context, u user.NewUser) (user.User, error) {
	row := r.db.QueryRowContext(ctx,
		`INSERT INTO users (username, password_hash, role, created_at)
		 SELECT ?, ?, ?, ? WHERE NOT EXISTS (SELECT 1 FROM users)
		 RETURNING `+userColumns,
		u.Username, u.PasswordHash, string(u.Role), unix(u.CreatedAt))
	created, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return user.User{}, user.ErrSetupCompleted
	}
	if err != nil {
		return user.User{}, fmt.Errorf("sqlite: insert first admin: %w", err)
	}
	return created, nil
}

// ByID implements user.Repository.
func (r *UserRepository) ByID(ctx context.Context, id int64) (user.User, error) {
	return r.one(ctx, `SELECT `+userColumns+` FROM users WHERE id = ?`, id)
}

// ByUsername implements user.Repository.
func (r *UserRepository) ByUsername(ctx context.Context, username string) (user.User, error) {
	return r.one(ctx, `SELECT `+userColumns+` FROM users WHERE username = ?`, username)
}

func (r *UserRepository) one(ctx context.Context, query string, arg any) (user.User, error) {
	u, err := scanUser(r.db.QueryRowContext(ctx, query, arg))
	if errors.Is(err, sql.ErrNoRows) {
		return user.User{}, user.ErrNotFound
	}
	if err != nil {
		return user.User{}, fmt.Errorf("sqlite: get user: %w", err)
	}
	return u, nil
}

// List implements user.Repository.
func (r *UserRepository) List(ctx context.Context) ([]user.User, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+userColumns+` FROM users ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list users: %w", err)
	}
	defer rows.Close()
	users := []user.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("sqlite: scan user: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: list users: %w", err)
	}
	return users, nil
}
