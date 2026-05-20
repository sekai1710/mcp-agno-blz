#!/usr/bin/env bash
# Setup a daily cron job to refresh agno-docs-pp-cli's local index.
# Idempotent: rerun replaces existing entry, never duplicates.
#
# Customize via env:
#   SCHEDULE="0 */6 * * *"  # default: "0 6 * * *" (daily 06:00)
#   BIN=/custom/path/agno-docs-pp-cli
set -euo pipefail

SCHEDULE="${SCHEDULE:-0 6 * * *}"
BIN="${BIN:-$(command -v agno-docs-pp-cli || true)}"

if [ -z "$BIN" ]; then
  echo "agno-docs-pp-cli not on PATH. Install first:" >&2
  echo "  go install -tags sqlite_fts5 github.com/sekai1710/agno-docs-pp-cli/cmd/agno-docs-pp-cli@latest" >&2
  exit 1
fi

MARKER="# agno-docs-pp-cli daily sync"
LINE="$SCHEDULE $BIN sync >/dev/null 2>&1 $MARKER"

TMP=$(mktemp)
crontab -l 2>/dev/null | grep -v "$MARKER" > "$TMP" || true
echo "$LINE" >> "$TMP"
crontab "$TMP"
rm -f "$TMP"

echo "Installed: $LINE"
echo "Verify:    crontab -l | grep agno-docs-pp-cli"
