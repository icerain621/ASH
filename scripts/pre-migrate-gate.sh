#!/usr/bin/env bash
# MVP §6: backup SQLite + optional migrate plan before Postgres copy.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

export ASH_DATA_DIR="${ASH_DATA_DIR:-.ash}"
export ASH_SQLITE_PATH="${ASH_SQLITE_PATH:-$ASH_DATA_DIR/ash.db}"

if [[ ! -f "$ASH_SQLITE_PATH" ]]; then
  echo "skip pre-migrate-gate: SQLite not found at $ASH_SQLITE_PATH"
  echo "hint: run Worker once or copy existing ash.db into .ash/"
  exit 0
fi

echo "== backup before migrate =="
bash scripts/ash-data-backup.sh

if [[ -z "${ASH_DATABASE_URL:-}" ]]; then
  echo "skip migrate plan (ASH_DATABASE_URL unset; set for cloud/staging plan)"
  echo "OK pre-migrate-gate (backup only)"
  exit 0
fi

echo "== migrate plan (dry-run row counts) =="
go run ./cmd/cli migrate plan \
  --data-dir "$ASH_DATA_DIR" \
  --sqlite "$ASH_SQLITE_PATH" \
  --postgres "$ASH_DATABASE_URL"

echo "OK pre-migrate-gate"
