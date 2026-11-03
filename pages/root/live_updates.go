package root

import (
	"fmt"
	"net/http"

	"github.com/delaneyj/toolbelt/id"
	"github.com/gorilla/sessions"
	"github.com/nats-io/nats.go"
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
	sessionID = id.NextEncodedID()
	session.Values["id"] = sessionID
	if err := session.Save(r, w); err != nil {
		return "", fmt.Errorf("failed to save attendee session: %w", err)
	}
	return sessionID, nil
}
