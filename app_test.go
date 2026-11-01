package main

import (
	"context"
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
)

// Test harness for HTTP-level tests: the real router from newApp, with a fresh SQLite file
// and JetStream store per test. Each person (attendee, presenter) is an http.Client with
// its own cookie jar, like a separate phone.

const (
	testLocalKey     = "test-local-key"
	testPresenterKey = "test-presenter-key"
)

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
	app, closeApp, err := newApp(ctx, silentLogger, appConfig{
		DatabasePath:  filepath.Join(t.TempDir(), "presentation.db"),
		NATSStoreDir:  t.TempDir(),
		LocalKey:      testLocalKey,
		RemoteKey:     "test-remote-key",
		SessionSecret: "test-session-secret",
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

// post sends a request to a route under the local invite key and returns the status code.
func (app *testApp) post(client *http.Client, path string) int {
	app.t.Helper()
	response, err := client.Post(app.server.URL+"/"+testLocalKey+path, "", nil)
	if err != nil {
		app.t.Fatalf("POST %s failed: %v", path, err)
	}
	closeBody(app.t, response)
	return response.StatusCode
}

// deckPage returns the HTML of the deck as this person sees it.
func (app *testApp) deckPage(client *http.Client) string {
	app.t.Helper()
	response, err := client.Get(app.server.URL + "/" + testLocalKey + "/")
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
