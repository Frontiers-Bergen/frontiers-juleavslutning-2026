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
# Requires: chromium or google-chrome, curl.
set -euo pipefail

repo="$(cd "$(dirname "$0")/.." && pwd)"
out="$(realpath -m "${1:-$repo/tmp/screenshots}")"
port="${PORT:-8099}"
base="http://localhost:$port"
browser="$(command -v chromium || command -v google-chrome-stable || command -v google-chrome || true)"
if [ -z "$browser" ]; then
	echo "screenshots: chromium or google-chrome is required" >&2
	exit 1
fi

work="$(mktemp -d)"
mkdir -p "$out"
(cd "$repo" && go tool templ generate -log-level error && go build -o "$work/app" .)
cp -r "$repo/static" "$work/"
cd "$work"
PORT="$port" PRESENTER_KEY=screenshot-key ./app >server.log 2>&1 &
app_pid=$!
trap 'kill $app_pid 2>/dev/null; wait $app_pid 2>/dev/null; rm -rf "$work"' EXIT
for _ in $(seq 1 40); do curl -s -o /dev/null "$base/" && break; sleep 0.25; done

curl -s -c presenter.jar -o /dev/null -d key=screenshot-key "$base/presenter/login"

# A few votes on the app name poll, so results slides are not empty. Harmless if the poll is gone.
vote() { # person invite-key option
	curl -s -c "$1.jar" -b "$1.jar" -o /dev/null "$base/$2/"
	curl -s -b "$1.jar" -o /dev/null -X POST "$base/$2/root/api/poll/vote?poll=app-name&option=$3"
}
vote a local-dev-key its-tiiime
vote b local-dev-key its-tiiime
vote c remote-dev-key its-tiiime
vote d local-dev-key smash-the-halls
vote e remote-dev-key rage-against-the-debounce

move_to_slide() { curl -s -b presenter.jar -o /dev/null -X POST "$base/local-dev-key/root/api/slides/set?index=$1"; }
shown_slide() { curl -s -b presenter.jar "$base/local-dev-key/" | grep -oE 'slideIndex: [0-9]+' | head -1 | grep -oE '[0-9]+'; }

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
	shot "slide$slide-phone" "$base/local-dev-key/" 390 844
	shot "slide$slide-desktop" "$base/local-dev-key/" 1440 900
	slide=$((slide + 1))
done

# Log the browser profile in as presenter, then capture the presenter view of the first slide.
printf '<form method="post" action="%s/presenter/login"><input name="key" value="screenshot-key"></form><script>document.forms[0].submit()</script>' "$base" >login.html
shot login-step "file://$work/login.html" 800 600
rm -f "$out/login-step.png"
move_to_slide 0
shot presenter-slide0-desktop "$base/local-dev-key/" 1440 900
move_to_slide $((slide - 1))
shot presenter-last-slide-desktop "$base/local-dev-key/" 1440 900

echo "screenshots: $(ls "$out" | wc -l) files in $out"
