-- Applied on every startup (main.go), so every statement must be idempotent.
-- There are no migrations: changing an existing table means deleting the database file and starting fresh.
-- That is fine until the juleavslutning; freeze schema changes on the evening itself.

-- The presenter's current slide. Exactly one row; the app only ever UPDATEs it.
CREATE TABLE IF NOT EXISTS slide_state (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  current_index INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

INSERT OR IGNORE INTO slide_state (id, current_index) VALUES (1, 0);

-- One vote per person (attendee session) per poll; voting again replaces the vote.
CREATE TABLE IF NOT EXISTS poll_votes (
  poll_key TEXT NOT NULL,
  session_id TEXT NOT NULL,
  option_key TEXT NOT NULL,
  voted_at_unix INTEGER NOT NULL,
  PRIMARY KEY (poll_key, session_id)
);

-- Emoji reaction counts. Incremented in SQL so concurrent smashes cannot overwrite each other
-- (pages/root/emoji_counter.templ). A row appears on an emoji's first smash.
CREATE TABLE IF NOT EXISTS emoji_counter (
  emoji TEXT PRIMARY KEY,
  click_count INTEGER NOT NULL DEFAULT 0
);
