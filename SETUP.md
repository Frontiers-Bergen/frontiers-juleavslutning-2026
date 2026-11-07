# Setup guide (for agents helping someone get started)

Use this when an attendee asks for help running the app. Work through it in order, one step at a time,
and check each result before moving on. The short version for people is under "Getting started" in README.md.

Status: verified on Linux. The macOS and Windows notes are known pitfalls, still to be confirmed on real machines;
update this file with anything that turns out different.

## 1. Find out where they are

Ask which operating system they use (macOS, Windows, Linux) and, on Windows, which terminal (PowerShell, Git Bash, WSL).
The commands below work in all of them unless marked otherwise.

## 2. Go and git

```sh
go version    # needs go1.21 or newer; go.mod asks for 1.27 and newer Go downloads it automatically
git --version
```

Install if missing:

| OS | Go | git |
|---|---|---|
| macOS | `brew install go`, or the installer from https://go.dev/dl/ | `xcode-select --install`, or `brew install git` |
| Windows | `winget install GoLang.Go`, or the MSI from https://go.dev/dl/ | `winget install Git.Git` |
| Linux | the distribution's package if it is 1.21+, otherwise the tarball from https://go.dev/dl/ | the distribution's package |

Open a new terminal after installing, so the PATH is updated.

If `go version` shows an old version and the automatic download fails (no internet, a proxy blocking
proxy.golang.org): install Go 1.27 directly from https://go.dev/dl/.

## 3. Clone and start

```sh
git clone https://github.com/frontiers-bergen/frontiers-juleavslutning-2026.git
cd frontiers-juleavslutning-2026
go tool task start
```

- The first run downloads modules and tools; a minute or two is normal. On Windows, antivirus scanning can make it slower.
- Success looks like a log line `starting HTTP server` and http://localhost:8080/dev-key/ showing the deck.
- `/` on its own shows "You need an invite link"; that is correct, the deck lives under `/dev-key/`.
- Editing a `.templ`, `.css`, `.go` or `.md` file rebuilds and restarts automatically; reload the browser.
- Ctrl+C stops everything.

## 4. Common problems

**"address already in use" / port 8080 busy.** Another program uses the port. Start on another one:
- bash / zsh / Git Bash: `PORT=8081 go tool task start`
- PowerShell: `$env:PORT=8081; go tool task start`

**Votes fail or the page errors after pulling new code.** The database schema changed (there are no migrations).
Stop the app, delete `presentation.db`, start again.

**`go tool task check` reports every file as unformatted (Windows).** The files were checked out with CRLF line endings.
`.gitattributes` prevents this for new clones; for an older clone: `git add --renormalize .` or clone again.

**`golangci-lint: command not found` in `task check`.** It is a separate tool:
`brew install golangci-lint` (macOS) or `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`
(any OS; make sure `$(go env GOPATH)/bin` is on the PATH). If it reports a typecheck error about "export data",
an older copy built with a different Go version comes first on the PATH; remove that one.

**Errors mentioning `_templ.go` files.** Those are generated and not in git. `go tool task start` and `task build`
generate them; for a plain `go build`, run `go tool templ generate` first.

## 5. Presenter mode

Create a `.env` file in the repository root with `PRESENTER_KEY=something`, restart, open http://localhost:8080/presenter/
and enter the key. Then open the deck in the same browser to get the slide list and next/previous controls.

## 6. Try it on a phone

The phone must be on the same network as the computer. Find the computer's LAN address:
- macOS: `ipconfig getifaddr en0` (wifi)
- Windows: `ipconfig`, the "IPv4 Address" line
- Linux: `hostname -I` or `ip -4 addr`

Open `http://<address>:8080/dev-key/` on the phone. If it does not load:
- Windows Defender Firewall asks the first time the app listens; allow it on private networks.
- macOS asks to accept incoming connections when its firewall is on; allow it.
- Some venue or guest wifi networks isolate devices from each other. Then use the computer's browser only, or let
  the computer join the phone's hotspot.

## 7. Before the first pull request

Point them to `AGENTS.md` (the rules for people and AI agents) and to `go tool task check`.

To get a preview of their change, they need write access to the repository: they tell Gerhard (or another organiser)
their GitHub username and accept the invitation (in the email, or at
https://github.com/frontiers-bergen/frontiers-juleavslutning-2026/invitations). Then: a branch, `git push -u origin <branch>`,
and a pull request. CI runs the checks; after an organiser approves the deployment, the preview link appears in the pull request.
Without access, a pull request from a fork works too, but gets no preview.
Optional tools: Chrome or Chromium for `task screenshots` (needs bash) and `task browser-check` (needs Node 22+).
