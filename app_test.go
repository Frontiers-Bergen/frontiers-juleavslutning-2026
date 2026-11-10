package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/bdd"
	root "github.com/frontiers-bergen/frontiers-juleavslutning-2026/pages/root"
)

// Test harness for HTTP-level tests: the real router from New, with a fresh SQLite file
// and JetStream store per test. Each person (attendee, presenter) is an http.Client with
// its own cookie jar, like a separate phone.

const (
	testInviteKey    = "test-invite-key"
	testPresenterKey = "test-presenter-key"
)

// Short tug of war rounds, so a whole round fits in a test.
var testTugOfWarRules = root.TugOfWarRules{Countdown: 100 * time.Millisecond, Round: 1500 * time.Millisecond, WinningLead: 5}

type testApp struct {
	t      *testing.T
	server *httptest.Server
}

func startTestApp(t *testing.T) *testApp {
	t.Helper()

	// pages/presenter.templ reads PRESENTER_KEY from the environment on every login.
	t.Setenv("PRESENTER_KEY", testPresenterKey)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	silentLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app, closeApp, err := New(ctx, silentLogger, Config{
		DatabasePath:  filepath.Join(t.TempDir(), "presentation.db"),
		NATSStoreDir:  t.TempDir(),
		InviteKey:     testInviteKey,
		SessionSecret: "test-session-secret",
		TugOfWarRules: testTugOfWarRules,
	})
	if err != nil {
		t.Fatalf("failed to start app: %v", err)
	}
	t.Cleanup(closeApp)

	server := httptest.NewServer(app)
	t.Cleanup(server.Close)

	return &testApp{t: t, server: server}
}

func (app *testApp) newAttendee() *http.Client {
	app.t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		app.t.Fatalf("failed to create cookie jar: %v", err)
	}
	return &http.Client{Jar: jar}
}

func (app *testApp) newPresenter() *http.Client {
	app.t.Helper()
	presenter := app.newAttendee()
	response, err := presenter.PostForm(app.server.URL+"/presenter/login", url.Values{"key": {testPresenterKey}})
	if err != nil {
		app.t.Fatalf("presenter login failed: %v", err)
	}
	closeBody(app.t, response)
	if response.Request.URL.Query().Get("err") != "" {
		app.t.Fatalf("presenter login was rejected: %s", response.Request.URL)
	}
	return presenter
}

// post sends a request to a route under the invite key and returns the status code.
func (app *testApp) post(client *http.Client, path string) int {
	app.t.Helper()
	response, err := client.Post(app.server.URL+"/"+testInviteKey+path, "", nil)
	if err != nil {
		app.t.Fatalf("POST %s failed: %v", path, err)
	}
	closeBody(app.t, response)
	return response.StatusCode
}

// deckPage returns the HTML of the deck as this person sees it.
func (app *testApp) deckPage(client *http.Client) string {
	app.t.Helper()
	response, err := client.Get(app.server.URL + "/" + testInviteKey + "/")
	if err != nil {
		app.t.Fatalf("GET deck failed: %v", err)
	}
	defer closeBody(app.t, response)
	body, err := io.ReadAll(response.Body)
	if err != nil {
		app.t.Fatalf("failed to read deck page: %v", err)
	}
	return string(body)
}

var slideIndexSignal = regexp.MustCompile(`slideIndex:\s*(\d+)`)

// shownSlideIndex reads which slide the page shows, from the slideIndex Datastar signal
// rendered by ActiveSlide. Using the index instead of slide text keeps tests independent of content.
func shownSlideIndex(t *testing.T, page string) int {
	t.Helper()
	match := slideIndexSignal.FindStringSubmatch(page)
	if match == nil {
		t.Fatalf("page has no slideIndex signal:\n%s", truncate(page, 500))
	}
	index, err := strconv.Atoi(match[1])
	if err != nil {
		t.Fatalf("invalid slideIndex %q: %v", match[1], err)
	}
	return index
}

// showSlideWith steps the presenter through the deck until a slide containing marker is shown, like a
// presenter clicking "next", so tests do not depend on slide order. Returns the page an attendee sees then.
// It checks an attendee's page on purpose: the presenter's page also holds previews of every slide in its
// sidebar, so it contains every slide even when another one is active.
func (app *testApp) showSlideWith(presenter *http.Client, marker string) string {
	app.t.Helper()
	viewer := app.newAttendee()
	for index := 0; ; index++ {
		if status := app.post(presenter, fmt.Sprintf("/root/api/slides/set?index=%d", index)); status != http.StatusNoContent {
			app.t.Fatalf("presenter could not move to slide %d: status %d", index, status)
		}
		page := app.deckPage(viewer)
		if strings.Contains(page, marker) {
			return page
		}
		if shownSlideIndex(app.t, page) < index {
			app.t.Fatalf("reached the end of the deck without a slide containing %q", marker)
		}
	}
}

func closeBody(t *testing.T, response *http.Response) {
	t.Helper()
	if err := response.Body.Close(); err != nil {
		t.Errorf("failed to close response body: %v", err)
	}
}

func truncate(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return strings.TrimSpace(text[:maxLength]) + "…"
}

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
		InviteKey:     testInviteKey,
		SessionSecret: "test-session-secret",
	}

	// When
	_, _, err := New(t.Context(), slog.New(slog.NewTextHandler(io.Discard, nil)), config)

	// Then
	if err == nil || !strings.Contains(err.Error(), expectedErrorText) {
		t.Fatalf("expected an error containing %q, got %v", expectedErrorText, err)
	}
}
