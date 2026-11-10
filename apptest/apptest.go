// Package apptest is the harness for HTTP-level tests: the real router from app.New, with a fresh SQLite file and
// JetStream store per test. Each person (attendee, presenter) is an http.Client with its own cookie jar, like a
// separate phone. Feature tests live next to their code (pages/root/tug_of_war_test.go next to tug_of_war.templ)
// and start the app through Start; their own helpers (smash, shake, vote) stay in their test files.
package apptest

import (
	"bufio"
	"context"
	"database/sql"
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

	app "github.com/frontiers-bergen/frontiers-juleavslutning-2026"
	root "github.com/frontiers-bergen/frontiers-juleavslutning-2026/pages/root"
)

const (
	InviteKey    = "test-invite-key"
	PresenterKey = "test-presenter-key"
)

// TugOfWarRules are short tug of war rounds, so a whole round fits in a test.
var TugOfWarRules = root.TugOfWarRules{Countdown: 100 * time.Millisecond, Round: 1500 * time.Millisecond, WinningLead: 5}

// App is one running copy of the app, for one test.
type App struct {
	T            *testing.T
	server       *httptest.Server
	databasePath string
}

// Start runs the app with a fresh database and NATS store; everything is stopped when the test ends.
func Start(t *testing.T) *App {
	t.Helper()

	// pages/presenter.templ reads PRESENTER_KEY from the environment on every login.
	t.Setenv("PRESENTER_KEY", PresenterKey)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	silentLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	databasePath := filepath.Join(t.TempDir(), "presentation.db")
	handler, closeApp, err := app.New(ctx, silentLogger, app.Config{
		DatabasePath:  databasePath,
		NATSStoreDir:  t.TempDir(),
		InviteKey:     InviteKey,
		SessionSecret: "test-session-secret",
		TugOfWarRules: TugOfWarRules,
	})
	if err != nil {
		t.Fatalf("failed to start app: %v", err)
	}
	t.Cleanup(closeApp)

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &App{T: t, server: server, databasePath: databasePath}
}

// ExecSQL changes the app's database directly, for states the app's own routes cannot reach (a deck that got
// shorter while the database kept the old slide index). Prefer the routes: they are what people use.
func (a *App) ExecSQL(query string, args ...any) {
	a.T.Helper()
	db, err := sql.Open("sqlite", a.databasePath)
	if err != nil {
		a.T.Fatalf("failed to open the app's database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			a.T.Errorf("failed to close the app's database: %v", err)
		}
	}()
	if _, err := db.Exec(query, args...); err != nil {
		a.T.Fatalf("failed to run %q: %v", query, err)
	}
}

// URL is the address of the running app, without a trailing slash.
func (a *App) URL() string {
	return a.server.URL
}

func (a *App) NewAttendee() *http.Client {
	a.T.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		a.T.Fatalf("failed to create cookie jar: %v", err)
	}
	return &http.Client{Jar: jar}
}

func (a *App) NewPresenter() *http.Client {
	a.T.Helper()
	presenter := a.NewAttendee()
	response, err := presenter.PostForm(a.server.URL+"/presenter/login", url.Values{"key": {PresenterKey}})
	if err != nil {
		a.T.Fatalf("presenter login failed: %v", err)
	}
	CloseBody(a.T, response)
	if response.Request.URL.Query().Get("err") != "" {
		a.T.Fatalf("presenter login was rejected: %s", response.Request.URL)
	}
	return presenter
}

// Post sends a request to a route under the invite key and returns the status code.
func (a *App) Post(client *http.Client, path string) int {
	a.T.Helper()
	response, err := client.Post(a.server.URL+"/"+InviteKey+path, "", nil)
	if err != nil {
		a.T.Fatalf("POST %s failed: %v", path, err)
	}
	CloseBody(a.T, response)
	return response.StatusCode
}

// Get returns the status, content type and body of a GET request; path starts at the root, not the invite key.
func (a *App) Get(client *http.Client, path string) (status int, contentType string, body string) {
	a.T.Helper()
	response, err := client.Get(a.server.URL + path)
	if err != nil {
		a.T.Fatalf("GET %s failed: %v", path, err)
	}
	defer CloseBody(a.T, response)
	content, err := io.ReadAll(response.Body)
	if err != nil {
		a.T.Fatalf("failed to read %s: %v", path, err)
	}
	return response.StatusCode, response.Header.Get("Content-Type"), string(content)
}

// DeckPage returns the HTML of the deck as this person sees it.
func (a *App) DeckPage(client *http.Client) string {
	a.T.Helper()
	_, _, page := a.Get(client, "/"+InviteKey+"/")
	return page
}

var slideIndexSignal = regexp.MustCompile(`slideIndex:\s*(\d+)`)

// ShownSlideIndex reads which slide the page shows, from the slideIndex Datastar signal rendered by
// ActiveSlide. Using the index instead of slide text keeps tests independent of content.
func ShownSlideIndex(t *testing.T, page string) int {
	t.Helper()
	match := slideIndexSignal.FindStringSubmatch(page)
	if match == nil {
		t.Fatalf("page has no slideIndex signal:\n%s", Truncate(page, 500))
	}
	index, err := strconv.Atoi(match[1])
	if err != nil {
		t.Fatalf("invalid slideIndex %q: %v", match[1], err)
	}
	return index
}

// ShowSlideWith steps the presenter through the deck until a slide containing marker is shown, like a
// presenter clicking "next", so tests do not depend on slide order. Returns the page an attendee sees then.
// It checks an attendee's page on purpose: the presenter's page also holds previews of every slide in its
// sidebar, so it contains every slide even when another one is active.
func (a *App) ShowSlideWith(presenter *http.Client, marker string) string {
	a.T.Helper()
	viewer := a.NewAttendee()
	for index := 0; ; index++ {
		if status := a.Post(presenter, fmt.Sprintf("/root/api/slides/set?index=%d", index)); status != http.StatusNoContent {
			a.T.Fatalf("presenter could not move to slide %d: status %d", index, status)
		}
		page := a.DeckPage(viewer)
		if strings.Contains(page, marker) {
			return page
		}
		if ShownSlideIndex(a.T, page) < index {
			a.T.Fatalf("reached the end of the deck without a slide containing %q", marker)
		}
	}
}

// Event is one server-sent event: its name and its data lines.
type Event struct {
	Name string
	Data []string
}

// Stream collects the server-sent events of a live stream in the background until the test ends.
type Stream struct {
	T      *testing.T
	Events chan Event
}

// OpenStream connects to a live stream under the invite key, like a page's data-init does.
func (a *App) OpenStream(client *http.Client, path string) *Stream {
	a.T.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	a.T.Cleanup(cancel)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, a.server.URL+"/"+InviteKey+path, nil)
	if err != nil {
		a.T.Fatalf("failed to create stream request: %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		a.T.Fatalf("failed to open stream %s: %v", path, err)
	}
	a.T.Cleanup(func() { _ = response.Body.Close() }) // closing an aborted stream reports the abort

	stream := &Stream{T: a.T, Events: make(chan Event, 1000)}
	go func() {
		defer close(stream.Events)
		scanner := bufio.NewScanner(response.Body)
		current := Event{}
		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case line == "":
				if current.Name != "" {
					stream.Events <- current
				}
				current = Event{}
			case strings.HasPrefix(line, "event: "):
				current.Name = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				current.Data = append(current.Data, strings.TrimPrefix(line, "data: "))
			}
		}
	}()
	return stream
}

func CloseBody(t *testing.T, response *http.Response) {
	t.Helper()
	if err := response.Body.Close(); err != nil {
		t.Errorf("failed to close response body: %v", err)
	}
}

func Truncate(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return strings.TrimSpace(text[:maxLength]) + "…"
}
