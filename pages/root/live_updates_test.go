package root

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/bdd"
)

func TestLiveStream_PhoneLeft_FailedSendIsNotAnError(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given a phone that has closed its tab while its live stream was sending.",
		When:  "When the send fails.",
		Then:  "Then nothing is logged as an error: it is a normal goodbye.",
	})

	// Given
	expectedErrors := 0
	logs, logger := capturingLogger()
	ctx, phoneLeaves := context.WithCancel(t.Context())
	request := httptest.NewRequest("GET", "/dev-key/root/api/", nil).WithContext(ctx)
	phoneLeaves()

	// When
	logStreamFailure(logger, request, "failed to send page patch", context.Canceled)

	// Then
	if actualErrors := strings.Count(logs.String(), "level=ERROR"); actualErrors != expectedErrors {
		t.Fatalf("expected %d errors logged, got %d:\n%s", expectedErrors, actualErrors, logs)
	}
}

func TestLiveStream_PhoneStillThere_FailedSendIsAnError(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given a phone that is still on the page.",
		When:  "When sending to its live stream fails.",
		Then:  "Then the failure is logged as an error.",
	})

	// Given
	expectedErrors := 1
	logs, logger := capturingLogger()
	request := httptest.NewRequest("GET", "/dev-key/root/api/", nil)

	// When
	logStreamFailure(logger, request, "failed to send page patch", errors.New("broken pipe"))

	// Then
	if actualErrors := strings.Count(logs.String(), "level=ERROR"); actualErrors != expectedErrors {
		t.Fatalf("expected %d errors logged, got %d:\n%s", expectedErrors, actualErrors, logs)
	}
}

// capturingLogger logs every level, as text, into the returned buffer.
func capturingLogger() (*bytes.Buffer, *slog.Logger) {
	logs := &bytes.Buffer{}
	return logs, slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
}
