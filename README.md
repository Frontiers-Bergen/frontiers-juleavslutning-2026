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
- Scaling beyond one room. Everyone is at the venue; there is no remote participation or live stream.
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
- [ ] Draft a programme proposal for the juleavslutning — first draft in `planning/juleavslutning_programme.md`; details to be settled in a meeting after the workshop
- [x] Remove remote participation: in person only, no live stream (one invite key, polls not split by audience)
- [x] Rename Go module to `github.com/frontiers-bergen/frontiers-juleavslutning-2026`
- [x] Remove `presentation.db*` from git and un-comment the ignore rules
- [x] Apply `schema.sql` automatically at startup so a fresh clone runs without a DB file (no migrations; decided against goose)
- [x] Move the `poll_votes` table from the comment in `pages/root/poll.templ` into `schema.sql`
- [x] One-command setup for attendees (`go tool task start`), documented under Getting started
- [x] Fresh clone verified on Linux; macOS/Windows pitfalls fixed by reading (`.exe` via `{{exeExt}}`, LF line endings via `.gitattributes`, portable browser tools)
- [ ] Someone tries a fresh clone on macOS and on Windows before the workshop (Gerhard, weekend of 7–8 Nov); update `SETUP.md` with the findings
- [x] Update Go (1.27) and dependencies (templ, datastar-go, nats, sqlite, chi, toolbelt) and tools (task, air, dlv)
- [x] Update vendored `static/datastar.js` to v1.0.4 (with its source map), self-hosted in `static/`
- [x] Rewrite `AGENTS.md` with the new engineering guidelines; make `CLAUDE.md` point to it
- [x] Add `domain_dictionary.md` with the project's domain terms
- [x] Remove the January docs (`agent_project.md`, `project_summary.md`, `presentationOutline.md`); `README.md`, `AGENTS.md` and code comments replace them
- [ ] A short "how to add your own mini-game" guide (the extension point attendees will use)
- [ ] Two example mini-games that show the pattern end to end: one with only Go + SQLite + Datastar, one with a web component built with Rocket (web components only when needed)
- [ ] Prepare "good first tasks" for attendees (GitHub issues, label `good first issue`) — draft list in `planning/workshop_good_first_issues.md`; first one created: #41 smash milestones
- [ ] CI/CD like conorganizer: test + build on every PR, each PR gets its own preview environment, main deploys to the big screen (needed to merge PRs live at the workshop; Gerhard, weekend of 7–8 Nov)
- [ ] Opening demo + codebase walkthrough slides for Gerhard — first draft is the deck in `slides/` (13 slides incl. the app name vote); refine once the mini-games and CI/CD exist
- [x] QR code with the invite link on the welcome slide (server-rendered SVG at `/{inviteKey}/qr.svg`, big screen only)
- [ ] App name vote at the workshop: a poll with 3–4 candidates, the winner becomes the app name shown on the big screen and phones.
      Candidates so far: Rage Against the Debounce, Last Christmas I Gave You My Clicks, It's Tiiime, Smash the Halls.
      The name must make people laugh out loud, not just chuckle.

### At and after the juleavslutning

- [ ] Sound: Gerhard has a portable system (venue not confirmed yet)
- [ ] Back up the production database afterwards; ideas and feedback are analysed from it
- [x] Slides written in markdown: one `.md` file per content slide in `slides/`, embedded and rendered at startup with gomarkdown (as in conorganizer)

### Code health

- [x] **Fix: emoji smashes could be lost — the very bug this app exists to fix.** Counts moved to SQLite (atomic increment), one NATS message per smash gives one balloon per smash, unknown emojis are rejected. (Datastar request cancellation was suspected too; `task browser-check` showed smashes are counted either way, so disabling it is only defensive.) Tests in `emoji_reactions_test.go`.

- [x] Stop ignoring errors in `main.go`, the demo handler and the root/presenter pages (`http.Error` after SSE has started)
- [ ] Stop ignoring errors in templates (`{{ x, _ := GetCurrentSlideIndex(db) }}`); needs data loaded before rendering — part of the architecture refactor
- [x] Clean up the emoji SSE handlers together with the emoji counting fix (the `services` package is gone)
- [x] Read the session secret from `SESSION_SECRET` instead of the hardcoded `"session-secret"`
- [x] Replace the Todo-example state and the `helpers` package that drove live updates: one `deck.changed` NATS message now re-renders every open page (`pages/root/live_updates.go`); the KV bucket is gone
- [x] Remove unused slides that are not in the deck, and unreferenced images
- [x] Replace the January deck with a placeholder deck: a welcome slide and the app name poll with its results
- [x] Remove the January `/demo` page (`handler/`, `services/demo_service.go`, `pages/demo.templ`, `components/demo.templ`)
- [x] BDD helper (`bdd/bdd.go`, from conorganizer) and first behavior tests (`database_test.go`)
- [x] Access tests (`access_test.go`): wrong invite keys are redirected, the invite key opens the deck, a wrong or missing presenter key gives no control
- [x] Safety net for polls before the CSS refactor (`poll_voting_test.go`): changing a vote replaces it, every person counts once, unknown options are rejected
- [ ] Bring over conorganizer's `testutil/templtest` when a test needs to render a component on its own (not needed yet: HTTP tests read semantic `data-` attributes)
- [x] Fix: SQLite pragmas were set on one pooled connection only, so concurrent writes failed with `SQLITE_BUSY`
- [x] Refactor the CSS: `@layer tokens, base, layout, features` in `static/index.css`, feature CSS in `<style>` blocks inside the `.templ` files, no inline styles, dead classes and unused CDN fonts/tippy.js removed (pixel-identical before/after)
- [x] Mobile first pass: poll and results use one column on phones (no more mid-word breaks), less slide padding on phones, thumb-sized reaction buttons
- [x] Confirm the CSS approach with Torstein (coldasice): feature CSS in `.css` files next to the `.templ` files, served by `feature_styles.go`
- [x] **Fix: any attendee can change the presenter's slide.** `POST /{inviteKey}/root/api/slides/set` does not check presenter mode, so anyone with the invite link can move the deck (curl or devtools). Start with a test.
- [ ] Add secret keys to prevent the internet from accessing the services (carried over)
- [x] Remove all Tailwind leftovers from the northstar example (`gotailwind` tool, utility classes); the project uses plain CSS

- [x] Presenter controls: on the first and last slide the previous/next box is visibly empty and not clickable (it looked like a clickable "End" slide)
- [x] Presenter view: the presenter notes box overlapped the top of the reaction bar (stale `--reactions-min-height` after the thumb-sized buttons)

- [x] Look like Frontiers: palette, font and heading style from frontiersbergen.dev
- [x] Our own style guide, `STYLE_GUIDE.md` (no official one will arrive before the juleavslutning)
- [x] Accessibility: near-black text on the yellow poll option (6.7:1; white was 2.4:1)
- [x] Accessibility: emoji balloons respect `prefers-reduced-motion` (checked by `task browser-check`)

### Features (AhaSlides-inspired)

- [x] Multiple choice poll (Kahoot style)
- [x] Emoji reactions without debouncing
- [ ] Retro board with sticky notes (the juleavslutning retrospective use case)
- [ ] Word cloud
- [ ] Audience Q&A with upvotes
- [ ] Quiz with scores and leaderboard
- [ ] Spinner wheel

### Mini-game ideas

- [ ] Tug of war: two halves of the room (teams assigned per phone)
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

- [x] Rocket (Datastar web components): free since Datastar v1.0.4. Decision: no migration of `<emoji-balloon>` (fire-and-forget animation, nothing for Rocket to improve, double bundle size); try Rocket in the web component mini-game demo
- [ ] Load test: how many button smashers can one server take?

## Getting started

You need [Go](https://go.dev/dl/) and git. Linux, macOS and Windows all work. Everything else (templ, Task, air) comes in as a Go tool, so there is nothing more to install to run the app.

```sh
git clone https://github.com/frontiers-bergen/frontiers-juleavslutning-2026.git
cd frontiers-juleavslutning-2026
go tool task start
```

Then open http://localhost:8080/dev-key/ and edit any `.templ`, `.css` or `.go` file; the page reloads after a save.

- Presenter mode: create a `.env` file with `PRESENTER_KEY=something`, restart, open http://localhost:8080/presenter/ and enter it.
- Try it on your phone: open `http://<your computer's LAN address>:8080/dev-key/` on the same wifi.
- Older Go? `go.mod` asks for Go 1.27; with Go 1.21 or newer, the right version is downloaded automatically on first run.
- Before opening a pull request: `go tool task check` (needs golangci-lint, see below). Read `AGENTS.md` first; it holds the rules for people and AI agents alike.
- Stuck? `SETUP.md` covers installing Go per OS, busy ports, the phone on the LAN and other common problems; it is written so an AI agent can walk you through it.
- Optional tools: [golangci-lint](https://golangci-lint.run) for `task check`; Chrome/Chromium (and Node for `task browser-check`, bash for `task screenshots`) for the browser tools.

## Development

The commands below are `go tool task <name>`; with Task installed globally, plain `task <name>` works too.

- `task start` — templ watch + hot reload on http://localhost:8080
- `task build` — production binary in `bin/main` (`bin/main.exe` on Windows)
- `task test` — run all tests
- `task check` — everything a PR must pass (formatting, vet, `go fix`, lint, tests); `task fmt` fixes formatting
- `task screenshots` — screenshots of every screen (phone and big screen, presenter view) into `tmp/screenshots`, for checking UI changes; needs bash and Chrome/Chromium. See `browser-tools/screenshots.sh` for comparing two runs
- `task browser-check` — clicks real buttons in headless Chromium on a simulated slow network: fast smashing, a live slide change, voting; needs Node and Chrome/Chromium
- Requires [golangci-lint](https://golangci-lint.run) v2 on your PATH, built with your Go version (`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`)
- `*_templ.go` files are generated and not in git; `task build` and `task start` generate them. Plain `go build` needs `go tool templ generate` first.
- Open `/dev-key/` to see the deck; `/` only shows "you need a key"
- Runtime data lives next to the app: `presentation.db` (SQLite) and `nats-data/` (embedded NATS JetStream). Both are ignored by git; delete both to start fresh

Environment variables (put them in `.env`, which is not in git). All except `PRESENTER_KEY` fall back to dev values with a warning:

- `INVITE_KEY` — the secret in the invite link (`/<INVITE_KEY>/`) that gives access to the deck
- `PRESENTER_KEY` — unlocks presenter mode at `/presenter`
- `SESSION_SECRET` — signs the session cookies
- `PORT` — HTTP port, default 8080
