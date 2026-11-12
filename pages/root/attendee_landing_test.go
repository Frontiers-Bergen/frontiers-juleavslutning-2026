package root_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/apptest"
	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/bdd"
	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/pages/root"
)

func TestAttendeeLanding_ShowsProfileChoicesWithoutJoining(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given an attendee who has not chosen their name and avatar.",
		When:  "When the landing screen is shown.",
		Then:  "Then they can choose their details but cannot yet join the deck.",
	})

	// Given
	expected := []string{
		`for="attendee-name"`,
		`name="username"`,
		`<legend>Choose your avatar</legend>`,
		`type="radio" name="avatar"`,
		`disabled`,
		`Joining the deck is coming soon`,
	}
	var page bytes.Buffer

	// When
	err := root.AttendeeLanding().Render(t.Context(), &page)

	// Then
	if err != nil {
		t.Fatalf("failed to render landing screen: %v", err)
	}
	for _, content := range expected {
		if !strings.Contains(page.String(), content) {
			t.Errorf("landing screen does not contain %q", content)
		}
	}
}

func TestAttendeeLanding_InviteLink_ShowsLandingWithoutDeckStreams(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given an attendee with an invite link.",
		When:  "When they open and reload the page.",
		Then:  "Then they see the landing screen without the deck or snow globe.",
	})

	// Given
	expected := `class="attendee-landing"`
	unexpected := []string{`id="root-page-wrapper"`, `class="snow-globe"`, `data-slide-index=`, `/root/api/`}
	app := apptest.Start(t)
	attendee := app.NewAttendee()

	// When
	pages := []string{app.DeckPage(attendee), app.DeckPage(attendee)}

	// Then
	for _, page := range pages {
		if !strings.Contains(page, expected) {
			t.Errorf("attendee page does not contain %q", expected)
		}
		for _, content := range unexpected {
			if strings.Contains(page, content) {
				t.Errorf("attendee page unexpectedly contains %q", content)
			}
		}
	}
}

func TestAttendeeLanding_Presenter_StillSeesDeck(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given a presenter with an invite link.",
		When:  "When they open the page.",
		Then:  "Then they see the presenter deck and its live features, not the landing screen.",
	})

	// Given
	expected := []string{`class="presenter-layout"`, `id="root-page-wrapper"`, `/root/api/`, `class="snow-globe"`, `data-slide-index=`}
	app := apptest.Start(t)
	presenter := app.NewPresenter()

	// When
	page := app.DeckPage(presenter)

	// Then
	for _, content := range expected {
		if !strings.Contains(page, content) {
			t.Errorf("presenter page does not contain %q", content)
		}
	}
	if strings.Contains(page, `class="attendee-landing"`) {
		t.Error("presenter page unexpectedly contains the attendee landing")
	}
}
