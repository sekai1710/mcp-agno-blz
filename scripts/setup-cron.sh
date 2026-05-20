#!/usr/bin/env bash
# Setup daily cron jobs to refresh agno-docs-pp-cli + openrouter-docs-pp-cli.
# Idempotent: rerun replaces existing entries, never duplicates.
# Also removes any stale `blz refresh` entry (BLZ has been replaced).
#
# Customize via env:
#   SCHEDULE="0 6 * * *"   # default: daily 06:00
set -euo pipefail

SCHEDULE="${SCHEDULE:-0 6 * * *}"

AGNO_BIN="$(command -v agno-docs-pp-cli || echo "$HOME/go/bin/agno-docs-pp-cli")"
OR_BIN="$(command -v openrouter-docs-pp-cli || echo "$HOME/go/bin/openrouter-docs-pp-cli")"

[ -x "$AGNO_BIN" ] || { echo "agno-docs-pp-cli not found at $AGNO_BIN" >&2; exit 1; }
[ -x "$OR_BIN"   ] || { echo "openrouter-docs-pp-cli not found at $OR_BIN" >&2; exit 1; }

AGNO_MARKER="# agno-docs-pp-cli daily sync"
OR_MARKER="# openrouter-docs-pp-cli daily sync"
AGNO_LINE="$SCHEDULE $AGNO_BIN sync >/dev/null 2>&1 $AGNO_MARKER"
OR_LINE="$SCHEDULE $OR_BIN sync >/dev/null 2>&1 $OR_MARKER"

TMP=$(mktemp)
# Drop: existing markers + stale blz lines + the literal "blz-refresh-all" comment line
crontab -l 2>/dev/null \
  | grep -v "$AGNO_MARKER" \
  | grep -v "$OR_MARKER" \
  | grep -v "blz refresh" \
  | grep -v "^# blz-refresh-all$" \
  > "$TMP" || true
echo "$AGNO_LINE" >> "$TMP"
echo "$OR_LINE"   >> "$TMP"
crontab "$TMP"
rm -f "$TMP"

echo "Installed daily sync at: $SCHEDULE"
echo "  $AGNO_LINE"
echo "  $OR_LINE"
echo ""
echo "Verify: crontab -l"
