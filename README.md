# Frontiers Bergen — Christmas Meetup App

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

## Workshop

[Workshop: Let's Build Our Christmas Meetup App!](https://www.meetup.com/frontiers-bergen/events/317219351/)
— Thursday 12 November 2026, 17:30, Solea AS, Bergen.

Attendees hack on this codebase together: collaborative mini-games, easter eggs, festive surprises.
The workshop is for fun and learning; it is fine if it produces nothing tangible.
That means the app must be ready for the juleavslutning without depending on what gets built at the workshop.

Forked from the January 2026 meetup deck ("Next.js + Firebase vs Go + Datastar").

## Checklist

### Before the workshop (must have by 12 Nov)

- [ ] Decide the project name
- [ ] Create the public GitHub repository under Frontiers and push (done by someone with access; we work locally until then)
- [ ] Rename Go module and titles away from `frontiers-meetup-january-2026`
- [ ] Remove `presentation.db*` from git and un-comment the ignore rules
- [ ] Apply `schema.sql` automatically at startup so a fresh clone runs without a DB file
- [ ] Move the `poll_votes` table from the comment in `pages/root/poll.templ` into `schema.sql`
- [ ] One-command setup for attendees (`task start`), documented in this README
- [ ] Verify a fresh clone works on Linux, macOS and Windows
- [ ] Update Go (go.mod says 1.25.4) and dependencies (templ, datastar-go, nats)
- [ ] Update vendored `static/datastar.js` (currently v1.0.0-RC.7) to match datastar-go
- [ ] Rewrite `AGENTS.md` with the new engineering guidelines; make `CLAUDE.md` point to it
- [ ] Replace `agent_project.md` / `project_summary.md` with docs for the new purpose
- [ ] A short "how to add your own mini-game" guide (the extension point attendees will use)
- [ ] First example multiplayer game that shows the pattern end to end
- [ ] Prepare "good first tasks" for attendees (GitHub issues)
- [ ] Opening demo + codebase walkthrough slides for Gerhard
- [ ] CI/CD like conorganizer: test + build on every PR, each PR gets its own preview environment, main deploys to the big screen
- [ ] Slides written in markdown, rendered as part of the existing build (see `MdToHTML` in conorganizer)

### Code health

- [ ] Stop ignoring errors (`NewEmojiBalloonService`, `NewDemoHandler`, `http.Error` after SSE has started)
- [ ] Read the session secret from env instead of the hardcoded `"session-secret"`
- [ ] Remove leftover Todo-example code (`models/state.go`, Todo state in `helpers/nats_helpers.go`)
- [ ] Remove unused slides that are not in the deck
- [ ] Test setup based on conorganizer (`testutil/bdd`, `testutil/templtest`) and a first behavior test in given-when-then form
- [ ] Refactor the CSS
- [ ] Add secret keys to prevent the internet from accessing the services (carried over)
- [ ] Setup styling (carried over — `gotailwind` is a tool dependency but unused?)

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

### To investigate (spike before committing)

- [ ] Rocket (Datastar web components) — beta, check licensing (Pro?) and whether it earns its place
- [ ] Load test: how many button smashers can one server take?

## Development

- `task start` — templ watch + hot reload on http://localhost:8080
- `task build` — production binary in `bin/main`
- Open `/local-dev-key/` (or `/remote-dev-key/`) to see the deck; `/` only shows "you need a key"
