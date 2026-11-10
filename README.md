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
- [x] Datastar v1.0.4 with Rocket (`static/datastar-rocket.js`, with its source map), self-hosted in `static/`
- [x] Rewrite `AGENTS.md` with the new engineering guidelines; make `CLAUDE.md` point to it
- [x] Add `domain_dictionary.md` with the project's domain terms
- [x] Remove the January docs (`agent_project.md`, `project_summary.md`, `presentationOutline.md`); `README.md`, `AGENTS.md` and code comments replace them
- [x] Mini-game without JavaScript: tug of war, two random, even teams in the logo's colours, the team that smashes the most emojis wins, presenter-controlled countdown (`pages/root/tug_of_war.templ`)
- [x] Mini-game with a web component built with Rocket: shake the snow globe (`static/snow-globe.js`, `pages/root/snow_globe.templ`); shakes swirl snow on every screen, subtle on phones
- [ ] Snow globe: try it on real phones, an iPhone in particular (the "Shake for snow" permission button); tune the shake threshold and how much snow
- [x] Short "how to add your own mini-game" guide, pointing at the two example games (`MINI_GAMES.md`)
- [x] Good first issues for attendees (label `good first issue`): #41 smash milestones and four mini-games, #70 reindeer race, #71 decorate the tree, #72 snowball fight, #73 avatar builder (`planning/workshop_good_first_issues.md`)
- [x] CI: `task check` and `task browser-check` on every PR (`.github/workflows/ci-cd.yml`)
- [x] CD: a preview per PR at `juleavslutning-<PR>.lekeplassen.regncon.no`, main on the big screen at https://juleavslutning.lekeplassen.regncon.no, previews removed on close (`deploy/`); tested end to end
- [ ] Workshop slides for Gerhard: the intro (under 15 minutes, 18:00) and show and tell (20:30), in `slides/`; text reviewed 8 Nov
- [x] Workshop slides: visual support: the repository QR code, the live update diagram (with the Go gopher), icons and a balloon chart on What we learned, icons on The stack, a file tree on Add your own mini-game, idea tiles on Build what you want, and a green grump on How to contribute
- [ ] Maybe later: the live update diagram lights up for real whenever someone smashes (needs a templ slide instead of markdown)
- [x] QR code with the invite link on the welcome slide (server-rendered SVG at `/{inviteKey}/qr.svg`, big screen only)
- [x] Workshop contributions: attendees who want to push are added as collaborators with write access (forks get no preview: GitHub gives them no secrets). How: `deploy/README.md`
- [x] Decided (7 Nov): no deploy approval or branch protection for workshop collaborators; trust and review instead ("Collaborators and trust" in `deploy/README.md`)
- [x] App name vote: dropped from the workshop intro for now (it added little and took a lot); the poll and its results stay at the end of the deck, outside the talk
      Candidates so far: Rage Against the Debounce, Last Christmas I Gave You My Clicks, It's Tiiime, Smash the Halls.
      The name must make people laugh out loud, not just chuckle.

### At and after the juleavslutning

- [ ] Sound: Gerhard has a portable system (venue not confirmed yet)
- [ ] Back up the production database afterwards; ideas and feedback are analysed from it
- [ ] Remove the workshop collaborators when the project ends (`deploy/README.md`)
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
- [x] Refactor the CSS: `@layer tokens, base, layout, features` in `static/index.css`, feature CSS next to each feature (first `<style>` blocks, now `.css` files, see below), no inline styles, dead classes and unused CDN fonts/tippy.js removed (pixel-identical before/after)
- [x] Mobile first pass: poll and results use one column on phones (no more mid-word breaks), less slide padding on phones, thumb-sized reaction buttons
- [x] Confirm the CSS approach with Torstein (coldasice): feature CSS in `.css` files next to the `.templ` files, served by `feature_styles.go`
- [x] Nest all CSS (Torstein's wish, rule in `AGENTS.md`): `top_navigation.css` and `snow_globe.css` nested (pixel-identical); `static/index.css` already nests like conorganizer's
- [x] **Fix: any attendee can change the presenter's slide.** `POST /{inviteKey}/root/api/slides/set` does not check presenter mode, so anyone with the invite link can move the deck (curl or devtools). Start with a test.
- [x] Keep the internet out: the invite key opens the deck, the presenter key gives control, and the app listens on 127.0.0.1 behind Caddy (`HOST`)
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

- [x] Tug of war: two even teams, drawn again every round among the phones watching (user test: 6 people felt the first-come teams were unfair)
- [x] Tug of war: your team in heading-size letters above your own end of the rope (user test: hard to see your side); the big screen labels both ends; the start button sits in the heading row, clear of the presenter notes
- [x] Presenter view: the notes covered the slide on most slides, and the bottom of a slide hid under the reactions bar. Now the notes sit below the slide, the presenter's copy of the slide uses smaller type and less padding, and the stage scrolls if a slide with long notes still does not fit (`static/index.css`)
- [ ] Spaceship snowball shooter
- [ ] Collaborative pixel canvas (r/place, Christmas edition)
- [ ] Button-mash reindeer race (#70)
- [ ] Snowball fight (#72)
- [ ] Hidden easter eggs
- [ ] Decorate the tree together (#71)
- [ ] Avatar builder (#73)

### Later: architecture and design

- [ ] Refactor the architecture and design once a few more features exist and the shape of the project shows.
- [ ] Discuss as part of the refactor: should each mini-game live in its own folder (its templ, CSS, tests, and maybe its own table and route registration)? Tied to the merge conflicts in the shared files (`schema.sql`, `active_slide.templ`, `root_index.templ`)
      Use a Reverse Conway's Law approach: design the code structure to match how we want people to work.
      Example: many workshop attendees build mini-games in parallel and PRs are merged live, so each game should be a self-contained feature folder,
      and shared hotspots like the slide list in `pages/root/active_slide.templ` should not force everyone to edit the same lines.

### To discuss later

- [ ] Demo data in a separate `.sql` file, if we need it

- [ ] Reconnecting without a password. Links, especially on Android, often open in a temporary in-app browser session instead of the full browser, so the session cookie is lost. Maybe store a token in local storage?

### To investigate (spike before committing)

- [x] Rocket (Datastar web components): free since Datastar v1.0.4. Decision: no migration of `<emoji-balloon>` (fire-and-forget animation, nothing for Rocket to improve, double bundle size); try Rocket in the web component mini-game demo
- [x] Load test: how many button smashers can one server take? `go tool task loadtest`, local only (the Hetzner server is shared). First results on a 20-core laptop, 6 taps/s per phone: 40 phones missed the 100 ms goal (P99 340 ms, 6 cores busy, 1.4 GB memory); the findings below fix most of it
- [x] Fix: SQLite connections were opened and closed under load (database/sql keeps 2 idle by default); opening one applies the pragmas and parses the schema, half the CPU at 40 phones. A fixed pool of 32 (`database.go`): 40 phones P99 19 ms on 3 cores, 80 phones P99 35 ms
- [x] Memory: every live stream has its own Brotli compressor, and the default 4 MB window cost about 36 MB per phone (1.4 GB at 40 phones, on a shared server). A 256 KB window (`compressedStream` in `pages/root/live_updates.go`): 330 MB at 40 phones, same bytes and CPU
- [x] Streams take what is already waiting along: the emoji stream sends waiting smashes in one patch (one balloon each, the counts read once), the page stream renders once for waiting changes. Load test: at 160 phones 30% of the balloons were lost (NATS dropped them for streams that fell behind), now 100% arrive; 40 phones on 2 cores went from P99 244 ms to 9 ms
- Load test results after both fixes (20-core laptop, the load generator on the same machine, 6 taps/s per phone): 40 phones P99 10–35 ms, about 3 cores, 330 MB; 160 phones P99 35–60 ms; 320 phones just under 100 ms; 640 phones break (P99 460 ms, 4 GB). With the app limited to 2 cores, 40 phones missed the goal (P99 145 to 244 ms) until the streams took what was already waiting along; now P99 9 ms
- [x] Fix: balloons that arrived while a window was not drawn (another workspace) all took off at once when it was drawn again; they also flew twice as fast on 120 Hz screens. They now fly by the clock with Web Animations on transform (`static/emoji-balloon.js`), which also cut the main-thread work with 400 balloons in the air by 58% (JavaScript −82%, layout −93%)
- [x] Fix: a snowfall that arrived while a window was not drawn waited, then started with the others when the window was drawn again, cut off by its removal; its animations now start at the moment it arrives (`snowfallInit` in `pages/root/snow_globe.templ`)
- [x] Balloons: read the buttons' positions once per frame instead of once per balloon (layout -57%, main-thread work -11 to -16% with 400 to 1500 balloons). Kept: the shadow DOM per balloon, which measured cheaper than none, because Datastar inspects everything added to `<body>`
- [ ] Balloons: optimise further for slow phones, without limiting how many: crashing old phones with balloon spam is part of the fun. Biggest remaining cost: style for every new balloon
- [x] A phone that leaves while its stream is sending no longer logs an ERROR ("context canceled"): a normal goodbye is logged at debug level (`logStreamFailure` in `pages/root/live_updates.go`)

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
- Want your change on the big screen? Tell Gerhard your GitHub username: you get write access to this repository (accept the invitation in the email, or at https://github.com/frontiers-bergen/frontiers-juleavslutning-2026/invitations). Then make a branch, push it and open a pull request. CI runs the checks and adds a preview link to the pull request. No access? A pull request from a fork works too, just without a preview.
- Want to build a mini-game? `MINI_GAMES.md` walks you through it, using the two example games.
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
- `task loadtest` — simulated phones smash a local copy of the app during a tug of war round and report smash-to-screen time, lost smashes, CPU and memory. `task loadtest -- -phones 80`, `-- -scenario ramp` doubles the phones until it is no longer instant, `-- -server-cores 2` mimics a small server. Never run it against the deployed app: the server is shared
- Requires [golangci-lint](https://golangci-lint.run) v2 on your PATH, built with your Go version (`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`)
- `*_templ.go` files are generated and not in git; `task build` and `task start` generate them. Plain `go build` needs `go tool templ generate` first.
- Open `/dev-key/` to see the deck; `/` only shows "you need a key"
- Runtime data: `presentation.db` (SQLite, ignored by git; delete it to start fresh). Embedded NATS keeps nothing between runs: a temp directory per run, or `NATS_STORE_DIR` on the server

Environment variables (put them in `.env`, which is not in git). All except `PRESENTER_KEY` fall back to dev values with a warning:

- `INVITE_KEY` — the secret in the invite link (`/<INVITE_KEY>/`) that gives access to the deck
- `PRESENTER_KEY` — unlocks presenter mode at `/presenter`
- `SESSION_SECRET` — signs the session cookies
- `PORT` — HTTP port, default 8080
- `HOST` — address to listen on; default empty (all interfaces, so phones on the LAN can connect). The server uses `127.0.0.1` behind Caddy
- `NATS_STORE_DIR` — where embedded NATS keeps its data while running; default a temp directory removed on exit (the server uses a systemd runtime directory)
