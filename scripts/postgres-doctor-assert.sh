#!/usr/bin/env bash
# Assert doctor cases passed without skip (for postgres e2e scripts).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

SUITE="${1:-M3}"
REQUIRE="${2:-}"
if [[ -z "$REQUIRE" ]]; then
  echo "usage: postgres-doctor-assert.sh <suite> <case-id>[,case-id...]" >&2
  exit 2
fi

echo "== doctor $SUITE require $REQUIRE =="
go run ./cmd/cli doctor --suite "$SUITE" --agent static --require "$REQUIRE"
echo "OK doctor $SUITE require $REQUIRE"
