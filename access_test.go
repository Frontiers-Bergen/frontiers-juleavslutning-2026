package main

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/bdd"
)

func TestAccess_WrongInviteKey_IsSentToTheNeedAKeyPage(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given someone on the internet who guesses a URL.",
		When:  "When they open the deck with an invite key that is not ours.",
		Then:  "Then they are redirected to the start page that says they need a key, without seeing the deck.",
	})

	// Given
	expectedPath := "/"
	app := startTestApp(t)
	stranger := app.newAttendee()

	// When
	actualPath := app.finalPathAfterGet(stranger, "/not-our-key/")

	// Then
	if actualPath != expectedPath {
		t.Fatalf("expected to end up on %q, got %q", expectedPath, actualPath)
	}
}

func TestAccess_LocalAndRemoteInviteKeys_OpenTheDeck(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given an attendee in the room and one watching remotely, each with their own link.",
		When:  "When they open the deck.",
		Then:  "Then both see the deck.",
	})

	for _, inviteKey := range []string{testLocalKey, testRemoteKey} {
		t.Run(inviteKey, func(t *testing.T) {
			// Given
			expectedPath := "/" + inviteKey + "/"
			app := startTestApp(t)

			// When
			actualPath := app.finalPathAfterGet(app.newAttendee(), "/"+inviteKey+"/")

			// Then
			if actualPath != expectedPath {
				t.Fatalf("expected to stay on %q, got %q", expectedPath, actualPath)
			}
		})
	}
}

func TestPresenterLogin_WrongKey_DoesNotGivePresenterControl(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given an attendee who tries to log in as presenter with a guessed key.",
		When:  "When they then try to move the deck.",
		Then:  "Then the login is rejected and they cannot change the slide.",
	})

	// Given
	expectedLoginError := "invalid_key"
	expectedStatus := http.StatusForbidden
	app := startTestApp(t)
	attendee := app.newAttendee()

	// When
	actualLoginError := app.presenterLogin(attendee, "guessed-key")
	actualStatus := app.post(attendee, "/root/api/slides/set?index=1")

	// Then
	if actualLoginError != expectedLoginError {
		t.Fatalf("expected login error %q, got %q", expectedLoginError, actualLoginError)
	}
	if actualStatus != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, actualStatus)
	}
}

func TestPresenterLogin_NoPresenterKeyConfigured_NobodyCanLogIn(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given a server started without PRESENTER_KEY.",
		When:  "When someone logs in as presenter with an empty key.",
		Then:  "Then the login is rejected; a missing key never means everyone is presenter.",
	})

	// Given
	expectedLoginError := "invalid_key"
	app := startTestApp(t)
	t.Setenv("PRESENTER_KEY", "")
	attendee := app.newAttendee()

	// When
	actualLoginError := app.presenterLogin(attendee, "")

	// Then
	if actualLoginError != expectedLoginError {
		t.Fatalf("expected login error %q, got %q", expectedLoginError, actualLoginError)
	}
}

// --- Local helpers for the access tests ---

// finalPathAfterGet follows redirects and returns the path the client ends up on.
func (app *testApp) finalPathAfterGet(client *http.Client, path string) string {
	app.t.Helper()
	response, err := client.Get(app.server.URL + path)
	if err != nil {
		app.t.Fatalf("GET %s failed: %v", path, err)
	}
	closeBody(app.t, response)
	if response.StatusCode != http.StatusOK {
		app.t.Fatalf("GET %s ended with status %d", path, response.StatusCode)
	}
	return response.Request.URL.Path
}

// presenterLogin submits the presenter login form and returns the error code the login page shows ("" on success).
func (app *testApp) presenterLogin(client *http.Client, key string) string {
	app.t.Helper()
	response, err := client.PostForm(app.server.URL+"/presenter/login", url.Values{"key": {key}})
	if err != nil {
		app.t.Fatalf("presenter login failed: %v", err)
	}
	closeBody(app.t, response)
	return response.Request.URL.Query().Get("err")
}
