package app

import (
	"net/http"
	"regexp"
	"strconv"
	"sync"
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

func TestTugOfWar_PhonesJoiningAtOnce_TeamsAreEven(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given the tug of war slide is shown.",
		When:  "When the first phone has it open and 39 more open it at the same moment.",
		Then:  "Then the forty phones split into two teams of twenty.",
	})

	// Given
	expectedTeamSize := 20
	app := startTestApp(t)
	firstPhonePage := app.showSlideWith(app.newPresenter(), "data-tug-of-war-phase")
	teamSizes := map[string]int{tugOfWarState(t, firstPhonePage).team: 1}
	var mu sync.Mutex

	// When
	var phones sync.WaitGroup
	for range 2*expectedTeamSize - 1 {
		phones.Go(func() {
			team := tugOfWarState(t, app.deckPage(app.newAttendee())).team
			mu.Lock()
			defer mu.Unlock()
			teamSizes[team]++
		})
	}
	phones.Wait()

	// Then
	if teamSizes["teal"] != expectedTeamSize || teamSizes["orange"] != expectedTeamSize {
		t.Fatalf("expected two teams of %d, got %v", expectedTeamSize, teamSizes)
	}
}

func TestTugOfWar_StartingARound_DrawsNewTeams(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given two phones on the same team, and others who opened the slide and left.",
		When:  "When the presenter starts a round with only those two phones still watching.",
		Then:  "Then the two are drawn into one team each, so the teams are even among the people playing now.",
	})

	// Given
	expectedTeamSizes := map[string]int{"teal": 1, "orange": 1}
	app := startTestApp(t)
	presenter := app.newPresenter()
	app.showSlideWith(presenter, "data-tug-of-war-phase")
	stillWatching := app.twoPhonesOnTheSameTeam()

	// When
	app.startTugOfWar(presenter)
	teamSizes := map[string]int{}
	for _, phone := range stillWatching {
		teamSizes[tugOfWarState(t, app.deckPage(phone)).team]++
	}

	// Then
	if teamSizes["teal"] != expectedTeamSizes["teal"] || teamSizes["orange"] != expectedTeamSizes["orange"] {
		t.Fatalf("expected one phone on each team, got %v", teamSizes)
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

func TestTugOfWar_SmashBeforeTheCountdownEnds_DoesNotPull(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given the presenter has just started the countdown.",
		When:  "When an eager attendee smashes an emoji before it ends.",
		Then:  "Then no team gets a pull.",
	})

	// Given
	expectedPulls := 0
	app := startTestApp(t)
	presenter := app.newPresenter()
	app.showSlideWith(presenter, "data-tug-of-war-phase")
	attendee := app.newAttendee()
	app.deckPage(attendee)
	app.startTugOfWar(presenter)

	// When
	app.smash(attendee)

	// Then
	state := tugOfWarState(t, app.deckPage(attendee))
	if state.tealPulls != expectedPulls || state.orangePulls != expectedPulls {
		t.Fatalf("expected no pulls, got teal %d, orange %d", state.tealPulls, state.orangePulls)
	}
}

func TestTugOfWar_SmashesDuringTheRound_PullForYourTeam(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given a round has started and the countdown is over.",
		When:  "When an attendee smashes emojis three times.",
		Then:  "Then their team has three pulls and the other team none.",
	})

	// Given
	expectedOwnPulls, expectedOtherPulls := 3, 0
	app := startTestApp(t)
	presenter := app.newPresenter()
	app.showSlideWith(presenter, "data-tug-of-war-phase")
	attendee := app.newAttendee()
	app.startTugOfWar(presenter)
	time.Sleep(testTugOfWarRules.Countdown + 50*time.Millisecond)
	team := tugOfWarState(t, app.deckPage(attendee)).team

	// When
	for range expectedOwnPulls {
		app.smash(attendee)
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
		Given: "Given a round is running and one team smashes alone.",
		When:  "When that team reaches the winning lead and keeps smashing.",
		Then:  "Then the round is over at once, that team wins, and further smashes do not pull.",
	})

	// Given
	expectedPhase := "finished"
	expectedPulls := testTugOfWarRules.WinningLead
	app := startTestApp(t)
	presenter := app.newPresenter()
	app.showSlideWith(presenter, "data-tug-of-war-phase")
	attendee := app.newAttendee()
	app.startTugOfWar(presenter)
	time.Sleep(testTugOfWarRules.Countdown + 50*time.Millisecond)
	team := tugOfWarState(t, app.deckPage(attendee)).team

	// When
	for range testTugOfWarRules.WinningLead + 2 {
		app.smash(attendee)
	}

	// Then
	state := tugOfWarState(t, app.deckPage(attendee))
	if state.phase != expectedPhase || state.winner != team {
		t.Fatalf("expected phase %q won by %s, got phase %q won by %q", expectedPhase, team, state.phase, state.winner)
	}
	if state.tealPulls+state.orangePulls != expectedPulls {
		t.Fatalf("expected the pulls to stop at %d, got teal %d, orange %d", expectedPulls, state.tealPulls, state.orangePulls)
	}
}

func TestTugOfWar_AfterTheRoundEnds_SmashesDoNotPull(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given a round that has run out of time.",
		When:  "When an attendee smashes an emoji.",
		Then:  "Then no team gets a pull and the round shows as finished.",
	})

	// Given
	expectedPulls := 0
	expectedPhase := "finished"
	app := startTestApp(t)
	presenter := app.newPresenter()
	app.showSlideWith(presenter, "data-tug-of-war-phase")
	attendee := app.newAttendee()
	app.deckPage(attendee)
	app.startTugOfWar(presenter)
	time.Sleep(testTugOfWarRules.Countdown + testTugOfWarRules.Round + 100*time.Millisecond)

	// When
	app.smash(attendee)

	// Then
	state := tugOfWarState(t, app.deckPage(attendee))
	if state.phase != expectedPhase {
		t.Fatalf("expected phase %q, got %q", expectedPhase, state.phase)
	}
	if state.tealPulls != expectedPulls || state.orangePulls != expectedPulls {
		t.Fatalf("expected no pulls, got teal %d, orange %d", state.tealPulls, state.orangePulls)
	}
}

// --- Local helpers for the tug of war tests ---

func (app *testApp) startTugOfWar(presenter *http.Client) {
	app.t.Helper()
	if status := app.post(presenter, "/root/api/tug-of-war/start"); status != http.StatusNoContent {
		app.t.Fatalf("presenter could not start the tug of war: status %d", status)
	}
}

// twoPhonesOnTheSameTeam opens the slide on new phones until two of them are on the same team; the other
// phones leave (they never load the page again).
func (app *testApp) twoPhonesOnTheSameTeam() []*http.Client {
	app.t.Helper()
	phonesByTeam := map[string][]*http.Client{}
	for range 3 {
		phone := app.newAttendee()
		team := tugOfWarState(app.t, app.deckPage(phone)).team
		phonesByTeam[team] = append(phonesByTeam[team], phone)
		if len(phonesByTeam[team]) == 2 {
			return phonesByTeam[team]
		}
	}
	app.t.Fatalf("three phones on two teams, yet no two on the same team: %v", phonesByTeam)
	return nil
}

// smash taps an emoji reaction button, which also pulls for the attendee's team during a round.
func (app *testApp) smash(attendee *http.Client) {
	app.t.Helper()
	if status := app.postEmoji(attendee, "🔥"); status != http.StatusNoContent {
		app.t.Fatalf("a smash got status %d", status)
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
