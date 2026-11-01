package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/delaneyj/toolbelt"
	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/models"
	"github.com/gorilla/sessions"
	"github.com/nats-io/nats.go/jetstream"
)

func notifyUpdate(sessionID string) {
	_ = sessionID
}

// BroadcastUpdate makes every connected client re-render.
// Each client's SSE handler (pages/root/root_index.templ) watches its own session key in the KV bucket
// and patches the whole page when that key changes. Re-saving every session's state, even unchanged,
// fires all those watchers. The stored state itself is a leftover from the Datastar Todo example.
func BroadcastUpdate(kv jetstream.KeyValue, r *http.Request) error {
	ctx := r.Context()

	allKeys, err := kv.Keys(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve keys: %w", err)
	}

	for _, sessionID := range allKeys {
		entry, err := kv.Get(ctx, sessionID)
		if err != nil {
			// Key may have been deleted or changed between Keys and Get; skip it.
			continue
		}

		var state models.TodoPageState
		if err := json.Unmarshal(entry.Value(), &state); err != nil {
			// If the stored value is not a valid TodoPageState, skip this session.
			continue
		}

		if err := savePageState(ctx, &state, sessionID, kv, notifyUpdate); err != nil {
			return fmt.Errorf("failed to re-save page state for session %s: %w", sessionID, err)
		}
	}

	return nil
}

func LoadOrCreateState(w http.ResponseWriter, r *http.Request, kv jetstream.KeyValue, store sessions.Store) (string, *models.TodoPageState, error) {
	ctx := r.Context()

	sessionID, err := upsertSessionID(store, r, w)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get session id: %w", err)
	}

	pageState := &models.TodoPageState{}
	entry, err := kv.Get(ctx, sessionID)
	if err != nil {
		if err != jetstream.ErrKeyNotFound {
			return "", nil, fmt.Errorf("failed to get key value: %w", err)
		}

		// First visit → save an empty state
		if err := savePageState(ctx, pageState, sessionID, kv, notifyUpdate); err != nil {
			return "", nil, fmt.Errorf("failed to save initial state: %w", err)
		}
		return sessionID, pageState, nil
	}

	if err := json.Unmarshal(entry.Value(), pageState); err != nil {
		return "", nil, fmt.Errorf("failed to unmarshal page state: %w", err)
	}

	return sessionID, pageState, nil
}

func savePageState(
	ctx context.Context,
	state *models.TodoPageState,
	sessionID string,
	kv jetstream.KeyValue,
	poke func(string),
) error {

	bytes, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal page state: %w", err)
	}

	if _, err := kv.Put(ctx, sessionID, bytes); err != nil {
		return fmt.Errorf("failed to put key value: %w", err)
	}

	if poke != nil {
		poke(sessionID)
	}

	return nil
}

func upsertSessionID(store sessions.Store, r *http.Request, w http.ResponseWriter) (string, error) {
	const cookieName = "fmj26-session"

	sess, err := store.Get(r, cookieName)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	id, ok := sess.Values["id"].(string)
	if !ok || id == "" {
		id = toolbelt.NextEncodedID()
		sess.Values["id"] = id
	}

	if err := sess.Save(r, w); err != nil {
		return "", fmt.Errorf("failed to save session: %w", err)
	}

	return id, nil
}
