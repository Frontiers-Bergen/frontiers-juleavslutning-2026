package main

import (
	"net/http"
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
