# Frontiers Juleavslutning 2026

Live, multiplayer audience app for Frontiers Bergen.

## Why

We want the people at the Frontiers juleavslutning to have fun.

This started because our old retro tool debounced emoji reactions, so our button smashes didn't count.
We would rather build an entire app than settle for that — or for PowerPoint.
Every button smash deserves to count.

**Success** is the people at the juleavslutning having fun.

**How:** we build it together as a community, with tools simple enough that anyone can contribute in an evening.

**Not goals:**

- Replacing AhaSlides in general. A feature is built because it makes the juleavslutning more fun, not because another tool has it.
- Showing off the stack or any specific technology.
- Multi-tenancy, accounts or logins.
- Scaling beyond one room plus remote viewers.
- Exporting ideas or feedback. It happens once; we read it straight from the database after the event.

## Workshop

[Workshop: Let's Build Our Christmas Meetup App!](https://www.meetup.com/frontiers-bergen/events/317219351/)
— Thursday 12 November 2026, 17:30, Solea AS, Bergen.

Attendees hack on this codebase together: collaborative mini-games, easter eggs, festive surprises.
The workshop is for fun and learning; it is fine if it produces nothing tangible.
That means the app must be ready for the juleavslutning without depending on what gets built at the workshop.

Forked from the January 2026 meetup deck ("Next.js + Firebase vs Go + Datastar").

## Checklist

Order of work: decide the repo name, then tidy first (refactoring and cleanup), then new features.

### Before the workshop (must have by 12 Nov)

- [x] Create the public GitHub repository: [frontiers-bergen/frontiers-juleavslutning-2026](https://github.com/frontiers-bergen/frontiers-juleavslutning-2026)
- [x] Rename the repo to `frontiers-juleavslutning-2026` (explicit for developers; the fun app name is decided separately)
- [x] Get write access to the new repo and move the `christmas-app` branch there from `Regncon/frontiers-meetup-january-2026`
- [ ] Draft a programme proposal for the juleavslutning (no programme exists yet)
- [x] Rename Go module to `github.com/frontiers-bergen/frontiers-juleavslutning-2026`
- [x] Remove `presentation.db*` from git and un-comment the ignore rules
- [x] Apply `schema.sql` automatically at startup so a fresh clone runs without a DB file (no migrations; decided against goose)
- [x] Move the `poll_votes` table from the comment in `pages/root/poll.templ` into `schema.sql`
- [ ] One-command setup for attendees (`task start`), documented in this README
- [ ] Verify a fresh clone works on Linux, macOS and Windows
- [x] Update Go (1.27) and dependencies (templ, datastar-go, nats, sqlite, chi, toolbelt) and tools (task, air, dlv)
- [ ] Update vendored `static/datastar.js` (currently v1.0.0-RC.7) to match datastar-go v1.2.2 — needs a manual check in the browser
- [x] Rewrite `AGENTS.md` with the new engineering guidelines; make `CLAUDE.md` point to it
- [x] Add `domain_dictionary.md` with the project's domain terms
- [x] Remove the January docs (`agent_project.md`, `project_summary.md`, `presentationOutline.md`); `README.md`, `AGENTS.md` and code comments replace them
- [ ] A short "how to add your own mini-game" guide (the extension point attendees will use)
- [ ] First example multiplayer game that shows the pattern end to end
- [ ] Prepare "good first tasks" for attendees (GitHub issues)
- [ ] Opening demo + codebase walkthrough slides for Gerhard
- [ ] App name vote at the workshop: a poll with 3–4 candidates, the winner becomes the app name shown on the big screen and phones.
      Candidates so far: Rage Against the Debounce, Last Christmas I Gave You My Clicks, It's Tiiime, Smash the Halls.
      The name must make people laugh out loud, not just chuckle.

### At and after the juleavslutning

- [ ] Sound: Gerhard has a portable system (venue not confirmed yet)
- [ ] Back up the production database afterwards; ideas and feedback are analysed from it
- [ ] CI/CD like conorganizer: test + build on every PR, each PR gets its own preview environment, main deploys to the big screen
- [ ] Slides written in markdown, rendered as part of the existing build (see `MdToHTML` in conorganizer)

### Code health

- [x] **Fix: emoji smashes could be lost — the very bug this app exists to fix.** Counts moved to SQLite (atomic increment), one NATS message per smash gives one balloon per smash, Datastar no longer cancels in-flight smashes, unknown emojis are rejected. Tests in `emoji_reactions_test.go`.

- [x] Stop ignoring errors in `main.go`, the demo handler and the root/presenter pages (`http.Error` after SSE has started)
- [ ] Stop ignoring errors in templates (`{{ x, _ := GetCurrentSlideIndex(db) }}`); needs data loaded before rendering — part of the architecture refactor
- [x] Clean up the emoji SSE handlers together with the emoji counting fix (the `services` package is gone)
- [x] Read the session secret from `SESSION_SECRET` instead of the hardcoded `"session-secret"`
- [ ] Replace the Todo-example state (`models/state.go`) used as the broadcast payload in `helpers/nats_helpers.go`, and move that code out of the `helpers` package. It is not dead code: `BroadcastUpdate` drives every live update
- [x] Remove unused slides that are not in the deck, and unreferenced images
- [x] Replace the January deck with a placeholder deck: a welcome slide and the app name poll with its results
- [x] Remove the January `/demo` page (`handler/`, `services/demo_service.go`, `pages/demo.templ`, `components/demo.templ`)
- [x] BDD helper (`bdd/bdd.go`, from conorganizer) and first behavior tests (`database_test.go`)
- [x] Access tests (`access_test.go`): wrong invite keys are redirected, local and remote keys open the deck, a wrong or missing presenter key gives no control
- [x] Safety net for polls before the CSS refactor (`poll_voting_test.go`): changing a vote replaces it, local + remote votes add up, unknown options are rejected
- [ ] Bring over conorganizer's `testutil/templtest` when a test needs to render a component on its own (not needed yet: HTTP tests read semantic `data-` attributes)
- [x] Fix: SQLite pragmas were set on one pooled connection only, so concurrent writes failed with `SQLITE_BUSY`
- [x] Refactor the CSS: `@layer tokens, base, layout, features` in `static/index.css`, feature CSS in `<style>` blocks inside the `.templ` files, no inline styles, dead classes and unused CDN fonts/tippy.js removed (pixel-identical before/after)
- [x] Mobile first pass: poll and results use one column on phones (no more mid-word breaks), less slide padding on phones, thumb-sized reaction buttons
- [x] Confirm the CSS approach with Torstein (coldasice): feature CSS in `.css` files next to the `.templ` files, served by `feature_styles.go`
- [x] **Fix: any attendee can change the presenter's slide.** `POST /{inviteKey}/root/api/slides/set` does not check presenter mode, so anyone with the invite link can move the deck (curl or devtools). Start with a test.
- [ ] Add secret keys to prevent the internet from accessing the services (carried over)
- [x] Remove all Tailwind leftovers from the northstar example (`gotailwind` tool, utility classes); the project uses plain CSS

### Features (AhaSlides-inspired)

- [x] Multiple choice poll (Kahoot style)
- [x] Emoji reactions without debouncing
- [ ] Retro board with sticky notes (the Julebord retrospective use case)
- [ ] Word cloud
- [ ] Audience Q&A with upvotes
- [ ] Quiz with scores and leaderboard
- [ ] Spinner wheel

### Mini-game ideas

- [ ] Tug of war: local vs remote audience (reuses the existing invite-key split)
- [ ] Collaborative pixel canvas (r/place, Christmas edition)
- [ ] Button-mash reindeer race
- [ ] Snowball fight
- [ ] Hidden easter eggs

### Later: architecture and design

- [ ] Refactor the architecture and design once a few more features exist and the shape of the project shows.
      Use a Reverse Conway's Law approach: design the code structure to match how we want people to work.
      Example: many workshop attendees build mini-games in parallel and PRs are merged live, so each game should be a self-contained feature folder,
      and shared hotspots like the slide list in `pages/root/active_slide.templ` should not force everyone to edit the same lines.

### To discuss later

- [ ] Demo data in a separate `.sql` file, if we need it

- [ ] Reconnecting without a password. Links, especially on Android, often open in a temporary in-app browser session instead of the full browser, so the session cookie is lost. Maybe store a token in local storage?

### To investigate (spike before committing)

- [ ] Rocket (Datastar web components) — beta, check licensing (Pro?) and whether it earns its place
- [ ] Load test: how many button smashers can one server take?

## Development

- `task start` — templ watch + hot reload on http://localhost:8080
- `task build` — production binary in `bin/main`
- `task test` — run all tests
- `task check` — everything a PR must pass (formatting, vet, `go fix`, lint, tests); `task fmt` fixes formatting
- `task screenshots` — screenshots of every screen (phone and big screen, presenter view) into `tmp/screenshots`, for checking UI changes; needs chromium or google-chrome. See `screenshots.sh` for comparing two runs
- Requires [golangci-lint](https://golangci-lint.run) v2 on your PATH, built with your Go version (`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`)
- `*_templ.go` files are generated and not in git; `task build` and `task start` generate them. Plain `go build` needs `go tool templ generate` first.
- Open `/local-dev-key/` (or `/remote-dev-key/`) to see the deck; `/` only shows "you need a key"

Environment variables (put them in `.env`, which is not in git). All except `PRESENTER_KEY` fall back to dev values with a warning:

- `ATTENDEE_LOCAL_KEY`, `ATTENDEE_REMOTE_KEY` — the URL keys that give access to the deck
- `PRESENTER_KEY` — unlocks presenter mode at `/presenter`
- `SESSION_SECRET` — signs the session cookies
- `PORT` — HTTP port, default 8080
