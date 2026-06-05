#!/usr/bin/env bash
# SQLite → Postgres migration helper (Git Bash / Linux)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

CMD="${1:-plan}"
shift || true

DATA_DIR="${ASH_DATA_DIR:-.ash}"
SQLITE="${ASH_SQLITE_PATH:-$DATA_DIR/ash.db}"
POSTGRES="${ASH_DATABASE_URL:-}"

if [[ -z "$POSTGRES" ]]; then
  echo "ASH_DATABASE_URL is required (postgres target)" >&2
  exit 2
fi

case "$CMD" in
  plan|copy|verify|sync)
    go run ./cmd/cli migrate "$CMD" \
      --data-dir "$DATA_DIR" \
      --sqlite "$SQLITE" \
      --postgres "$POSTGRES" \
      "$@"
    ;;
  dual-write)
    go run ./cmd/cli migrate dual-write "${1:-status}" \
      --data-dir "$DATA_DIR" \
      --sqlite "$SQLITE" \
      --postgres "$POSTGRES" \
      "${@:2}"
    ;;
  *)
    echo "Usage: $0 plan|copy|verify|sync|dual-write [args...]" >&2
    echo "  ASH_DATA_DIR=$DATA_DIR  ASH_SQLITE_PATH=$SQLITE" >&2
    exit 2
    ;;
esac
