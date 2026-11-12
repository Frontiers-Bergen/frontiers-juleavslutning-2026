# Setup guide (for agents helping someone get started)

Use this when an attendee asks for help running the app. Work through it in order, one step at a time,
and check each result before moving on. The short version for people is under "Getting started" in README.md.

Status: verified on Linux, and on Windows 11 with PowerShell and Git Bash (12 Nov 2026). The macOS notes are known
pitfalls, still to be confirmed on a real machine; update this file with anything that turns out different.

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

Open a new terminal after installing, so the PATH is updated. On Windows, restart the whole app the terminal runs in
(Windows Terminal, VS Code, the Claude desktop app): a new tab inherits the app's old PATH.

**Windows:** `winget install GoLang.Go` needs an admin prompt, which can open behind other windows. Unanswered, it gives
up after about two minutes and winget says "You cancelled the installation" (exit code 1602): run it again and click Yes.
Without admin rights, unpack the `.zip` from https://go.dev/dl/ into the user folder and add its `bin` to the user PATH.

If `go version` shows an old version and the automatic download fails (no internet, a proxy blocking
proxy.golang.org): install Go 1.27 directly from https://go.dev/dl/.

## 3. Clone and start

```sh
git clone https://github.com/frontiers-bergen/frontiers-juleavslutning-2026.git
cd frontiers-juleavslutning-2026
go tool task start
```

- The first run downloads modules and tools; several minutes is normal (on Windows 11, the first `go tool task check`
  took about 4 minutes, and the first `start` downloads air's own dependencies). Antivirus scanning and a crowded wifi
  make it slower, so do the first run before the workshop if you can.
- Success looks like a log line `starting HTTP server` and http://localhost:8080/dev-key/ showing the deck.
- `/` on its own shows "You need an invite link"; that is correct, the deck lives under `/dev-key/`.
- Editing a `.templ`, `.css`, `.go` or `.md` file rebuilds and restarts automatically; reload the browser.
  On Windows, air prints `Process Exit with Code: 1` on every restart: it stops the old app the hard way. Harmless.
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
(any OS; make sure `$(go env GOPATH)/bin` is on the PATH; the Windows Go installer adds it). If it reports a typecheck error about "export data",
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
- Windows Defender Firewall asks the first time the app listens. The popup can open behind other windows, and the app
  is blocked until it is answered. Allow the network type you are on: venue and guest wifi are usually **Public**, not
  Private (`Get-NetConnectionProfile` in PowerShell tells). After a dismissed popup the block stays and Windows does not
  ask again; allow `tmp\bin\main.exe` under "Allow an app through Windows Firewall".
- macOS asks to accept incoming connections when its firewall is on; allow it.
- Linux with ufw (on by default in some setups, such as Omarchy) blocks incoming connections. Allow the port for the
  local network only, for example `sudo ufw allow from 192.168.1.0/24 to any port 8080 proto tcp` (use the network
  part of the address above). `systemctl is-active ufw` tells whether it runs.
- Some venue or guest wifi networks isolate devices from each other. Then use the computer's browser only, or let
  the computer join the phone's hotspot.

## 7. Before the first pull request

Point them to `AGENTS.md` (the rules for people and AI agents) and to `go tool task check`.

To get a preview of their change, they need write access to the repository: they tell Gerhard (or another organiser)
their GitHub username and accept the invitation (in the email, or at
https://github.com/frontiers-bergen/frontiers-juleavslutning-2026/invitations). Then: a branch, `git push -u origin <branch>`,
and a pull request. CI runs the checks and adds the preview link to the pull request.
Without access, a pull request from a fork works too, but gets no preview.
Optional tools: Chrome or Chromium for `task screenshots` (needs bash) and `task browser-check` (needs Node 22+).
On Windows, run `screenshots` from Git Bash: in PowerShell, `bash` can be WSL's (`C:\Windows\System32\bash.exe`),
which has neither this Go nor Windows' Chrome. `browser-check` works from any terminal.
