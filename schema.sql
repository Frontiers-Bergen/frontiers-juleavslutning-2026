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

-- Tug of war (pages/root/tug_of_war.templ): which team each attendee pulls for in the current round. Cleared when
-- a round starts; assigned when the attendee's page next shows the game, to the smaller team, so the teams stay even.
CREATE TABLE IF NOT EXISTS tug_of_war_teams (
  session_id TEXT PRIMARY KEY,
  team TEXT NOT NULL CHECK (team IN ('teal', 'orange'))
);

-- The current tug of war round. Exactly one row; the presenter starting a round resets it.
-- The round's rules are stored with it, so the slide can work out the phase from this row alone.
CREATE TABLE IF NOT EXISTS tug_of_war_round (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  started_at_unix_ms INTEGER,            -- NULL until the presenter starts the first round
  countdown_ms INTEGER NOT NULL DEFAULT 0,
  round_ms INTEGER NOT NULL DEFAULT 0,
  winning_lead INTEGER NOT NULL DEFAULT 0,
  teal_pulls INTEGER NOT NULL DEFAULT 0,
  orange_pulls INTEGER NOT NULL DEFAULT 0
);

INSERT OR IGNORE INTO tug_of_war_round (id) VALUES (1);
