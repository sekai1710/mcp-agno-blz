#!/usr/bin/env bash
# install.sh — one-command installer for agno-docs-pp-cli
# Usage:  curl -sSf https://raw.githubusercontent.com/sekai1710/agno-docs-pp-cli/main/install.sh | bash
set -euo pipefail

REPO="github.com/sekai1710/agno-docs-pp-cli"
TAG="${TAG:-latest}"

echo "→ checking Go toolchain"
if ! command -v go >/dev/null 2>&1; then
  echo "  Go not found. Install Go 1.26+ from https://go.dev/dl/ then re-run." >&2
  exit 1
fi
go version

echo "→ installing agno-docs-pp-cli@${TAG} (with sqlite_fts5)"
go install -tags sqlite_fts5 "${REPO}/cmd/agno-docs-pp-cli@${TAG}"

BIN_PATH="$(go env GOBIN)"
[ -z "$BIN_PATH" ] && BIN_PATH="$(go env GOPATH)/bin"

if ! echo "$PATH" | tr ':' '\n' | grep -qx "$BIN_PATH"; then
  echo
  echo "⚠  $BIN_PATH is not on your PATH. Add this to your shell rc:"
  echo "    export PATH=\"$BIN_PATH:\$PATH\""
  echo
fi

echo "→ syncing docs.agno.com/llms-full.txt (≈5 s)"
"$BIN_PATH/agno-docs-pp-cli" sync

echo
echo "✓ done. Try:"
echo "    agno-docs-pp-cli which \"how do teams work\""
echo "    agno-docs-pp-cli context teams"
echo "    agno-docs-pp-cli examples \"PostgresDb\" --language python"
