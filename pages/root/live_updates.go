package root

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/nats-io/nats.go"
	"github.com/starfederation/datastar-go/datastar"
)

// Live updates for the deck page: when anything shown on it changes (the presenter moves the deck, someone
// votes), every open page renders itself again.
//
// A change publishes one core NATS message on deckChangedSubject. Each page's SSE stream (RootLayoutRoute)
// subscribes and patches the whole page on every message. The message carries no data: the page reads
// fresh state from SQLite when it renders. Emoji smashes have their own subject (emoji_counter.templ).

const deckChangedSubject = "deck.changed"

func publishDeckChanged(nc *nats.Conn) error {
	if err := nc.Publish(deckChangedSubject, nil); err != nil {
		return fmt.Errorf("failed to publish deck change: %w", err)
	}
	return nil
}

// compressedStream is the compression every live stream uses. Each stream keeps its own Brotli compressor for as
// long as the page is open, and the default window (4 MB) cost about 36 MB per phone over its three streams:
// 1.4 GB at 40 phones, on a shared server. A 256 KB window still holds the previous page, so an update compresses
// just as well (same bytes and CPU, measured with go tool task loadtest), at about 8 MB per phone.
func compressedStream() datastar.SSEOption {
	return datastar.WithCompression(datastar.WithBrotli(datastar.WithBrotliLevel(5), datastar.WithBrotliLGWin(18)))
}

// logStreamFailure logs a failed send on a live stream, which then ends. When the request is already over, the
// phone simply left: it closed the tab, locked its screen, or Datastar closed the stream of a hidden page. That is
// a normal goodbye, so it is logged at debug level; as an error it drowned out the real ones.
func logStreamFailure(logger *slog.Logger, r *http.Request, message string, err error) {
	if r.Context().Err() != nil {
		logger.Debug(message+" (the phone left)", slog.String("error", err.Error()))
		return
	}
	logger.Error(message, slog.String("error", err.Error()))
}

// attendeeSessionCookie identifies one phone (browser) across requests, so a poll vote can be changed
// instead of counted twice. There are no accounts; clearing cookies makes a new attendee.
const attendeeSessionCookie = "attendee-session"

// attendeeSessionID returns this browser's session ID, creating it on the first visit.
// Call it before an SSE stream starts: it may need to set the cookie.
func attendeeSessionID(store sessions.Store, w http.ResponseWriter, r *http.Request) (string, error) {
	session, err := store.Get(r, attendeeSessionCookie)
	if err != nil {
		return "", fmt.Errorf("failed to read attendee session: %w", err)
	}
	sessionID, ok := session.Values["id"].(string)
	if ok && sessionID != "" {
		return sessionID, nil
	}
	// crypto/rand: unguessable, and safe when many phones join at the same moment. toolbelt's id generator,
	// used before, shared a counter without a lock, so two phones joining at once could get the same ID.
	sessionID = rand.Text()
	session.Values["id"] = sessionID
	if err := session.Save(r, w); err != nil {
		return "", fmt.Errorf("failed to save attendee session: %w", err)
	}
	return sessionID, nil
}
