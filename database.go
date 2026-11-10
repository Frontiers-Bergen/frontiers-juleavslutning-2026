package app

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// openDatabase opens the SQLite file at path, creating it if needed, and applies schema.sql.
func openDatabase(path string) (*sql.DB, error) {
	// Pragmas go in the connection string, not in db.Exec: database/sql keeps a pool of connections,
	// and busy_timeout and synchronous apply per connection. Set via Exec, only one pooled connection
	// would wait for locks; the others fail at once with SQLITE_BUSY and concurrent writes get lost.
	dataSourceName := "file:" + path +
		"?_pragma=journal_mode(WAL)" +
		"&_pragma=synchronous(NORMAL)" +
		"&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// A fixed pool. database/sql keeps only 2 idle connections by default, so under load (every smash makes every
	// open page query the database) it kept opening and closing connections, and opening one applies the pragmas
	// and parses the schema: half the CPU at 40 phones (go tool task loadtest). With the pool, queries beyond 32
	// wait a moment for a free connection instead.
	db.SetMaxOpenConns(32)
	db.SetMaxIdleConns(32)
	if err := db.Ping(); err != nil {
		return nil, errors.Join(fmt.Errorf("failed to ping database: %w", err), db.Close())
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		return nil, errors.Join(fmt.Errorf("failed to apply schema.sql: %w", err), db.Close())
	}

	return db, nil
}
