#!/usr/bin/env bash
# Setup cron job to refresh BLZ Agno docs index daily at 06:00 local time.
# Cross-platform: macOS (BSD cron) + Linux (Vixie cron).
# Idempotent: if entry already present, exits 0 without duplicating.

set -euo pipefail

BLZ_BIN="${BLZ_BIN:-$(command -v blz || echo /opt/homebrew/bin/blz)}"
SOURCE="${SOURCE:-agno}"
SCHEDULE="${SCHEDULE:-0 6 * * *}"
LOG_FILE="${LOG_FILE:-$HOME/.blz/refresh-${SOURCE}.log}"

if [[ ! -x "$BLZ_BIN" ]]; then
  echo "✗ blz binary not found at $BLZ_BIN" >&2
  echo "  Install: brew install outfitter-dev/tap/blz" >&2
  echo "  Or set BLZ_BIN=/path/to/blz before re-running." >&2
  exit 1
fi

mkdir -p "$(dirname "$LOG_FILE")"

CRON_LINE="${SCHEDULE} ${BLZ_BIN} refresh ${SOURCE} >> ${LOG_FILE} 2>&1"
MARKER="# blz-refresh-${SOURCE}"

# Read existing crontab (empty if none), filter out previous marker block.
EXISTING="$(crontab -l 2>/dev/null || true)"

if echo "$EXISTING" | grep -qF "$MARKER"; then
  echo "→ Cron entry for '${SOURCE}' already present, replacing..."
  CLEANED="$(echo "$EXISTING" | grep -v -F "$MARKER" | grep -v -F "refresh ${SOURCE}" || true)"
else
  CLEANED="$EXISTING"
fi

NEW_CRONTAB="$(printf '%s\n%s\n%s\n' "$CLEANED" "$MARKER" "$CRON_LINE")"
echo "$NEW_CRONTAB" | crontab -

echo "✓ Cron installed:"
echo "  Schedule:    ${SCHEDULE}"
echo "  Command:     ${BLZ_BIN} refresh ${SOURCE}"
echo "  Log file:    ${LOG_FILE}"
echo
echo "Verify:        crontab -l | grep blz"
echo "Disable:       crontab -l | grep -v '${MARKER}' | crontab -"
