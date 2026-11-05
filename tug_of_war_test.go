package main

import (
	"net/http"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/bdd"
)

func TestTugOfWar_Teams_AreEven(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given the tug of war slide is shown.",
		When:  "When seven people look at it on their phones.",
		Then:  "Then each gets a team, and the teams differ by at most one person.",
	})

	// Given
	maximumDifference := 1
	app := startTestApp(t)
	firstViewerPage := app.showSlideWith(app.newPresenter(), "data-tug-of-war-phase")
	teamSizes := map[string]int{tugOfWarState(t, firstViewerPage).team: 1}

	// When
	for range 6 {
		teamSizes[tugOfWarState(t, app.deckPage(app.newAttendee())).team]++
	}

	// Then
	difference := teamSizes["teal"] - teamSizes["orange"]
	if teamSizes["teal"]+teamSizes["orange"] != 7 || difference > maximumDifference || difference < -maximumDifference {
		t.Fatalf("expected two teams differing by at most %d, got %v", maximumDifference, teamSizes)
	}
}

func TestTugOfWar_OnlyThePresenter_CanStartARound(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given an attendee on the tug of war slide.",
		When:  "When they try to start the round themselves.",
		Then:  "Then it is forbidden and the game keeps waiting for the presenter.",
	})

	// Given
	expectedStatus := http.StatusForbidden
	expectedPhase := "waiting"
	app := startTestApp(t)
	app.showSlideWith(app.newPresenter(), "data-tug-of-war-phase")
	attendee := app.newAttendee()

	// When
	actualStatus := app.post(attendee, "/root/api/tug-of-war/start")

	// Then
	if actualStatus != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, actualStatus)
	}
	if phase := tugOfWarState(t, app.deckPage(attendee)).phase; phase != expectedPhase {
		t.Fatalf("expected phase %q, got %q", expectedPhase, phase)
	}
}

func TestTugOfWar_PullBeforeTheCountdownEnds_DoesNotCount(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given the presenter has just started the countdown.",
		When:  "When an eager attendee pulls before it ends.",
		Then:  "Then the pull is rejected and no team gets a point.",
	})

	// Given
	expectedStatus := http.StatusConflict
	expectedPulls := 0
	app := startTestApp(t)
	presenter := app.newPresenter()
	app.showSlideWith(presenter, "data-tug-of-war-phase")
	attendee := app.newAttendee()
	app.deckPage(attendee)
	app.startTugOfWar(presenter)

	// When
	actualStatus := app.post(attendee, "/root/api/tug-of-war/pull")

	// Then
	if actualStatus != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, actualStatus)
	}
	state := tugOfWarState(t, app.deckPage(attendee))
	if state.tealPulls != expectedPulls || state.orangePulls != expectedPulls {
		t.Fatalf("expected no pulls, got teal %d, orange %d", state.tealPulls, state.orangePulls)
	}
}

func TestTugOfWar_PullsDuringTheRound_CountForYourTeam(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given a round has started and the countdown is over.",
		When:  "When an attendee pulls three times.",
		Then:  "Then their team has three pulls and the other team none.",
	})

	// Given
	expectedOwnPulls, expectedOtherPulls := 3, 0
	app := startTestApp(t)
	presenter := app.newPresenter()
	app.showSlideWith(presenter, "data-tug-of-war-phase")
	attendee := app.newAttendee()
	team := tugOfWarState(t, app.deckPage(attendee)).team
	app.startTugOfWar(presenter)
	time.Sleep(testTugOfWarRules.Countdown + 50*time.Millisecond)

	// When
	for range expectedOwnPulls {
		app.pullForTeam(attendee)
	}

	// Then
	state := tugOfWarState(t, app.deckPage(attendee))
	ownPulls, otherPulls := state.tealPulls, state.orangePulls
	if team == "orange" {
		ownPulls, otherPulls = state.orangePulls, state.tealPulls
	}
	if ownPulls != expectedOwnPulls || otherPulls != expectedOtherPulls {
		t.Fatalf("expected %d pulls for %s and %d for the other team, got teal %d, orange %d",
			expectedOwnPulls, team, expectedOtherPulls, state.tealPulls, state.orangePulls)
	}
}

func TestTugOfWar_WinningLead_EndsTheRoundEarly(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given a round is running and one team pulls alone.",
		When:  "When that team reaches the winning lead.",
		Then:  "Then the round is over at once, that team wins, and further pulls do not count.",
	})

	// Given
	expectedPhase := "finished"
	expectedStatusAfterWin := http.StatusConflict
	app := startTestApp(t)
	presenter := app.newPresenter()
	app.showSlideWith(presenter, "data-tug-of-war-phase")
	attendee := app.newAttendee()
	team := tugOfWarState(t, app.deckPage(attendee)).team
	app.startTugOfWar(presenter)
	time.Sleep(testTugOfWarRules.Countdown + 50*time.Millisecond)

	// When
	for range testTugOfWarRules.WinningLead {
		app.pullForTeam(attendee)
	}
	statusAfterWin := app.post(attendee, "/root/api/tug-of-war/pull")

	// Then
	state := tugOfWarState(t, app.deckPage(attendee))
	if state.phase != expectedPhase || state.winner != team {
		t.Fatalf("expected phase %q won by %s, got phase %q won by %q", expectedPhase, team, state.phase, state.winner)
	}
	if statusAfterWin != expectedStatusAfterWin {
		t.Fatalf("expected a pull after the win to get status %d, got %d", expectedStatusAfterWin, statusAfterWin)
	}
}

func TestTugOfWar_AfterTheRoundEnds_PullsDoNotCount(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given a round that has run out of time.",
		When:  "When an attendee pulls.",
		Then:  "Then the pull is rejected and the round shows as finished.",
	})

	// Given
	expectedStatus := http.StatusConflict
	expectedPhase := "finished"
	app := startTestApp(t)
	presenter := app.newPresenter()
	app.showSlideWith(presenter, "data-tug-of-war-phase")
	attendee := app.newAttendee()
	app.deckPage(attendee)
	app.startTugOfWar(presenter)
	time.Sleep(testTugOfWarRules.Countdown + testTugOfWarRules.Round + 100*time.Millisecond)

	// When
	actualStatus := app.post(attendee, "/root/api/tug-of-war/pull")

	// Then
	if actualStatus != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, actualStatus)
	}
	if phase := tugOfWarState(t, app.deckPage(attendee)).phase; phase != expectedPhase {
		t.Fatalf("expected phase %q, got %q", expectedPhase, phase)
	}
}

// --- Local helpers for the tug of war tests ---

func (app *testApp) startTugOfWar(presenter *http.Client) {
	app.t.Helper()
	if status := app.post(presenter, "/root/api/tug-of-war/start"); status != http.StatusNoContent {
		app.t.Fatalf("presenter could not start the tug of war: status %d", status)
	}
}

func (app *testApp) pullForTeam(attendee *http.Client) {
	app.t.Helper()
	if status := app.post(attendee, "/root/api/tug-of-war/pull"); status != http.StatusNoContent {
		app.t.Fatalf("a pull during the round got status %d", status)
	}
}

type tugOfWarPage struct {
	phase       string
	team        string
	winner      string
	tealPulls   int
	orangePulls int
}

var (
	tugOfWarPhase       = regexp.MustCompile(`data-tug-of-war-phase="([a-z]+)"`)
	tugOfWarTeam        = regexp.MustCompile(`data-team="([a-z]+)"`)
	tugOfWarTealPulls   = regexp.MustCompile(`data-teal-pulls="(\d+)"`)
	tugOfWarOrangePulls = regexp.MustCompile(`data-orange-pulls="(\d+)"`)
	tugOfWarWinner      = regexp.MustCompile(`data-winner="([a-z]+)"`)
)

// tugOfWarState reads what the tug of war slide shows from its semantic data attributes.
func tugOfWarState(t *testing.T, page string) tugOfWarPage {
	t.Helper()
	find := func(pattern *regexp.Regexp, required bool) string {
		match := pattern.FindStringSubmatch(page)
		if match == nil {
			if required {
				t.Fatalf("tug of war slide has no %s:\n%s", pattern, truncate(page, 500))
			}
			return ""
		}
		return match[1]
	}
	number := func(text string) int {
		value, err := strconv.Atoi(text)
		if err != nil {
			t.Fatalf("invalid number %q: %v", text, err)
		}
		return value
	}
	return tugOfWarPage{
		phase:       find(tugOfWarPhase, true),
		team:        find(tugOfWarTeam, true),
		winner:      find(tugOfWarWinner, false),
		tealPulls:   number(find(tugOfWarTealPulls, true)),
		orangePulls: number(find(tugOfWarOrangePulls, true)),
	}
}
