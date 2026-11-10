package root

import (
	"crypto/rand"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
)

// Named types for values that are easily mixed up: the invite key and a session ID are both strings that travel
// side by side through every page. Parse them at the boundary (the URL, the cookie) and pass the typed value on.

// InviteKey is the secret first path segment of every deck URL (/<InviteKey>/). Everyone at the venue shares it.
type InviteKey string

// SessionID identifies one phone (browser) across requests, from its attendee-session cookie. There are no
// accounts: clearing cookies makes a new attendee.
type SessionID string

// inviteKeyOf is the invite key in the request's URL. The invite key middleware (app.go) has already checked it.
func inviteKeyOf(r *http.Request) InviteKey {
	return InviteKey(chi.URLParam(r, "inviteKey"))
}

// attendeeSessionCookie identifies one phone (browser) across requests, so a poll vote can be changed
// instead of counted twice. There are no accounts; clearing cookies makes a new attendee.
const attendeeSessionCookie = "attendee-session"

// attendeeSessionID returns this browser's session ID, creating it on the first visit.
// Call it before an SSE stream starts: it may need to set the cookie.
func attendeeSessionID(store sessions.Store, w http.ResponseWriter, r *http.Request) (SessionID, error) {
	session, err := store.Get(r, attendeeSessionCookie)
	if err != nil {
		return "", fmt.Errorf("failed to read attendee session: %w", err)
	}
	if stored, ok := session.Values["id"].(string); ok && stored != "" {
		return SessionID(stored), nil
	}
	// crypto/rand: unguessable, and safe when many phones join at the same moment. toolbelt's id generator,
	// used before, shared a counter without a lock, so two phones joining at once could get the same ID.
	sessionID := rand.Text()
	session.Values["id"] = sessionID
	if err := session.Save(r, w); err != nil {
		return "", fmt.Errorf("failed to save attendee session: %w", err)
	}
	return SessionID(sessionID), nil
}
