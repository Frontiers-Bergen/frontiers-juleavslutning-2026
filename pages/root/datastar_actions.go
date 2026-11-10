package root

import "fmt"

// Datastar actions for data-* attributes: data-on:click, data-init, data-on:<event>. They are JavaScript
// expressions, not URLs, so pass them as plain strings, never through templ.URL: templ.URL treats the text
// before the first colon as a URL scheme and silently replaces anything it does not trust with about:invalid.
//
// Every action keeps its request when another one starts (requestCancellation disabled): a second tap must not
// cancel the first, because every smash, vote and slide change counts.

// postAction is the expression that POSTs to a route under the invite key, for example
// postAction(inviteKey, "/root/api/tug-of-war/start").
func postAction(inviteKey InviteKey, path string) string {
	return fmt.Sprintf("@post('/%s%s',{requestCancellation: 'disabled'})", inviteKey, path)
}

// streamAction is the expression that opens a live stream (server-sent events) under the invite key.
func streamAction(inviteKey InviteKey, path string) string {
	return fmt.Sprintf("@get('/%s%s',{requestCancellation: 'disabled'})", inviteKey, path)
}

// activatedByKeyboard runs action on Enter or Space, as a real button does. For an element with role="button"
// that cannot be a <button>, such as a slide preview, which itself contains buttons.
func activatedByKeyboard(action string) string {
	return fmt.Sprintf("if (evt.key === 'Enter' || evt.key === ' ') { evt.preventDefault(); %s }", action)
}
