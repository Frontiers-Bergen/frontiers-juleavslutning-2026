package root_test

import (
	"encoding/json"
	"net/http"
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
	stream := openEmojiStream(app, app.NewAttendee())
	actualCounts := nextEmojiCounts(stream)
	if actualCounts.ThumbsUpEmojiCount != expectedThumbsUpCount {
		t.Fatalf("expected 👍 count %d, got %d", expectedThumbsUpCount, actualCounts.ThumbsUpEmojiCount)
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
	nextEmojiCounts(stream) // initial counts on connect
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
	expectedCounts := emojiCounts{}
	app := apptest.Start(t)
	attendee := app.NewAttendee()

	// When
	actualStatus := postEmoji(app, attendee, "💩")

	// Then
	if actualStatus != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, actualStatus)
	}
	actualCounts := nextEmojiCounts(openEmojiStream(app, app.NewAttendee()))
	if actualCounts != expectedCounts {
		t.Fatalf("expected counts %+v, got %+v", expectedCounts, actualCounts)
	}
}

// --- Local helpers for the emoji reaction tests ---

// emojiCounts mirrors the Datastar signals the reaction bar reads ($ThumbsUpEmojiCount etc.).
type emojiCounts struct {
	ThumbsUpEmojiCount int
	ConfettiEmojiCount int
	CryLaughEmojiCount int
	FireEmojiCount     int
	HeartEmojiCount    int
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

func nextEmojiCounts(stream *apptest.Stream) emojiCounts {
	stream.T.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case event, open := <-stream.Events:
			if !open {
				stream.T.Fatalf("emoji stream closed before sending counts")
			}
			if event.Name != "datastar-patch-signals" {
				continue
			}
			counts := emojiCounts{}
			signals := strings.TrimPrefix(strings.Join(event.Data, ""), "signals ")
			if err := json.Unmarshal([]byte(signals), &counts); err != nil {
				stream.T.Fatalf("failed to decode emoji counts %q: %v", signals, err)
			}
			return counts
		case <-timeout:
			stream.T.Fatalf("no emoji counts within 5 seconds")
		}
	}
}

// countBalloonsUntilQuiet counts appended <emoji-balloon> elements for emoji until the stream
// has been quiet for quietPeriod. One patch can carry several balloons: smashes already waiting
// for a stream go out together.
func countBalloonsUntilQuiet(stream *apptest.Stream, emoji string, quietPeriod time.Duration) int {
	stream.T.Helper()
	balloonMarkup := `emoji="` + emoji + `"`
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
