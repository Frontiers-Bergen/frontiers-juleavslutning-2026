# Get it running

```sh
git clone https://github.com/frontiers-bergen/frontiers-juleavslutning-2026
cd frontiers-juleavslutning-2026
go tool task start
```

Open `localhost:8080/dev-key/`; on your phone, use your laptop's address instead of localhost.

Stuck? Ask your AI agent to follow SETUP.md.

Notes:

- You need Go and git; everything else comes in as Go tools. The first run downloads modules: a minute or two.
- Guest wifi often keeps phones from reaching laptops; then use the laptop's browser.
- Linux with ufw (Omarchy) blocks the port: SETUP.md has the line to open it for the local network.
