# Frontiers Juleavslutning 2026

A live, multiplayer audience app for Frontiers Bergen: slides on the big screen, and everyone's phone as a controller
for emoji reactions, polls and mini-games.

## Why

We want the people at the Frontiers juleavslutning to have fun.

This started because our old retro tool debounced emoji reactions, so our button smashes didn't count.
We would rather build an entire app than settle for that — or for PowerPoint.
**Every button smash deserves to count.**

**Success** is the people at the juleavslutning having fun.

**How:** we build it together as a community, with tools simple enough that anyone can contribute in an evening.

### Not goals

- Replacing AhaSlides in general. A feature is built because it makes the juleavslutning more fun, not because another tool has it.
- Showing off the stack or any specific technology.
- Multi-tenancy, accounts or logins.
- Scaling beyond one room. Everyone is at the venue; there is no remote participation or live stream.
- Exporting ideas or feedback. It happens once; we read it straight from the database after the event.

## The workshop

[Let's Build Our Christmas Meetup App!](https://www.meetup.com/frontiers-bergen/events/317219351/)
Thursday 12 November 2026, 17:30, Solea AS, Bergen.

We hack on this codebase together: mini-games, easter eggs, festive surprises. It is for fun and learning, and it is
fine if nothing ships. The app is ready for the juleavslutning without depending on what gets built at the workshop.

## Getting started

You need [Go](https://go.dev/dl/) 1.21 or newer and git, on Linux, macOS or Windows. Everything else comes in as a
Go tool, and the Go version this project asks for (1.27) is downloaded automatically on the first run.

```sh
git clone https://github.com/frontiers-bergen/frontiers-juleavslutning-2026.git
cd frontiers-juleavslutning-2026
go tool task start
```

Open http://localhost:8080/dev-key/ and edit any `.templ`, `.css` or `.go` file: the page reloads when you save.

- **On your phone:** open `http://<your computer's LAN address>:8080/dev-key/` on the same wifi.
- **As the presenter:** put `PRESENTER_KEY=something` in a `.env` file, restart, and enter it at http://localhost:8080/presenter/.
- **Stuck?** [`SETUP.md`](SETUP.md) covers installing Go, busy ports, phones on the LAN and other common problems.
  It is written so an AI agent can walk you through it.

## Contributing

1. Read [`AGENTS.md`](AGENTS.md). It holds the rules for people and AI agents alike.
2. Want to build a mini-game? [`MINI_GAMES.md`](MINI_GAMES.md) walks you through it, using the two example games.
3. Tell Gerhard your GitHub username to get write access, and accept the invitation
   (in the email, or at https://github.com/frontiers-bergen/frontiers-juleavslutning-2026/invitations).
4. Make a branch, run `go tool task check`, push, and open a pull request.
   CI runs the checks and adds a preview link to the pull request.

No access? A pull request from a fork works too, just without a preview.

## Commands

Run them as `go tool task <name>` (or plain `task <name>` with Task installed).

| Command | What it does |
| --- | --- |
| `start` | Hot reload on http://localhost:8080 |
| `test` | Run all tests |
| `check` | Everything a pull request must pass: formatting, vet, `go fix`, lint, tests. `fmt` fixes formatting |
| `build` | Production binary in `bin/main` (`bin/main.exe` on Windows) |
| `screenshots` | Every screen (phone, big screen, presenter) into `tmp/screenshots`, to check UI changes |
| `browser-check` | Clicks real buttons in headless Chromium on a simulated slow network |
| `loadtest` | Simulated phones smash a local copy of the app (`-- -phones 80`, `-- -scenario ramp`, `-- -server-cores 2`). Never run it against the deployed app: the server is shared |

What the tools need:

- `check` needs [golangci-lint](https://golangci-lint.run) v2, built with your Go version:
  `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`
- `screenshots` needs bash and Chrome or Chromium; `browser-check` needs Node and Chrome or Chromium.
- `*_templ.go` files are generated and not in git. `start` and `build` generate them; plain `go build` needs
  `go tool templ generate` first.

## Configuration

Put these in `.env` (not in git). All except `PRESENTER_KEY` fall back to dev values with a warning.

| Variable | Purpose |
| --- | --- |
| `INVITE_KEY` | The secret in the invite link (`/<INVITE_KEY>/`) that opens the deck |
| `PRESENTER_KEY` | Unlocks presenter mode at `/presenter` |
| `SESSION_SECRET` | Signs the session cookies |
| `PORT` | HTTP port, default 8080 |
| `HOST` | Address to listen on. Empty by default (all interfaces, so phones on the LAN can connect); the server uses `127.0.0.1` behind Caddy |

All state lives in `presentation.db` (SQLite, not in git): delete it to start fresh. NATS runs inside the app,
without a port, and keeps nothing between runs. `/` only says that you need a key; the deck is at `/dev-key/`.

## Where to read more

| File | What is in it |
| --- | --- |
| [`TODO.md`](TODO.md) | What is left to do, and what is done |
| [`AGENTS.md`](AGENTS.md) | Engineering principles, the page life cycle, project conventions |
| [`SETUP.md`](SETUP.md) | Installing and running, step by step |
| [`MINI_GAMES.md`](MINI_GAMES.md) | How to add a mini-game |
| [`STYLE_GUIDE.md`](STYLE_GUIDE.md) | Colours, fonts, layout and tone |
| [`domain_dictionary.md`](domain_dictionary.md) | The words we use, like juleavslutning |
| [`deploy/README.md`](deploy/README.md) | CI/CD, the previews and the server |

Forked from the January 2026 meetup deck ("Next.js + Firebase vs Go + Datastar").
