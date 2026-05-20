#!/usr/bin/env bash
# Setup cron job to refresh BLZ docs index daily at 06:00 local time.
# Cross-platform: macOS (BSD cron) + Linux (Vixie cron).
# Idempotent: rerun replaces existing entry, never duplicates.
#
# Defaults to `--all` (refresh every source you've added). Override via SOURCE env:
#   SOURCE=agno bash scripts/setup-cron.sh        # only refresh agno
#   SOURCE=openrouter bash scripts/setup-cron.sh  # only refresh openrouter
#   SCHEDULE="0 */4 * * *" bash scripts/setup-cron.sh  # every 4 hours

set -euo pipefail

BLZ_BIN="${BLZ_BIN:-$(command -v blz || echo /opt/homebrew/bin/blz)}"
SOURCE="${SOURCE:-all}"
SCHEDULE="${SCHEDULE:-0 6 * * *}"
LOG_FILE="${LOG_FILE:-$HOME/.blz/refresh-${SOURCE}.log}"

if [[ ! -x "$BLZ_BIN" ]]; then
  echo "✗ blz binary not found at $BLZ_BIN" >&2
  echo "  Install: brew install outfitter-dev/tap/blz" >&2
  echo "  Or set BLZ_BIN=/path/to/blz before re-running." >&2
  exit 1
fi

mkdir -p "$(dirname "$LOG_FILE")"

if [[ "$SOURCE" == "all" ]]; then
  REFRESH_ARG="--all"
else
  REFRESH_ARG="$SOURCE"
fi

CRON_LINE="${SCHEDULE} ${BLZ_BIN} refresh ${REFRESH_ARG} >> ${LOG_FILE} 2>&1"
MARKER="# blz-refresh-${SOURCE}"

EXISTING="$(crontab -l 2>/dev/null || true)"

# Strip any previous blz-refresh-* entries from this script to avoid duplicates.
CLEANED="$(echo "$EXISTING" | awk '
  /^# blz-refresh-/ { skip=1; next }
  skip==1 { skip=0; next }
  { print }
')"

NEW_CRONTAB="$(printf '%s\n%s\n%s\n' "$CLEANED" "$MARKER" "$CRON_LINE")"
echo "$NEW_CRONTAB" | crontab -

echo "✓ Cron installed:"
echo "  Schedule:    ${SCHEDULE}"
echo "  Command:     ${BLZ_BIN} refresh ${REFRESH_ARG}"
echo "  Log file:    ${LOG_FILE}"
echo
echo "Verify:        crontab -l | grep blz"
echo "Disable:       crontab -l | grep -v '${MARKER}' | crontab -"
