package app

import (
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/bdd"
)

// The HTTP test harness is in apptest; feature tests live next to their code. This file tests New itself.

func TestNewApp_WithoutNATSStoreDir_RefusesToStart(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given an app configuration without a NATS store directory.",
		When:  "When the app is started.",
		Then:  "Then it refuses to start instead of sharing the machine-wide NATS default with other instances.",
	})

	// Given
	expectedErrorText := "NATSStoreDir is required"
	config := Config{
		DatabasePath:  filepath.Join(t.TempDir(), "presentation.db"),
		InviteKey:     "test-invite-key",
		SessionSecret: "test-session-secret",
	}

	// When
	_, _, err := New(t.Context(), slog.New(slog.NewTextHandler(io.Discard, nil)), config)

	// Then
	if err == nil || !strings.Contains(err.Error(), expectedErrorText) {
		t.Fatalf("expected an error containing %q, got %v", expectedErrorText, err)
	}
}
