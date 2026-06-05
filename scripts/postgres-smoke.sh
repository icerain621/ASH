#!/usr/bin/env bash
# Postgres migration smoke (Git Bash / Linux)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "== parse ASH_DATABASE_URL profiles =="
go test ./internal/store/ -run 'TestParseDatabaseTargetPostgresURL|TestDatabaseProfileSQLiteDefault' -count=1

if [[ -z "${ASH_DATABASE_URL:-}" ]]; then
  echo "ASH_DATABASE_URL unset — skipping live postgres connection"
  echo "OK (parse-only)"
  exit 0
fi

echo "== live open with ASH_DATABASE_URL =="
go run ./cmd/cli doctor --suite M3

echo "OK"
