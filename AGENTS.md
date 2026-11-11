# Agent Instructions

Read the **Why** section in `README.md` first. Every change should serve it: the people at the Frontiers juleavslutning having fun.
The README also holds the checklist of planned work and the non-goals.

When you help someone install or run the app, follow `SETUP.md`. When you help someone add a mini-game, follow `MINI_GAMES.md`.

## Engineering principles

The goal while coding is to create a maintainable, high-quality solution that is easy to understand, change, and verify.

- Keep in mind good engineering practices such as KISS and YAGNI.
- Aim for low coupling, high cohesion and separation of concerns. Code that changes together should stay together.
- Avoid adding new dependencies without a clear benefit.
- Prefer clear names, straightforward control flow, and familiar project conventions.
- Complexity is our enemy. Avoid accidental complexity.
- Be pragmatic. If the rules conflict, use your judgment to decide which is more important for the current change.

### Abstraction and duplication

Do not prematurely abstract. Only share code when it represents the same underlying concept or business rule.
Never duplicate a business rule; duplicate incidental code freely until the third copy (Martin Fowler's rule of three).

### Avoid primitive obsession

Give domain concepts their own types instead of passing raw strings and ints around.
A named type (`type InviteKey string`, `type PollKey string`, `type SessionID string`) lets the compiler catch mixed-up arguments,
gives the concept a home for its rules, and makes function signatures read like the domain.
Parse and validate at the boundary (HTTP request, database row) and pass the typed value inward.
Use a named type when a value has rules or is easily confused with another value of the same primitive type; do not wrap values that are just data.

### Small changes

Strive for small, cohesive changes. Solve one problem completely and keep unrelated cleanup separate.
Small changes are easier to review, understand, and roll back. Work towards the smallest coherent change.

## Working with the user

- First inspect the relevant code and tests, then discuss.
- Ask clarifying questions, or interview the user, when the request is unclear.
- Always share your thoughts and recommendations.
- Challenge the user's assumptions when you see a better approach.
- Do not over-engineer. Discuss with the user before adding accidental complexity.

## Pull request workflow

Every feature follows three steps:

1. **Tidy first.** Before adding the feature, look at the code it will touch. If a refactor or improvement makes the feature easier to add, do that first, without changing behavior.
2. **Add the feature.** The behavior change, with tests.
3. **Refactor before finalizing.** Clean up what adding the feature revealed, again without changing behavior.

Keep structural changes (tidying, refactoring) and behavior changes apart so each can be reviewed on its own terms.
For small tidyings, separate commits in one pull request are enough.
When a step is big enough to deserve its own review, use stacked pull requests (GitHub stacked PRs, `gh stack` CLI extension): each step is one PR based on the previous, merged bottom-up.
Before merging the bottom PR of a stack with "delete branch", point the next PR at `main`: deleting the branch under an open PR closes that PR instead of retargeting it.
Skip steps that are not needed; an empty tidy or refactor step is fine.

Merge conflicts in the three shared lists are expected when several features land on the same day: the slide order
(`pages/root/active_slide.templ`), the routes (`pages/root/root_index.templ`) and `schema.sql`. Such a conflict is
almost always two additions in the same place: keep both, in a sensible order, and run `go tool task check`.

Run `go tool task check` before opening a pull request: formatting, vet, `go fix`, golangci-lint and tests. Fix with `go tool task fmt` where it applies.
For UI changes, run `go tool task screenshots` before and after the change and look at the result; for a pure refactor, the screenshots should be pixel-identical.
For changes to interactions (Datastar attributes, SSE, buttons), run `go tool task browser-check`: it clicks real buttons in headless Chromium.
Extend `browser-tools/browser_check.mjs` when you add an interaction that matters on the evening.

## Comments and documentation

Comments and documentation are primarily written for other agents.
Add them where they explain purpose, business rules, constraints, and non-obvious decisions — context that cannot easily be read from the code.
Do not repeat implementation details. Update existing comments and documentation when behavior changes.

Document features with a user story focused on one user goal.
The word "and" can indicate that a story contains unrelated goals, but it is not a problem by itself; keep related behavior together when it serves one coherent goal.
Use other documentation formats when a user story is not a good fit.

The code is the source of truth for what the system currently does. Requirements and acceptance criteria describe what it should do.
If the code, tests, and documentation disagree, point out the discrepancy.
Do not assume the current implementation is correct, and do not change tests or documentation just to match it.

## Testing

Test behavior, not implementation. Verify meaningful outcomes, important failure cases, and regressions.
A test should keep passing when implementation details change and the behavior stays the same.

Write tests in given-when-then form so they read from intent to verification:

- Start each test with three lines of BDD metadata describing the test in plain words: Given some initial context, When an event occurs, Then the expected outcome.
  Use the project's BDD helper, `bdd.Behavior` in `bdd/bdd.go`. The metadata is written in English.
- Split the body into three visible sections: `// Given`, `// When`, `// Then`.
- Start the Given section with the expected result, so the test's purpose is clear before the setup details.
- Keep each test to one behavior and give it a specific name. Split creation, idempotency, validation, and error cases into separate tests.
- Move noisy setup and repeated assertions into local test helpers when that makes the test read like behavior.

```go
func TestPollResults_NoVotes_EveryOptionHasZeroVotes(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given a poll nobody has voted on.",
		When:  "When the results are counted.",
		Then:  "Then every option has zero votes.",
	})

	// Given
	expected := map[string]int{"yes": 0, "no": 0}
	db := openTestDatabase(t, filepath.Join(t.TempDir(), "presentation.db"))
	poll := createPoll(t, db, "yes", "no")

	// When
	actual := countVotes(t, db, poll)

	// Then
	assertVoteCounts(t, expected, actual)
}
```

Tests exist to let agents change the code safely, not to reach a coverage number.
Prefer tests at feature boundaries with real dependencies (a real SQLite file in `t.TempDir()`, real embedded NATS) over mocks.
Run them with `go tool task test`. `database_test.go` is a working example.
A test lives next to the code it tests (cohesion): `pages/root/tug_of_war_test.go` next to `tug_of_war.templ`, in the external
test package (`package root_test`). HTTP-level tests use the harness in `apptest/`: `apptest.Start` runs the real router from
`app.New` (`app.go`) with a fresh database, and each attendee or presenter is an `http.Client` with its own cookies.
A feature's own helpers (`smash`, `vote`) stay in its test file. `pages/root/slide_control_test.go` is a working example.

## Go

- Handle every error explicitly. Never ignore an error; make failure paths obvious.
- Prefer concrete types. Introduce interfaces only at the point of use, when they are needed.
- Avoid packages named `helpers`, `utils` or `common`; put code next to the feature that uses it.
- Prefer explicit parameters over hidden state.

## Page life cycle

This stack turns a lot of conventional web practice upside down. Read this before you design anything that updates a page.

Most frontend practice (client state stores, optimistic updates, partial updates, cache invalidation, debouncing,
"only send what changed") exists because re-rendering a page is expensive. Here it is not: the server renders the
whole page from SQLite in a fraction of a millisecond (measured: 0.1–0.4 ms for an attendee's page, about 0.8 ms
for the presenter's, which previews every slide), Datastar morphs only what actually changed into the DOM (keeping
focus, scroll and local signals), and the live stream is compressed, so a further page update costs a phone tens of
bytes (measured: ~38 bytes for an attendee, ~170 for the presenter). So the page life cycle stays simple:
**on every change, every open page renders itself again, whole, from the database.**

For perspective: responses under about 100 ms feel instant to people (Jakob Nielsen's classic thresholds: 0.1 s
instant, 1 s keeps the flow, 10 s loses attention). conorganizer, on the same stack and server, answered with a P50 of
4 ms and a P99 of 113 ms over the whole Regncon festival weekend. We are far below the point where
anyone could notice; optimising page updates further buys nothing and costs simplicity.

### The life cycle

1. **First load.** `GET /<invite key>/` renders the full page from SQLite, so the first paint already has content.
2. **One read stream.** The page's `data-init` opens one long-lived SSE stream (`/<invite key>/root/api/`), the read side.
   The handler first does everything that needs response headers (attendee session cookie, validation, `http.Error`),
   subscribes to `deck.changed`, then calls `datastar.NewSSE(w, r, compressedStream())` and immediately sends
   a full patch of `#root-page-wrapper`. After `NewSSE` the headers are gone: report errors with `logStreamFailure` and end the stream (it stays quiet when the phone simply left).
3. **Short writes.** A tap or click is a short `POST` (the write side): it validates, writes SQLite, publishes
   `deck.changed` (`publishDeckChanged` in `pages/root/live_updates.go`) and answers `204`. It never returns HTML.
4. **Every stream re-renders.** Each open stream gets the message, renders the whole page from SQLite and patches it.
   The message carries no data; the database is the only truth.
5. **The end.** Closing the tab ends the stream. Datastar also closes it while a page is hidden and reopens it when
   the page is visible again; a reconnect, like a server restart, simply gets a fresh full patch.

Datastar calls this CQRS (one read stream, short writes) plus "fat morph" (send the whole page, let morphing find the
difference). See https://data-star.dev/guide/the_tao_of_datastar. conorganizer uses the same life cycle
(`documentation/live-update-lifecycle.md` in Regncon/conorganizer), with a KV bucket per page type instead of one message.

### Rules

- **Re-render the whole page on every change, even many times per second.** Do not write targeted patches for one
  element, diffing, client-side caches, or "send only what changed" logic, and do not throttle or debounce updates:
  every smash deserves to count, and every page should show it.
  Changes that arrived while a stream was busy are covered by its next render, which reads everything from SQLite:
  the stream takes the waiting notifications along and renders once (`RootLayoutRoute`). Nothing waits for more,
  so this is not throttling; it only spares renders that would show the same state. It is safe because a change is
  written before it is announced, and the stream takes the waiting notifications before it reads.
- **State lives on the server**, in SQLite (slides, votes, emoji counts, teams, later ideas and feedback). NATS only says
  that something changed and keeps nothing between restarts; anything that must survive a restart goes in SQLite.
  Datastar signals are for local UI (an open menu, form input), never for shared state.
- **No optimistic updates.** The page shows what the server knows. If a write is slow enough to notice, show a loading
  indicator (`data-indicator`) and let the next patch replace it.
- **Rendering must stay cheap,** because it happens for every open page on every change: small, simple SQL, no network calls,
  no heavy work. Rendering should not change state; the one exception is idempotent setup such as assigning a tug of war team.
- **A timer only for what changes with time alone,** like a countdown (`tug_of_war.templ` ticks once a second). Smashes,
  votes and slide changes update the page on their own.
- **Events are not state.** When every single event must show up on its own (one balloon per emoji smash), the event
  gets its own subject and the stream appends a small patch for it (`emoji_counter.templ`). Its count still lives in
  SQLite and arrives with the page. Events are never merged or dropped: smashes already waiting for a stream go out
  in one patch with one balloon each, with the counts read once.
- **Navigation:** attendees do not navigate; the presenter moves the deck (the slide index lives in SQLite) and every
  page follows. Anything else uses plain links and server redirects, never client-side routing or history management.
- **Partial updates are the exception that needs a reason:** only for data so large that rendering the whole page is
  measurably slow, such as millions of rows or a DOM with hundreds of thousands of elements. Measure first, and discuss
  it with the user before adding it.
- **Testing follows the same idea:** HTTP tests read the full page an attendee gets (`apptest`), and
  `go tool task browser-check` checks that open pages follow live in a real browser.

## Project conventions

- Stack: Go, templ, Datastar (SSE), embedded NATS (core messages) for live updates, SQLite for all state.
- Group code by feature, not by technical role. A simple feature can be a single file; a feature with several related files gets its own folder.
- Develop mobile first. Attendees use their phones as controllers, so every feature must work well on a small touch screen, one-handed.
  The big screen is the shared stage; design for both, starting with the phone.
  In CSS, the base rules are for the phone and `@media (width >= 40rem)` adds the big-screen layout. Tap targets are at least 3rem (48px) tall.
- Follow `STYLE_GUIDE.md` for colours, fonts, the logo, layout, components and tone. It is our own style guide, based on the
  Frontiers Bergen website; the design tokens in `static/index.css` hold the values.
- Styling is plain CSS. No Tailwind or other CSS frameworks; the project started from the northstar example and its Tailwind leftovers were removed on purpose.
- Shared CSS (design tokens, element defaults, app layout) lives in `static/index.css`, which declares the layers `tokens, base, layout, features`.
  A feature's CSS lives in a `.css` file next to its `.templ` file (`pages/root/poll.css` next to `poll.templ`), nested under the feature's
  root class (`.poll { ... }`), so features cannot break each other. All CSS is nested (Torstein's wish, agreed): a file has one
  top-level rule per root element the feature renders (usually just one), and everything else is nested inside it, with `&` where needed.
  `@keyframes` cannot be nested in a style rule, so they stay top level, named after the feature. Do not wrap a feature file in `@layer`:
  `/feature-styles.css` imports every feature file into `layer(features)`, which always wins over the shared layout. Reuse the tokens
  (`var(--space-2)`, `var(--radius)`, ...). `static/index.css` is shared, not a feature, and nests like conorganizer's `index.css`:
  tokens, element defaults and each layout piece are top level, with their states and children nested inside them.
  The files are embedded and served by `feature_styles.go`; a new folder with CSS needs a `go:embed` pattern there (a test fails until it is added).
  No `<style>` blocks in templates, and no inline styles except to pass dynamic values as custom properties (`style="--share: 42%"`).
- Write explicit SQL, keep it close to where it is used, and handle every database error.
- `schema.sql` is the single source of truth for the database and is applied on every startup, so every statement must be idempotent (`CREATE TABLE IF NOT EXISTS`, `INSERT OR IGNORE`).
  There are no migrations: changing an existing table means deleting the local database file. Schema changes are frozen on the juleavslutning evening.
- Live updates and where state lives: see "Page life cycle" above.
- Keep templates close to their data access. If a feature both reads and writes, keep both in the same file or feature folder.
- Slide order is defined in `pages/root/active_slide.templ`; slide content lives in `slides/`.
  Content slides are markdown files, one per slide (`slides/welcome.md`), added to the deck with `slides.MustMarkdownSlide("welcome")`.
  The first `# heading` is the slide title (also in the presenter's slide list); text after a line containing only `Notes:` is presenter notes;
  a line `Illustration: /static/illustrations/smash-button.svg` puts a picture beside the text (under it on a phone), never over it;
  optional per-slide CSS goes in `slides/welcome.css`, nested under `.slide-welcome`. The format is documented in `slides/markdown_slides.go`.
  Interactive slides (polls, games) are templ components.
- Use the Go + SQLite + Datastar stack first; it handles most interactive features. Add a web component only when a feature needs
  something only the browser can do (a sensor, dragging at 60 fps, physics). Rocket (Datastar's web component API, in
  `static/datastar-rocket.js`, which also contains Datastar) is the choice then; `static/snow-globe.js` is the example.
  Props down, events up: the component emits events (bubbling) and Datastar markup on a wrapper element sends them to the server.
  Do not put Datastar actions (`@get`, `@post`) on the Rocket element itself: Rocket rewrites them into calls to the component's own actions.
- Domain terms are listed in `domain_dictionary.md` and are used unchanged everywhere: code, database, UI and docs.
  Everything else is in English. Frontiers runs its events in English, so the app is English only (no bilingual frontend).

## Generated files

Do not edit `*_templ.go` files. Edit the `.templ` source and run `go tool templ generate` (or `task build:templ`).

templ reference for LLMs: https://templ.guide/llms.md
