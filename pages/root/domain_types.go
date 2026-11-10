package root

import (
	"net/http"

	"github.com/go-chi/chi/v5"
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
