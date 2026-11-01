package main

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

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
	app := startTestApp(t)
	attendees := make([]*http.Client, expectedThumbsUpCount)
	for i := range attendees {
		attendees[i] = app.newAttendee()
	}

	// When
	var smashes sync.WaitGroup
	for _, attendee := range attendees {
		smashes.Go(func() { app.smashEmoji(attendee, "👍") })
	}
	smashes.Wait()

	// Then
	stream := app.openEmojiStream(app.newAttendee())
	actualCounts := stream.nextEmojiCounts()
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
	app := startTestApp(t)
	stream := app.openEmojiStream(app.newAttendee())
	stream.nextEmojiCounts() // initial counts on connect
	smasher := app.newAttendee()

	// When
	var smashes sync.WaitGroup
	for range expectedBalloonCount {
		smashes.Go(func() { app.smashEmoji(smasher, "🔥") })
	}
	smashes.Wait()

	// Then
	actualBalloonCount := stream.countBalloonsUntilQuiet("🔥", time.Second)
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
	app := startTestApp(t)
	attendee := app.newAttendee()

	// When
	actualStatus := app.postEmoji(attendee, "💩")

	// Then
	if actualStatus != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, actualStatus)
	}
	actualCounts := app.openEmojiStream(app.newAttendee()).nextEmojiCounts()
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

func (app *testApp) postEmoji(client *http.Client, emoji string) int {
	app.t.Helper()
	body, err := json.Marshal(map[string]string{"emoji": emoji})
	if err != nil {
		app.t.Fatalf("failed to marshal emoji signals: %v", err)
	}
	response, err := client.Post(app.server.URL+"/"+testLocalKey+"/root/api/emoji/increment", "application/json", strings.NewReader(string(body)))
	if err != nil {
		app.t.Errorf("emoji POST failed: %v", err)
		return 0
	}
	closeBody(app.t, response)
	return response.StatusCode
}

// smashEmoji is safe to call from many goroutines; it reports failures with t.Errorf.
func (app *testApp) smashEmoji(client *http.Client, emoji string) {
	if status := app.postEmoji(client, emoji); status >= 300 {
		app.t.Errorf("smashing %s returned status %d", emoji, status)
	}
}

type sseEvent struct {
	name string
	data []string
}

type emojiStream struct {
	t      *testing.T
	events chan sseEvent
}

// openEmojiStream connects to the reaction stream like the reaction bar does, and collects
// its server-sent events in the background until the test ends.
func (app *testApp) openEmojiStream(client *http.Client) *emojiStream {
	app.t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	app.t.Cleanup(cancel)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, app.server.URL+"/"+testLocalKey+"/root/api/emoji/sse", nil)
	if err != nil {
		app.t.Fatalf("failed to create stream request: %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		app.t.Fatalf("failed to open emoji stream: %v", err)
	}
	app.t.Cleanup(func() { _ = response.Body.Close() }) // closing an aborted stream reports the abort

	stream := &emojiStream{t: app.t, events: make(chan sseEvent, 1000)}
	go func() {
		defer close(stream.events)
		scanner := bufio.NewScanner(response.Body)
		current := sseEvent{}
		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case line == "":
				if current.name != "" {
					stream.events <- current
				}
				current = sseEvent{}
			case strings.HasPrefix(line, "event: "):
				current.name = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				current.data = append(current.data, strings.TrimPrefix(line, "data: "))
			}
		}
	}()
	return stream
}

func (stream *emojiStream) nextEmojiCounts() emojiCounts {
	stream.t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case event, open := <-stream.events:
			if !open {
				stream.t.Fatalf("emoji stream closed before sending counts")
			}
			if event.name != "datastar-patch-signals" {
				continue
			}
			counts := emojiCounts{}
			signals := strings.TrimPrefix(strings.Join(event.data, ""), "signals ")
			if err := json.Unmarshal([]byte(signals), &counts); err != nil {
				stream.t.Fatalf("failed to decode emoji counts %q: %v", signals, err)
			}
			return counts
		case <-timeout:
			stream.t.Fatalf("no emoji counts within 5 seconds")
		}
	}
}

// countBalloonsUntilQuiet counts appended <emoji-balloon> elements for emoji until the stream
// has been quiet for quietPeriod.
func (stream *emojiStream) countBalloonsUntilQuiet(emoji string, quietPeriod time.Duration) int {
	stream.t.Helper()
	balloonMarkup := `emoji="` + emoji + `"`
	count := 0
	for {
		select {
		case event, open := <-stream.events:
			if !open {
				return count
			}
			if event.name == "datastar-patch-elements" && strings.Contains(strings.Join(event.data, "\n"), balloonMarkup) {
				count++
			}
		case <-time.After(quietPeriod):
			return count
		}
	}
}
