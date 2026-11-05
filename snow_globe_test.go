package main

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/bdd"
)

func TestSnowGlobe_Shaking_LetsItSnowOnOtherScreens(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given the big screen is showing the deck.",
		When:  "When someone shakes their phone five times.",
		Then:  "Then it snows on the big screen, swirling with the energy of five shakes.",
	})

	// Given
	expectedEnergy := 5
	expectedFlakes := 15
	app := startTestApp(t)
	bigScreen := app.openStream(app.newAttendee(), "/root/api/snow/sse")
	shaker := app.newAttendee()

	// When
	app.shake(shaker, 5)

	// Then
	snowfall, found := bigScreen.nextSnowfall(time.Second)
	if !found {
		t.Fatalf("expected a snowfall on the big screen, got none")
	}
	if snowfall.energy != expectedEnergy || snowfall.flakes != expectedFlakes {
		t.Fatalf("expected energy %d with %d flakes, got energy %d with %d flakes",
			expectedEnergy, expectedFlakes, snowfall.energy, snowfall.flakes)
	}
}

func TestSnowGlobe_TheShakingPhone_DoesNotGetItsOwnSnowTwice(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given a phone that already shows its own puffs of snow while it is shaken.",
		When:  "When it sends its shakes to the server.",
		Then:  "Then the server does not send that phone the same snow again.",
	})

	// Given
	app := startTestApp(t)
	shaker := app.newAttendee()
	ownScreen := app.openStream(shaker, "/root/api/snow/sse")

	// When
	app.shake(shaker, 3)

	// Then
	if _, found := ownScreen.nextSnowfall(500 * time.Millisecond); found {
		t.Fatalf("the shaking phone got its own snow from the server")
	}
}

func TestSnowGlobe_HugeNumberOfShakes_IsCapped(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given a phone that claims to have been shaken a thousand times.",
		When:  "When it sends its shakes.",
		Then:  "Then the snowfall is capped, so one phone cannot bury every screen in snow.",
	})

	// Given
	expectedEnergy := 20
	expectedFlakes := 24
	app := startTestApp(t)
	bigScreen := app.openStream(app.newAttendee(), "/root/api/snow/sse")

	// When
	app.shake(app.newAttendee(), 1000)

	// Then
	snowfall, found := bigScreen.nextSnowfall(time.Second)
	if !found || snowfall.energy != expectedEnergy || snowfall.flakes != expectedFlakes {
		t.Fatalf("expected a capped snowfall (energy %d, %d flakes), got %+v (found: %t)",
			expectedEnergy, expectedFlakes, snowfall, found)
	}
}

func TestSnowGlobe_NoShakes_IsRejected(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given the snow globe endpoint.",
		When:  "When a request claims zero shakes.",
		Then:  "Then it is rejected.",
	})

	// Given
	expectedStatus := http.StatusBadRequest
	app := startTestApp(t)

	// When
	actualStatus := app.postShakes(app.newAttendee(), 0)

	// Then
	if actualStatus != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, actualStatus)
	}
}

// --- Local helpers for the snow globe tests ---

// shake sends a burst of shakes the way <snow-globe> does when the shaking stops.
func (app *testApp) shake(phone *http.Client, shakes int) {
	app.t.Helper()
	if status := app.postShakes(phone, shakes); status != http.StatusNoContent {
		app.t.Fatalf("sending %d shakes got status %d", shakes, status)
	}
}

func (app *testApp) postShakes(phone *http.Client, shakes int) int {
	app.t.Helper()
	body, err := json.Marshal(map[string]int{"shakes": shakes})
	if err != nil {
		app.t.Fatalf("failed to marshal shakes: %v", err)
	}
	response, err := phone.Post(app.server.URL+"/"+testInviteKey+"/root/api/snow/shake", "application/json", strings.NewReader(string(body)))
	if err != nil {
		app.t.Fatalf("POST shakes failed: %v", err)
	}
	closeBody(app.t, response)
	return response.StatusCode
}

type snowfallPatch struct {
	energy int
	flakes int
}

var snowfallEnergy = regexp.MustCompile(`class="snowfall" style="--energy: (\d+);"`)

// nextSnowfall waits up to timeout for a snowfall appended to the page.
func (stream *sseStream) nextSnowfall(timeout time.Duration) (snowfallPatch, bool) {
	stream.t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case event, open := <-stream.events:
			if !open {
				return snowfallPatch{}, false
			}
			elements := strings.Join(event.data, "\n")
			match := snowfallEnergy.FindStringSubmatch(elements)
			if event.name != "datastar-patch-elements" || match == nil {
				continue
			}
			energy, err := strconv.Atoi(match[1])
			if err != nil {
				stream.t.Fatalf("invalid snowfall energy %q: %v", match[1], err)
			}
			return snowfallPatch{energy: energy, flakes: strings.Count(elements, `class="snowflake"`)}, true
		case <-deadline:
			return snowfallPatch{}, false
		}
	}
}
