package root_test

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/apptest"
	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/bdd"
)

func TestEmojiReactions_ConcurrentSmashes_EverySmashIsCounted(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given a room full of attendees.",
		When:  "When they smash the 👍 button 50 times at the same moment.",
		Then:  "Then the 👍 count is exactly 50; no smash is lost.",
	})

	// Given
	expectedThumbsUpCount := 50
	app := apptest.Start(t)
	attendees := make([]*http.Client, expectedThumbsUpCount)
	for i := range attendees {
		attendees[i] = app.NewAttendee()
	}

	// When
	var smashes sync.WaitGroup
	for _, attendee := range attendees {
		smashes.Go(func() { smashEmoji(app, attendee, "👍") })
	}
	smashes.Wait()

	// Then
	actualCounts := reactionCounts(t, app.DeckPage(app.NewAttendee()))
	if actualCounts["👍"] != expectedThumbsUpCount {
		t.Fatalf("expected 👍 count %d, got %d", expectedThumbsUpCount, actualCounts["👍"])
	}
}

func TestEmojiReactions_BurstOfSmashes_OneBalloonPerSmash(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given an attendee watching the reactions on their phone.",
		When:  "When others smash 🔥 20 times in a quick burst.",
		Then:  "Then the watcher sees 20 🔥 balloons, not one per burst.",
	})

	// Given
	expectedBalloonCount := 20
	app := apptest.Start(t)
	stream := openEmojiStream(app, app.NewAttendee())
	nextReactionCounts(stream) // the buttons with their counts, on connect
	smasher := app.NewAttendee()

	// When
	var smashes sync.WaitGroup
	for range expectedBalloonCount {
		smashes.Go(func() { smashEmoji(app, smasher, "🔥") })
	}
	smashes.Wait()

	// Then
	actualBalloonCount := countBalloonsUntilQuiet(stream, "🔥", time.Second)
	if actualBalloonCount != expectedBalloonCount {
		t.Fatalf("expected %d 🔥 balloons, got %d", expectedBalloonCount, actualBalloonCount)
	}
}

func TestEmojiReactions_UnknownEmoji_IsRejected(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given the reaction bar has five emojis.",
		When:  "When someone sends a reaction with an emoji that is not one of them.",
		Then:  "Then it is rejected and no count changes.",
	})

	// Given
	expectedStatus := http.StatusBadRequest
	expectedTotal := 0
	app := apptest.Start(t)
	attendee := app.NewAttendee()

	// When
	actualStatus := postEmoji(app, attendee, "💩")

	// Then
	if actualStatus != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, actualStatus)
	}
	actualCounts := reactionCounts(t, app.DeckPage(app.NewAttendee()))
	actualTotal := 0
	for _, count := range actualCounts {
		actualTotal += count
	}
	if actualTotal != expectedTotal {
		t.Fatalf("expected no counts to change, got %v", actualCounts)
	}
}

// --- Local helpers for the emoji reaction tests ---

var reactionButton = regexp.MustCompile(`data-emoji="([^"]+)"\s+data-count="(\d+)"`)

// reactionCounts reads each reaction button's count from its data-count, in a page or a buttons patch.
func reactionCounts(t *testing.T, html string) map[string]int {
	t.Helper()
	counts := map[string]int{}
	for _, match := range reactionButton.FindAllStringSubmatch(html, -1) {
		count, err := strconv.Atoi(match[2])
		if err != nil {
			t.Fatalf("invalid count %q: %v", match[2], err)
		}
		counts[match[1]] = count
	}
	if len(counts) == 0 {
		t.Fatalf("no reaction buttons found:\n%s", apptest.Truncate(html, 500))
	}
	return counts
}

func postEmoji(app *apptest.App, client *http.Client, emoji string) int {
	app.T.Helper()
	body, err := json.Marshal(map[string]string{"emoji": emoji})
	if err != nil {
		app.T.Errorf("failed to marshal emoji signals: %v", err)
		return 0
	}
	response, err := client.Post(app.URL()+"/"+apptest.InviteKey+"/root/api/emoji/increment", "application/json", strings.NewReader(string(body)))
	if err != nil {
		app.T.Errorf("emoji POST failed: %v", err)
		return 0
	}
	apptest.CloseBody(app.T, response)
	return response.StatusCode
}

// smashEmoji is safe to call from many goroutines: it and postEmoji report failures with t.Errorf, never t.Fatalf.
func smashEmoji(app *apptest.App, client *http.Client, emoji string) {
	if status := postEmoji(app, client, emoji); status >= 300 {
		app.T.Errorf("smashing %s returned status %d", emoji, status)
	}
}

// openEmojiStream connects to the reaction stream like the reaction bar does.
func openEmojiStream(app *apptest.App, client *http.Client) *apptest.Stream {
	app.T.Helper()
	return app.OpenStream(client, "/root/api/emoji/sse")
}

// nextReactionCounts waits for the reaction buttons on the emoji stream and returns their counts.
func nextReactionCounts(stream *apptest.Stream) map[string]int {
	stream.T.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case event, open := <-stream.Events:
			if !open {
				stream.T.Fatalf("emoji stream closed before sending the buttons")
			}
			if html := strings.Join(event.Data, "\n"); event.Name == "datastar-patch-elements" && strings.Contains(html, `id="reaction-buttons"`) {
				return reactionCounts(stream.T, html)
			}
		case <-timeout:
			stream.T.Fatalf("no reaction buttons within 5 seconds")
		}
	}
}

// countBalloonsUntilQuiet counts appended <emoji-balloon> elements for emoji until the stream
// has been quiet for quietPeriod. One patch can carry several balloons: smashes already waiting
// for a stream go out together.
func countBalloonsUntilQuiet(stream *apptest.Stream, emoji string, quietPeriod time.Duration) int {
	stream.T.Helper()
	balloonMarkup := `<emoji-balloon emoji="` + emoji + `"`
	count := 0
	for {
		select {
		case event, open := <-stream.Events:
			if !open {
				return count
			}
			if event.Name == "datastar-patch-elements" {
				count += strings.Count(strings.Join(event.Data, "\n"), balloonMarkup)
			}
		case <-time.After(quietPeriod):
			return count
		}
	}
}
