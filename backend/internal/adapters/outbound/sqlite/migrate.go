package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type migration struct {
	version int
	name    string
}

// Migrate applies all embedded migrations newer than the recorded schema
// version. Files are named NNNN_description.sql; each runs in its own
// transaction together with its schema_migrations row.
func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER PRIMARY KEY,
		applied_at INTEGER NOT NULL
	)`); err != nil {
		return fmt.Errorf("sqlite: create schema_migrations: %w", err)
	}
	var current int
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return fmt.Errorf("sqlite: read schema version: %w", err)
	}
	migs, err := listMigrations()
	if err != nil {
		return err
	}
	for _, m := range migs {
		if m.version <= current {
			continue
		}
		if err := apply(ctx, db, m); err != nil {
			return err
		}
	}
	return nil
}

func listMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("sqlite: read migrations: %w", err)
	}
	migs := make([]migration, 0, len(entries))
	seen := map[int]bool{}
	for _, e := range entries {
		prefix, _, ok := strings.Cut(e.Name(), "_")
		v, err := strconv.Atoi(prefix)
		if !ok || err != nil || v <= 0 {
			return nil, fmt.Errorf("sqlite: bad migration file name %q", e.Name())
		}
		if seen[v] {
			return nil, fmt.Errorf("sqlite: duplicate migration version %d", v)
		}
		seen[v] = true
		migs = append(migs, migration{version: v, name: e.Name()})
	}
	sort.Slice(migs, func(i, j int) bool { return migs[i].version < migs[j].version })
	return migs, nil
}

func apply(ctx context.Context, db *sql.DB, m migration) error {
	body, err := migrationFS.ReadFile("migrations/" + m.name)
	if err != nil {
		return fmt.Errorf("sqlite: read migration %s: %w", m.name, err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: begin migration %s: %w", m.name, err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, string(body)); err != nil {
		return fmt.Errorf("sqlite: apply migration %s: %w", m.name, err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
		m.version, time.Now().Unix()); err != nil {
		return fmt.Errorf("sqlite: record migration %s: %w", m.name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite: commit migration %s: %w", m.name, err)
	}
	return nil
}
