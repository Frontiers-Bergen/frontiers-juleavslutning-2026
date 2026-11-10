package root_test

import (
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"strconv"
	"testing"

	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/apptest"
	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/bdd"
)

// The tests use the app name poll, the only poll in the deck (pages/root/poll.templ).
const testPollKey = "app-name"

func TestPollVoting_ChangingYourVote_ReplacesThePreviousVote(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given an attendee who voted for Smash the Halls.",
		When:  "When they change their mind and vote for Rage Against the Debounce.",
		Then:  "Then the results show only their new vote.",
	})

	// Given
	expectedCounts := map[string]int{
		"rage-against-the-debounce": 1,
		"last-christmas":            0,
		"its-tiiime":                0,
		"smash-the-halls":           0,
	}
	app := apptest.Start(t)
	attendee := app.NewAttendee()
	vote(app, attendee, "smash-the-halls")

	// When
	vote(app, attendee, "rage-against-the-debounce")

	// Then
	actualCounts := voteCounts(t, showPollResults(app, app.NewPresenter(), testPollKey))
	assertVoteCounts(t, expectedCounts, actualCounts)
}

func TestPollResults_EveryPersonCountsOnce(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given four attendees, each on their own phone.",
		When:  "When three vote for It's Tiiime and one for Smash the Halls.",
		Then:  "Then the results show three and one votes: every person counts once.",
	})

	// Given
	expectedCounts := map[string]int{
		"rage-against-the-debounce": 0,
		"last-christmas":            0,
		"its-tiiime":                3,
		"smash-the-halls":           1,
	}
	app := apptest.Start(t)

	// When
	vote(app, app.NewAttendee(), "its-tiiime")
	vote(app, app.NewAttendee(), "its-tiiime")
	vote(app, app.NewAttendee(), "its-tiiime")
	vote(app, app.NewAttendee(), "smash-the-halls")

	// Then
	actualCounts := voteCounts(t, showPollResults(app, app.NewPresenter(), testPollKey))
	assertVoteCounts(t, expectedCounts, actualCounts)
}

func TestPollVoting_UnknownOption_IsRejected(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given the app name poll with four options.",
		When:  "When someone sends a vote for an option that does not exist.",
		Then:  "Then the vote is rejected and no option gets a vote.",
	})

	// Given
	expectedStatus := http.StatusBadRequest
	expectedCounts := map[string]int{
		"rage-against-the-debounce": 0,
		"last-christmas":            0,
		"its-tiiime":                0,
		"smash-the-halls":           0,
	}
	app := apptest.Start(t)

	// When
	actualStatus := app.Post(app.NewAttendee(), "/root/api/poll/vote?poll="+testPollKey+"&option=jingle-bugs")

	// Then
	if actualStatus != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, actualStatus)
	}
	actualCounts := voteCounts(t, showPollResults(app, app.NewPresenter(), testPollKey))
	assertVoteCounts(t, expectedCounts, actualCounts)
}

// --- Local helpers for the poll tests ---

func vote(app *apptest.App, client *http.Client, optionKey string) {
	app.T.Helper()
	path := fmt.Sprintf("/root/api/poll/vote?poll=%s&option=%s", testPollKey, optionKey)
	if status := app.Post(client, path); status >= 300 {
		app.T.Fatalf("voting for %s returned status %d", optionKey, status)
	}
}

// showPollResults moves the deck to the results slide for pollKey and returns the page an attendee sees.
func showPollResults(app *apptest.App, presenter *http.Client, pollKey string) string {
	app.T.Helper()
	return app.ShowSlideWith(presenter, fmt.Sprintf(`data-poll-results="%s"`, pollKey))
}

var pollOptionResult = regexp.MustCompile(`data-option-key="([^"]+)"\s+data-vote-count="(\d+)"`)

// voteCounts reads each option's vote count from a results slide.
func voteCounts(t *testing.T, resultsPage string) map[string]int {
	t.Helper()
	counts := map[string]int{}
	for _, match := range pollOptionResult.FindAllStringSubmatch(resultsPage, -1) {
		count, err := strconv.Atoi(match[2])
		if err != nil {
			t.Fatalf("invalid vote count %q for %s: %v", match[2], match[1], err)
		}
		counts[match[1]] = count
	}
	if len(counts) == 0 {
		t.Fatalf("results page shows no options:\n%s", apptest.Truncate(resultsPage, 500))
	}
	return counts
}

func assertVoteCounts(t *testing.T, expected, actual map[string]int) {
	t.Helper()
	if !maps.Equal(expected, actual) {
		t.Fatalf("vote counts mismatch\nexpected: %v\nactual:   %v", expected, actual)
	}
}
