#!/usr/bin/env bash
set -euo pipefail

# Removes a pull request preview when its PR closes (merged or not). Runs as root (via sudo) from the
# CI/CD workflow. Idempotent: everything may already be gone. Refuses anything but "pr-<number>",
# so the big screen (main) can never be removed by accident.

SAFE_NAME="${1:-}"
if [[ ! "$SAFE_NAME" =~ ^pr-[0-9]+$ ]]; then
  echo "[cleanup] ERROR: only previews named 'pr-<number>' can be removed, got '$SAFE_NAME'." >&2
  exit 1
fi

SERVICE_NAME="juleavslutning-$SAFE_NAME.service"

echo "[cleanup] Stopping $SERVICE_NAME"
systemctl disable --now "$SERVICE_NAME" 2>/dev/null || true
rm -f "/etc/systemd/system/$SERVICE_NAME"
systemctl daemon-reload

echo "[cleanup] Removing Caddy site"
rm -f "/etc/caddy/sites-enabled/juleavslutning-$SAFE_NAME.caddy"
systemctl reload caddy

echo "[cleanup] Removing /opt/juleavslutning/$SAFE_NAME (binary, static files, preview database)"
rm -rf "/opt/juleavslutning/$SAFE_NAME"

echo "[cleanup] Done."
