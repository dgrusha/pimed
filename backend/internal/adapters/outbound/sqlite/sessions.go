package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"pimed/internal/domain/session"
)

// SessionRepository implements session.Repository.
type SessionRepository struct{ db *sql.DB }

// NewSessionRepository returns a SessionRepository backed by db.
func NewSessionRepository(db *sql.DB) *SessionRepository { return &SessionRepository{db: db} }

var _ session.Repository = (*SessionRepository)(nil)

// Create implements session.Repository.
func (r *SessionRepository) Create(ctx context.Context, s session.Session) error {
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (token_hash, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		s.TokenHash, s.UserID, unix(s.CreatedAt), unix(s.ExpiresAt)); err != nil {
		return fmt.Errorf("sqlite: insert session: %w", err)
	}
	return nil
}

// Get implements session.Repository.
func (r *SessionRepository) Get(ctx context.Context, tokenHash string) (session.Session, error) {
	var (
		s                session.Session
		created, expires int64
	)
	err := r.db.QueryRowContext(ctx,
		`SELECT token_hash, user_id, created_at, expires_at FROM sessions WHERE token_hash = ?`, tokenHash).
		Scan(&s.TokenHash, &s.UserID, &created, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return session.Session{}, session.ErrNotFound
	}
	if err != nil {
		return session.Session{}, fmt.Errorf("sqlite: get session: %w", err)
	}
	s.CreatedAt, s.ExpiresAt = fromUnix(created), fromUnix(expires)
	return s, nil
}

// Delete implements session.Repository.
func (r *SessionRepository) Delete(ctx context.Context, tokenHash string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash); err != nil {
		return fmt.Errorf("sqlite: delete session: %w", err)
	}
	return nil
}

// DeleteExpired implements session.Repository.
func (r *SessionRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, unix(now))
	if err != nil {
		return 0, fmt.Errorf("sqlite: delete expired sessions: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("sqlite: delete expired sessions: %w", err)
	}
	return n, nil
}
