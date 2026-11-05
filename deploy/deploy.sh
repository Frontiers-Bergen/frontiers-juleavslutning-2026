#!/usr/bin/env bash
set -euo pipefail

# Installs or updates one environment of the juleavslutning app on the lekeplassen server.
# Runs as root (via sudo) from the CI/CD workflow, from /opt/juleavslutning/<SAFE_NAME>, after CI uploaded:
#   juleavslutning.new                          the new binary
#   static/                                      the static files the app serves from disk
#   juleavslutning-<SAFE_NAME>.service           generated from juleavslutning.service.tmpl
#   caddy-<SAFE_NAME>.caddy                      generated from caddy-site.tmpl
#
# SAFE_NAME is "main" (the big screen) or "pr-<number>" (a pull request preview).
# presentation.db lives in the same directory and is never touched here: main keeps its data across deploys,
# a preview starts with an empty database (created from schema.sql on first start) and loses it on cleanup.

SAFE_NAME="${1:-}"
if [[ ! "$SAFE_NAME" =~ ^(main|pr-[0-9]+)$ ]]; then
  echo "[deploy] ERROR: SAFE_NAME must be 'main' or 'pr-<number>', got '$SAFE_NAME'." >&2
  exit 1
fi

APP_DIR="/opt/juleavslutning/$SAFE_NAME"
SERVICE_NAME="juleavslutning-$SAFE_NAME.service"
SERVICE_SRC="$APP_DIR/$SERVICE_NAME"
SERVICE_UNIT="/etc/systemd/system/$SERVICE_NAME"
CADDY_SITE_SRC="$APP_DIR/caddy-$SAFE_NAME.caddy"
CADDY_SITE_DEST="/etc/caddy/sites-enabled/juleavslutning-$SAFE_NAME.caddy"
NEW_BIN="$APP_DIR/juleavslutning.new"
CUR_BIN="$APP_DIR/juleavslutning"

for required in "$NEW_BIN" "$SERVICE_SRC" "$CADDY_SITE_SRC" "$APP_DIR/static"; do
  if [[ ! -e "$required" ]]; then
    echo "[deploy] ERROR: missing $required" >&2
    exit 1
  fi
done

echo "[deploy] Installing systemd unit $SERVICE_UNIT"
mv "$SERVICE_SRC" "$SERVICE_UNIT"
chmod 644 "$SERVICE_UNIT"

echo "[deploy] Installing Caddy site $CADDY_SITE_DEST"
mkdir -p /etc/caddy/sites-enabled
mv "$CADDY_SITE_SRC" "$CADDY_SITE_DEST"
chmod 644 "$CADDY_SITE_DEST"
systemctl reload caddy

echo "[deploy] Promoting the new binary"
if [[ -f "$CUR_BIN" ]]; then
  mv "$CUR_BIN" "$CUR_BIN.old"
fi
mv "$NEW_BIN" "$CUR_BIN"
chmod +x "$CUR_BIN"
chown -R deploy:www-data "$APP_DIR"

echo "[deploy] Restarting $SERVICE_NAME"
systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl restart "$SERVICE_NAME"

sleep 2
if ! systemctl is-active --quiet "$SERVICE_NAME"; then
  echo "[deploy] ERROR: $SERVICE_NAME failed to start" >&2
  journalctl -u "$SERVICE_NAME" -n 50 --no-pager || true
  exit 1
fi
echo "[deploy] Done: $SERVICE_NAME is running."
