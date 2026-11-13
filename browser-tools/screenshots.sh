#!/usr/bin/env bash
# Screenshots of the main screens, for checking UI changes visually (go tool task screenshots).
#
# Builds the app from this checkout, runs it on $PORT (default 8099, so it does not collide with
# `task start` on 8080) with a fresh database, casts a few votes, logs in a presenter, and captures
# every slide as an attendee on a phone (390x844) and on the big screen (1440x900), plus the presenter view.
#
# Compare two runs, e.g. before and after a CSS change, with ImageMagick:
#   compare -metric AE -fuzz 2% before/slide0-phone.png after/slide0-phone.png diff.png
#
# Usage: browser-tools/screenshots.sh [output-dir]   (default: tmp/screenshots)
# Requires: bash, curl, and Chromium or Google Chrome. On Windows, run it from Git Bash: in PowerShell `bash`
# can be WSL's, which has neither this Go nor Windows' Chrome.
set -euo pipefail

repo="$(cd "$(dirname "$0")/.." && pwd)"
mkdir -p "${1:-$repo/tmp/screenshots}"
out="$(cd "${1:-$repo/tmp/screenshots}" && pwd)" # absolute path; realpath -m is GNU-only
port="${PORT:-8099}"
base="http://localhost:$port"
browser="$(command -v chromium || command -v google-chrome-stable || command -v google-chrome || true)"
# macOS and Windows (Git Bash) install Chrome outside the PATH.
for installed in "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
	"/c/Program Files/Google/Chrome/Application/chrome.exe" \
	"/c/Program Files (x86)/Google/Chrome/Application/chrome.exe"; do
	if [ -z "$browser" ] && [ -x "$installed" ]; then
		browser="$installed"
	fi
done
if [ -z "$browser" ]; then
	echo "screenshots: Chromium or Google Chrome is required" >&2
	exit 1
fi

work="$(mktemp -d)"
(cd "$repo" && go tool templ generate -log-level error && go build -o "$work/app" ./cmd/juleavslutning)
cp -r "$repo/static" "$work/"
cd "$work"
# Loopback only: listening on the LAN makes Windows Firewall ask on every run (a new temp path each time).
HOST=127.0.0.1 PORT="$port" PRESENTER_KEY=screenshot-key ./app >server.log 2>&1 &
app_pid=$!
# `|| true`: on Windows kill is a hard stop, so wait reports 143, and set -e would skip the rm and fail the run.
trap 'kill $app_pid 2>/dev/null || true; wait $app_pid 2>/dev/null || true; rm -rf "$work"' EXIT
started=""
for _ in $(seq 1 40); do curl -s -o /dev/null "$base/" && started=yes && break; sleep 0.25; done
if [ -z "$started" ]; then
	echo "screenshots: the app did not start; it said:" >&2
	tail -20 server.log >&2
	exit 1
fi

curl -s -c presenter.jar -o /dev/null -d key=screenshot-key "$base/presenter/login"

# A few votes on the app name poll, so results slides are not empty. Harmless if the poll is gone.
vote() { # person option
	curl -s -c "$1.jar" -b "$1.jar" -o /dev/null "$base/dev-key/"
	curl -s -b "$1.jar" -o /dev/null -X POST "$base/dev-key/root/api/poll/vote?poll=app-name&option=$2"
}
vote a its-tiiime
vote b its-tiiime
vote c its-tiiime
vote d smash-the-halls
vote e rage-against-the-debounce

move_to_slide() { curl -s -b presenter.jar -o /dev/null -X POST "$base/dev-key/root/api/slides/set?index=$1"; }
shown_slide() { curl -s -b presenter.jar "$base/dev-key/" | grep -oE 'data-slide-index="[0-9]+"' | head -1 | grep -oE '[0-9]+'; }

shot() { # name url width height
	"$browser" --headless=new --disable-gpu --hide-scrollbars --user-data-dir="$work/profile" \
		--window-size="$3,$4" --virtual-time-budget=2500 --screenshot="$out/$1.png" "$2" >/dev/null 2>&1 || true
}

shot need-key-phone "$base/" 390 844
shot presenter-login-desktop "$base/presenter/" 1440 900

# Every slide, until the deck stops advancing.
slide=0
while :; do
	move_to_slide "$slide"
	[ "$(shown_slide)" = "$slide" ] || break
	shot "slide$slide-phone" "$base/dev-key/" 390 844
	shot "slide$slide-desktop" "$base/dev-key/" 1440 900
	slide=$((slide + 1))
done

# Log the browser profile in as presenter, then capture the presenter view of the first slide.
printf '<form method="post" action="%s/presenter/login"><input name="key" value="screenshot-key"></form><script>document.forms[0].submit()</script>' "$base" >login.html
login_page="$work/login.html"
# Chrome on Windows cannot open Git Bash paths (/tmp/...); without this the presenter shots silently show the attendee view.
if command -v cygpath >/dev/null; then
	login_page="/$(cygpath -m "$login_page")"
fi
shot login-step "file://$login_page" 800 600
rm -f "$out/login-step.png"
move_to_slide 0
shot presenter-slide0-desktop "$base/dev-key/" 1440 900
move_to_slide $((slide - 1))
shot presenter-last-slide-desktop "$base/dev-key/" 1440 900

echo "screenshots: $(ls "$out" | wc -l) files in $out"
