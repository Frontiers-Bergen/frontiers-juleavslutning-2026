package main

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/bdd"
)

func TestOpenDatabase_FreshFile_CreatesTablesAndFirstSlide(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given no database file exists yet, as on a fresh clone.",
		When:  "When the database is opened.",
		Then:  "Then all tables exist and the presenter starts on the first slide.",
	})

	// Given
	expectedTables := []string{"emoji_counter", "poll_votes", "slide_state"}
	expectedSlideIndex := 0
	path := filepath.Join(t.TempDir(), "presentation.db")

	// When
	db := openTestDatabase(t, path)

	// Then
	actualTables := tableNames(t, db)
	if !slices.Equal(expectedTables, actualTables) {
		t.Fatalf("tables mismatch\nexpected: %v\nactual:   %v", expectedTables, actualTables)
	}
	actualSlideIndex := currentSlideIndex(t, db)
	if actualSlideIndex != expectedSlideIndex {
		t.Fatalf("expected slide index %d, got %d", expectedSlideIndex, actualSlideIndex)
	}
}

func TestOpenDatabase_AfterRestart_KeepsExistingData(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given the presenter is on slide 7 and someone has voted.",
		When:  "When the server restarts and opens the same database file.",
		Then:  "Then the schema is applied again without errors and the slide and vote are kept.",
	})

	// Given
	expectedSlideIndex := 7
	expectedVoteCount := 1
	path := filepath.Join(t.TempDir(), "presentation.db")
	firstRun := openTestDatabase(t, path)
	execOrFail(t, firstRun, `UPDATE slide_state SET current_index = ? WHERE id = 1`, expectedSlideIndex)
	execOrFail(t, firstRun,
		`INSERT INTO poll_votes (invite_key, poll_key, session_id, option_key, voted_at_unix) VALUES ('local', 'poll', 'session', 'option', 0)`)
	closeOrFail(t, firstRun)

	// When
	secondRun := openTestDatabase(t, path)

	// Then
	actualSlideIndex := currentSlideIndex(t, secondRun)
	if actualSlideIndex != expectedSlideIndex {
		t.Fatalf("expected slide index %d, got %d", expectedSlideIndex, actualSlideIndex)
	}
	var actualVoteCount int
	if err := secondRun.QueryRow(`SELECT COUNT(*) FROM poll_votes`).Scan(&actualVoteCount); err != nil {
		t.Fatalf("failed to count votes: %v", err)
	}
	if actualVoteCount != expectedVoteCount {
		t.Fatalf("expected %d vote, got %d", expectedVoteCount, actualVoteCount)
	}
}

func TestOpenDatabase_ConcurrentWrites_AllSucceed(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given the database is open and many people act at the same moment.",
		When:  "When 50 writes hit the database concurrently.",
		Then:  "Then every write succeeds; none is lost to a locked database.",
	})

	// Given
	expectedVoteCount := 50
	db := openTestDatabase(t, filepath.Join(t.TempDir(), "presentation.db"))

	// When
	errs := make(chan error, expectedVoteCount)
	var writers sync.WaitGroup
	for i := range expectedVoteCount {
		writers.Go(func() {
			_, err := db.Exec(
				`INSERT INTO poll_votes (invite_key, poll_key, session_id, option_key, voted_at_unix) VALUES ('local', 'poll', ?, 'option', 0)`,
				fmt.Sprintf("session-%d", i),
			)
			errs <- err
		})
	}
	writers.Wait()
	close(errs)

	// Then
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent write failed: %v", err)
		}
	}
	var actualVoteCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM poll_votes`).Scan(&actualVoteCount); err != nil {
		t.Fatalf("failed to count votes: %v", err)
	}
	if actualVoteCount != expectedVoteCount {
		t.Fatalf("expected %d votes, got %d", expectedVoteCount, actualVoteCount)
	}
}

func openTestDatabase(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := openDatabase(path)
	if err != nil {
		t.Fatalf("openDatabase(%q) failed: %v", path, err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("failed to close database: %v", err)
		}
	})
	return db
}

func closeOrFail(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close database: %v", err)
	}
}

func execOrFail(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("failed to exec %q: %v", query, err)
	}
}

func tableNames(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name`)
	if err != nil {
		t.Fatalf("failed to list tables: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Errorf("failed to close rows: %v", err)
		}
	}()

	names := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("failed to scan table name: %v", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed to read table names: %v", err)
	}
	return names
}

func currentSlideIndex(t *testing.T, db *sql.DB) int {
	t.Helper()
	var index int
	if err := db.QueryRow(`SELECT current_index FROM slide_state WHERE id = 1`).Scan(&index); err != nil {
		t.Fatalf("failed to read current slide index: %v", err)
	}
	return index
}
