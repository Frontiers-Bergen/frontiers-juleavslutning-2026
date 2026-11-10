package app_test

import (
	app "github.com/frontiers-bergen/frontiers-juleavslutning-2026"
	"net/http"
	"net/url"
	"testing"

	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/apptest"
	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/bdd"
)

func TestAccess_WrongInviteKey_IsSentToTheNeedAKeyPage(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given someone on the internet who guesses a URL.",
		When:  "When they open the deck with an invite key that is not ours.",
		Then:  "Then they are redirected to the start page that says they need a key.",
	})

	// Given
	expectedPath := "/"
	app := apptest.Start(t)
	stranger := app.NewAttendee()

	// When
	actualPath := finalPathAfterGet(app, stranger, "/not-our-key/")

	// Then
	if actualPath != expectedPath {
		t.Fatalf("expected to end up on %q, got %q", expectedPath, actualPath)
	}
}

func TestAccess_InviteKey_OpensTheDeck(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given an attendee with the invite link.",
		When:  "When they open it.",
		Then:  "Then they see the deck.",
	})

	// Given
	expectedPath := "/" + apptest.InviteKey + "/"
	app := apptest.Start(t)

	// When
	actualPath := finalPathAfterGet(app, app.NewAttendee(), "/"+apptest.InviteKey+"/")

	// Then
	if actualPath != expectedPath {
		t.Fatalf("expected to stay on %q, got %q", expectedPath, actualPath)
	}
}

func TestPresenterLogin_WrongKey_IsRejected(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given an attendee who guesses the presenter key.",
		When:  "When they log in as presenter with the guessed key.",
		Then:  "Then the login page says the key is wrong.",
	})

	// Given
	expectedLoginError := "invalid_key"
	app := apptest.Start(t)
	attendee := app.NewAttendee()

	// When
	actualLoginError := presenterLogin(app, attendee, "guessed-key")

	// Then
	if actualLoginError != expectedLoginError {
		t.Fatalf("expected login error %q, got %q", expectedLoginError, actualLoginError)
	}
}

func TestPresenterLogin_WrongKey_DoesNotGivePresenterControl(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given an attendee whose presenter login with a guessed key was rejected.",
		When:  "When they try to move the deck.",
		Then:  "Then they cannot change the slide.",
	})

	// Given
	expectedStatus := http.StatusForbidden
	app := apptest.Start(t)
	attendee := app.NewAttendee()
	presenterLogin(app, attendee, "guessed-key")

	// When
	actualStatus := app.Post(attendee, "/root/api/slides/set?index=1")

	// Then
	if actualStatus != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, actualStatus)
	}
}

func TestPresenterLogin_NoPresenterKeyConfigured_NobodyCanLogIn(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given a server started without a presenter key.",
		When:  "When someone logs in as presenter with an empty key.",
		Then:  "Then the login is rejected; a missing key never means everyone is presenter.",
	})

	// Given
	expectedLoginError := "invalid_key"
	app := apptest.StartWith(t, func(config *app.Config) { config.PresenterKey = "" })
	attendee := app.NewAttendee()

	// When
	actualLoginError := presenterLogin(app, attendee, "")

	// Then
	if actualLoginError != expectedLoginError {
		t.Fatalf("expected login error %q, got %q", expectedLoginError, actualLoginError)
	}
}

// --- Local helpers for the access tests ---

// finalPathAfterGet follows redirects and returns the path the client ends up on.
func finalPathAfterGet(app *apptest.App, client *http.Client, path string) string {
	app.T.Helper()
	response, err := client.Get(app.URL() + path)
	if err != nil {
		app.T.Fatalf("GET %s failed: %v", path, err)
	}
	apptest.CloseBody(app.T, response)
	if response.StatusCode != http.StatusOK {
		app.T.Fatalf("GET %s ended with status %d", path, response.StatusCode)
	}
	return response.Request.URL.Path
}

// presenterLogin submits the presenter login form and returns the error code the login page shows ("" on success).
func presenterLogin(app *apptest.App, client *http.Client, key string) string {
	app.T.Helper()
	response, err := client.PostForm(app.URL()+"/presenter/login", url.Values{"key": {key}})
	if err != nil {
		app.T.Fatalf("presenter login failed: %v", err)
	}
	apptest.CloseBody(app.T, response)
	return response.Request.URL.Query().Get("err")
}
