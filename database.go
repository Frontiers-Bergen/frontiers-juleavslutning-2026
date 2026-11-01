package main

import (
	"database/sql"
	_ "embed"
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

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to apply schema.sql: %w", err)
	}

	return db, nil
}
