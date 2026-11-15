// Package sqlite implements the domain repositories on SQLite using the
// pure-Go modernc.org/sqlite driver (no CGO).
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// Open opens (creating if needed) the database at path, applies connection
// pragmas and runs pending migrations.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	q := url.Values{}
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "synchronous(NORMAL)")
	q.Set("_txlock", "immediate") // write lock at BEGIN avoids upgrade deadlocks
	db, err := sql.Open("sqlite", path+"?"+q.Encode())
	if err != nil {
		return nil, fmt.Errorf("sqlite: open %s: %w", path, err)
	}
	db.SetMaxOpenConns(4)
	db.SetConnMaxIdleTime(5 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite: ping %s: %w", path, err)
	}
	if err := Migrate(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func isUniqueViolation(err error) bool {
	var se *sqlite.Error
	if !errors.As(err, &se) {
		return false
	}
	return se.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE || se.Code() == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY
}

func unix(t time.Time) int64 { return t.Unix() }

func fromUnix(s int64) time.Time { return time.Unix(s, 0).UTC() }
