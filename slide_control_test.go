package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/bdd"
)

func TestSlideControl_Attendee_CannotChangeTheSlide(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given an attendee who is not in presenter mode.",
		When:  "When they try to move the deck to the next slide.",
		Then:  "Then the request is forbidden and everyone still sees the first slide.",
	})

	// Given
	expectedStatus := http.StatusForbidden
	expectedSlideIndex := 0
	app := startTestApp(t)
	attendee := app.newAttendee()
	app.deckPage(attendee)

	// When
	actualStatus := app.post(attendee, "/root/api/slides/set?index=1")

	// Then
	if actualStatus != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, actualStatus)
	}
	actualSlideIndex := shownSlideIndex(t, app.deckPage(attendee))
	if actualSlideIndex != expectedSlideIndex {
		t.Fatalf("expected slide %d, got %d", expectedSlideIndex, actualSlideIndex)
	}
}

func TestSlideControl_Presenter_ChangesTheSlideForEveryone(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given a presenter and an attendee looking at the first slide.",
		When:  "When the presenter moves the deck to the next slide.",
		Then:  "Then the attendee sees the next slide.",
	})

	// Given
	expectedStatus := http.StatusNoContent
	expectedSlideIndex := 1
	app := startTestApp(t)
	presenter := app.newPresenter()
	attendee := app.newAttendee()
	app.deckPage(presenter)
	app.deckPage(attendee)

	// When
	actualStatus := app.post(presenter, "/root/api/slides/set?index=1")

	// Then
	if actualStatus != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, actualStatus)
	}
	actualSlideIndex := shownSlideIndex(t, app.deckPage(attendee))
	if actualSlideIndex != expectedSlideIndex {
		t.Fatalf("expected slide %d, got %d", expectedSlideIndex, actualSlideIndex)
	}
}

func TestSlideControl_NobodyHasOpenedTheDeck_PresenterCanStillChangeTheSlide(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given the server has just started and nobody has the deck open yet.",
		When:  "When the presenter moves the deck to the next slide.",
		Then:  "Then the slide changes instead of failing because there is nobody to notify.",
	})

	// Given
	expectedStatus := http.StatusNoContent
	expectedSlideIndex := 1
	app := startTestApp(t)
	presenter := app.newPresenter()

	// When
	actualStatus := app.post(presenter, "/root/api/slides/set?index=1")

	// Then
	if actualStatus != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, actualStatus)
	}
	actualSlideIndex := shownSlideIndex(t, app.deckPage(presenter))
	if actualSlideIndex != expectedSlideIndex {
		t.Fatalf("expected slide %d, got %d", expectedSlideIndex, actualSlideIndex)
	}
}

func TestPresenterNavigation_ShowsTheSlidesSourceFile(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given the presenter is on the first slide, a markdown slide.",
		When:  "When the presenter controls are shown.",
		Then:  "Then they name the markdown file the slide comes from, so it is easy to find.",
	})

	// Given
	expectedSource := regexp.MustCompile(`class="presenter-nav-source">slides/[a-z_]+\.md<`)
	app := startTestApp(t)
	presenter := app.newPresenter()

	// When
	page := app.deckPage(presenter)

	// Then
	if !expectedSource.MatchString(page) {
		t.Fatalf("the presenter controls do not name the slide's markdown file:\n%s", truncate(page, 1500))
	}
}

func TestPresenterNavigation_Attendee_SeesNoSourceFile(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given an attendee on the deck.",
		When:  "When the attendee's page is shown.",
		Then:  "Then it names no source file: that is for the presenter.",
	})

	// Given
	hiddenSource := `class="presenter-nav-source"`
	app := startTestApp(t)

	// When
	page := app.deckPage(app.newAttendee())

	// Then
	if strings.Contains(page, hiddenSource) {
		t.Fatalf("an attendee's page names the slide's source file")
	}
}

func TestPresenterNavigation_FirstSlide_OffersNoPreviousSlide(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given the presenter is on the first slide.",
		When:  "When the presenter controls are shown.",
		Then:  "Then there is no action to go to a previous slide, only to the next one.",
	})

	// Given
	expectedNextAction := slideSetAction(1)
	missingPreviousAction := slideSetAction(-1)
	app := startTestApp(t)
	presenter := app.newPresenter()

	// When
	page := app.deckPage(presenter)

	// Then
	if strings.Contains(page, missingPreviousAction) {
		t.Fatalf("first slide offers a previous-slide action %q", missingPreviousAction)
	}
	if !strings.Contains(page, expectedNextAction) {
		t.Fatalf("first slide is missing the next-slide action %q", expectedNextAction)
	}
}

func TestPresenterNavigation_LastSlide_OffersNoNextSlide(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given the presenter has moved to the last slide.",
		When:  "When the presenter controls are shown.",
		Then:  "Then there is no action to go to a next slide, only back to the previous one.",
	})

	// Given
	app := startTestApp(t)
	presenter := app.newPresenter()
	lastIndex := app.moveToLastSlide(presenter)
	expectedPreviousAction := slideSetAction(lastIndex - 1)
	missingNextAction := slideSetAction(lastIndex + 1)

	// When
	page := app.deckPage(presenter)

	// Then
	if strings.Contains(page, missingNextAction) {
		t.Fatalf("last slide offers a next-slide action %q", missingNextAction)
	}
	if !strings.Contains(page, expectedPreviousAction) {
		t.Fatalf("last slide is missing the previous-slide action %q", expectedPreviousAction)
	}
}

// slideSetAction is the Datastar action a presenter control uses to move the deck to index.
func slideSetAction(index int) string {
	return fmt.Sprintf("/root/api/slides/set?index=%d", index)
}

// moveToLastSlide steps the presenter forward until the deck stops advancing and returns the last index.
func (app *testApp) moveToLastSlide(presenter *http.Client) int {
	app.t.Helper()
	for index := 1; ; index++ {
		app.post(presenter, fmt.Sprintf("/root/api/slides/set?index=%d", index))
		if shown := shownSlideIndex(app.t, app.deckPage(presenter)); shown < index {
			return shown
		}
	}
}

func TestPresenterNotes_OnlyThePresenterSeesThem(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given a slide with presenter notes, an attendee and a presenter.",
		When:  "When both look at the slide.",
		Then:  "Then only the presenter sees the notes.",
	})

	// Given
	notesMarker := `class="presenter-notes"`
	app := startTestApp(t)
	attendee := app.newAttendee()
	presenter := app.newPresenter()

	// When
	attendeePage := app.deckPage(attendee)
	presenterPage := app.deckPage(presenter)

	// Then
	if strings.Contains(attendeePage, notesMarker) {
		t.Fatalf("the attendee sees presenter notes")
	}
	if !strings.Contains(presenterPage, notesMarker) {
		t.Fatalf("the presenter does not see the presenter notes")
	}
}
