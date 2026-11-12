# To do

What is left, and what is done. Moved out of `README.md` on the day of the workshop (12 Nov 2026), so we can pick
the work up again afterwards. Open items come first; the done items below them are kept as history, because they
explain decisions and measurements that are not obvious from the code.

Every item serves the Why in `README.md`: the people at the juleavslutning having fun.

## Open

### After the workshop

- [ ] Someone tries a fresh clone on macOS (planned for 7–8 Nov); update `SETUP.md` with the findings
- [ ] Snow globe: try it on real phones, an iPhone in particular (the "Shake for snow" permission button); tune the shake threshold and how much snow
- [ ] Workshop slides for Gerhard: the intro (under 15 minutes, 18:00) and show and tell (20:30), in `slides/`; text reviewed 8 Nov
- [ ] Maybe later: the live update diagram lights up for real whenever someone smashes (needs a templ slide instead of markdown)

### The juleavslutning

- [ ] Draft a programme proposal: first draft in `planning/juleavslutning_programme.md`; details to be settled in a meeting after the workshop
- [ ] Sound: Gerhard has a portable system (venue not confirmed yet)
- [ ] Back up the production database afterwards; ideas and feedback are analysed from it
- [ ] Remove the workshop collaborators when the project ends (`deploy/README.md`)
- [ ] When the server is cleaned up before next Regncon: the previews of #77 and #78 were left running (GitHub's Actions outage on 9 Nov stopped their cleanup; such runs cannot be re-run)

### Features (AhaSlides-inspired)

- [ ] Retro board with sticky notes (the juleavslutning retrospective use case)
- [ ] Word cloud
- [ ] Audience Q&A with upvotes
- [ ] Quiz with scores and leaderboard
- [ ] Spinner wheel

### Mini-game ideas

- [ ] Button-mash reindeer race (#70)
- [ ] Decorate the tree together (#71)
- [ ] Snowball fight (#72)
- [ ] Avatar builder (#73)
- [ ] Spaceship snowball shooter
- [ ] Collaborative pixel canvas (r/place, Christmas edition)
- [ ] Hidden easter eggs

### Code health and architecture

- [ ] Balloons: optimise further for slow phones, without limiting how many: crashing old phones with balloon spam is part of the fun. Biggest remaining cost: style for every new balloon
- [ ] Bring over conorganizer's `testutil/templtest` when a test needs to render a component on its own (not needed yet: HTTP tests read semantic `data-` attributes)
- [ ] Refactor the architecture and design once a few more features exist and the shape of the project shows.
- [ ] Discuss as part of the refactor: should each mini-game live in its own folder (its templ, CSS, tests, and maybe its own table and route registration)? Tied to the merge conflicts in the shared files (`schema.sql`, `active_slide.templ`, `root_index.templ`)
      Use a Reverse Conway's Law approach: design the code structure to match how we want people to work.
      Example: many workshop attendees build mini-games in parallel and PRs are merged live, so each game should be a self-contained feature folder,
      and shared hotspots like the slide list in `pages/root/active_slide.templ` should not force everyone to edit the same lines.

### To discuss

- [ ] Demo data in a separate `.sql` file, if we need it
- [ ] Reconnecting without a password. Links, especially on Android, often open in a temporary in-app browser session instead of the full browser, so the session cookie is lost. Maybe store a token in local storage?

## Done

### Before the workshop

- [x] Create the public GitHub repository: [frontiers-bergen/frontiers-juleavslutning-2026](https://github.com/frontiers-bergen/frontiers-juleavslutning-2026)
- [x] Rename the repo to `frontiers-juleavslutning-2026` (explicit for developers; the fun app name is decided separately)
- [x] Get write access to the new repo and move the `christmas-app` branch there from `Regncon/frontiers-meetup-january-2026`
- [x] Remove remote participation: in person only, no live stream (one invite key, polls not split by audience)
- [x] Rename Go module to `github.com/frontiers-bergen/frontiers-juleavslutning-2026`
- [x] Remove `presentation.db*` from git and un-comment the ignore rules
- [x] Apply `schema.sql` automatically at startup so a fresh clone runs without a DB file (no migrations; decided against goose)
- [x] Move the `poll_votes` table from the comment in `pages/root/poll.templ` into `schema.sql`
- [x] One-command setup for attendees (`go tool task start`), documented under Getting started
- [x] Fresh clone verified on Linux; macOS/Windows pitfalls fixed by reading (`.exe` via `{{exeExt}}`, LF line endings via `.gitattributes`, portable browser tools)
- [x] Fresh clone verified on Windows 11 (12 Nov): findings in `SETUP.md` (admin prompt, PATH, firewall on Public wifi); `screenshots` and `browser-check` fixed for Windows
- [x] Update Go (1.27) and dependencies (templ, datastar-go, nats, sqlite, chi) and tools (task, air, dlv)
- [x] Datastar v1.0.4 with Rocket (`static/datastar-rocket.js`, with its source map), self-hosted in `static/`
- [x] Rewrite `AGENTS.md` with the new engineering guidelines; make `CLAUDE.md` point to it
- [x] Add `domain_dictionary.md` with the project's domain terms
- [x] Remove the January docs (`agent_project.md`, `project_summary.md`, `presentationOutline.md`); `README.md`, `AGENTS.md` and code comments replace them
- [x] Mini-game without JavaScript: tug of war, two random, even teams in the logo's colours, the team that smashes the most emojis wins, presenter-controlled countdown (`pages/root/tug_of_war.templ`)
- [x] Mini-game with a web component built with Rocket: shake the snow globe (`static/snow-globe.js`, `pages/root/snow_globe.templ`); shakes swirl snow on every screen, subtle on phones
- [x] Short "how to add your own mini-game" guide, pointing at the two example games (`MINI_GAMES.md`)
- [x] Good first issues for attendees (label `good first issue`): #41 smash milestones and four mini-games, #70 reindeer race, #71 decorate the tree, #72 snowball fight, #73 avatar builder (`planning/workshop_good_first_issues.md`)
- [x] CI: `task check` and `task browser-check` on every PR (`.github/workflows/ci-cd.yml`)
- [x] CD: a preview per PR at `juleavslutning-<PR>.lekeplassen.regncon.no`, main on the big screen at https://juleavslutning.lekeplassen.regncon.no, previews removed on close (`deploy/`); tested end to end
- [x] Workshop slides: visual support: the repository QR code, the live update diagram (with the Go gopher), icons and a balloon chart on What we learned, icons on The stack, a file tree on Add your own mini-game, idea tiles on Build what you want, and a green grump on How to contribute
- [x] Slide illustrations: an `Illustration:` line puts a picture beside the text (under it on a phone); our own drawings for the origin story, why, show and tell, and the green grump (`static/illustrations/`)
- [x] Slides fit the big screen: at 1280×720 a slide with a lot of text no longer pushes its heading down and its last lines out of sight (shrinking spacers, `safe center`, no double room for the reactions bar)
- [x] QR code with the invite link on the welcome slide (server-rendered SVG at `/{inviteKey}/qr.svg`, big screen only)
- [x] Workshop contributions: attendees who want to push are added as collaborators with write access (forks get no preview: GitHub gives them no secrets). How: `deploy/README.md`
- [x] Decided (7 Nov): no deploy approval or branch protection for workshop collaborators; trust and review instead ("Collaborators and trust" in `deploy/README.md`)
- [x] App name vote: dropped from the workshop intro for now (it added little and took a lot); the poll and its results stay at the end of the deck, outside the talk
      Candidates so far: Rage Against the Debounce, Last Christmas I Gave You My Clicks, It's Tiiime, Smash the Halls.
      The name must make people laugh out loud, not just chuckle.
- [x] Slides written in markdown: one `.md` file per content slide in `slides/`, embedded and rendered at startup with gomarkdown (as in conorganizer)

### Code health

- [x] **Fix: emoji smashes could be lost — the very bug this app exists to fix.** Counts moved to SQLite (atomic increment), one NATS message per smash gives one balloon per smash, unknown emojis are rejected. (Datastar request cancellation was suspected too; `task browser-check` showed smashes are counted either way, so disabling it is only defensive.) Tests in `emoji_reactions_test.go`.
- [x] Stop ignoring errors in `main.go`, the demo handler and the root/presenter pages (`http.Error` after SSE has started)
- [x] Stop ignoring errors in templates: the slide, the presenter's navigation, the slide list and the poll each show what failed, and reading the current slide clamps it to the deck
- [x] Clean up the emoji SSE handlers together with the emoji counting fix (the `services` package is gone)
- [x] Read the session secret from `SESSION_SECRET` instead of the hardcoded `"session-secret"`
- [x] Replace the Todo-example state and the `helpers` package that drove live updates: one `deck.changed` NATS message now re-renders every open page (`pages/root/live_updates.go`); the KV bucket is gone
- [x] Remove unused slides that are not in the deck, and unreferenced images
- [x] Replace the January deck with a placeholder deck: a welcome slide and the app name poll with its results
- [x] Remove the January `/demo` page (`handler/`, `services/demo_service.go`, `pages/demo.templ`, `components/demo.templ`)
- [x] BDD helper (`bdd/bdd.go`, from conorganizer) and first behavior tests (`database_test.go`)
- [x] Access tests (`access_test.go`): wrong invite keys are redirected, the invite key opens the deck, a wrong or missing presenter key gives no control
- [x] Safety net for polls before the CSS refactor (`pages/root/poll_voting_test.go`): changing a vote replaces it, every person counts once, unknown options are rejected
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

### Features and mini-games

- [x] Multiple choice poll (Kahoot style)
- [x] Emoji reactions without debouncing
- [x] Tug of war: two even teams, drawn again every round among the phones watching (user test: 6 people felt the first-come teams were unfair)
- [x] Tug of war: your team in heading-size letters above your own end of the rope (user test: hard to see your side); the big screen labels both ends; the start button sits in the heading row, clear of the presenter notes
- [x] Presenter view: the notes covered the slide on most slides, and the bottom of a slide hid under the reactions bar. Now the notes sit below the slide, the presenter's copy of the slide uses smaller type and less padding, and the stage scrolls if a slide with long notes still does not fit (`static/index.css`)

### Investigations and load tests

- [x] Rocket (Datastar web components): free since Datastar v1.0.4. Decision: no migration of `<emoji-balloon>` (fire-and-forget animation, nothing for Rocket to improve, double bundle size); try Rocket in the web component mini-game demo
- [x] Load test: how many button smashers can one server take? `go tool task loadtest`, local only (the Hetzner server is shared). First results on a 20-core laptop, 6 taps/s per phone: 40 phones missed the 100 ms goal (P99 340 ms, 6 cores busy, 1.4 GB memory); the findings below fix most of it
- [x] Fix: SQLite connections were opened and closed under load (database/sql keeps 2 idle by default); opening one applies the pragmas and parses the schema, half the CPU at 40 phones. A fixed pool of 32 (`database.go`): 40 phones P99 19 ms on 3 cores, 80 phones P99 35 ms
- [x] Memory: every live stream has its own Brotli compressor, and the default 4 MB window cost about 36 MB per phone (1.4 GB at 40 phones, on a shared server). A 256 KB window (`compressedStream` in `pages/root/live_updates.go`): 330 MB at 40 phones, same bytes and CPU
- [x] Streams take what is already waiting along: the emoji stream sends waiting smashes in one patch (one balloon each, the counts read once), the page stream renders once for waiting changes. Load test: at 160 phones 30% of the balloons were lost (NATS dropped them for streams that fell behind), now 100% arrive; 40 phones on 2 cores went from P99 244 ms to 9 ms
- Load test results after both fixes (20-core laptop, the load generator on the same machine, 6 taps/s per phone): 40 phones P99 10–35 ms, about 3 cores, 330 MB; 160 phones P99 35–60 ms; 320 phones just under 100 ms; 640 phones break (P99 460 ms, 4 GB). With the app limited to 2 cores, 40 phones missed the goal (P99 145 to 244 ms) until the streams took what was already waiting along; now P99 9 ms
- [x] Fix: balloons that arrived while a window was not drawn (another workspace) all took off at once when it was drawn again; they also flew twice as fast on 120 Hz screens. They now fly by the clock with Web Animations on transform (`static/emoji-balloon.js`), which also cut the main-thread work with 400 balloons in the air by 58% (JavaScript −82%, layout −93%)
- [x] Fix: a snowfall that arrived while a window was not drawn waited, then started with the others when the window was drawn again, cut off by its removal; its animations now start at the moment it arrives (`snowfallInit` in `pages/root/snow_globe.templ`)
- [x] Balloons: read the buttons' positions once per frame instead of once per balloon (layout -57%, main-thread work -11 to -16% with 400 to 1500 balloons). Kept: the shadow DOM per balloon, which measured cheaper than none, because Datastar inspects everything added to `<body>`
- [x] A phone that leaves while its stream is sending no longer logs an ERROR ("context canceled"): a normal goodbye is logged at debug level (`logStreamFailure` in `pages/root/live_updates.go`)
