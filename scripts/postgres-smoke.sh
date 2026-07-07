#!/usr/bin/env bash
# Postgres migration smoke (Git Bash / Linux)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "== parse ASH_DATABASE_URL profiles =="
env -u ASH_DATABASE_URL -u ASH_DATABASE_APP_URL go test ./internal/store/ -run 'TestParseDatabaseTargetPostgresURL|TestDatabaseProfileSQLiteDefault' -count=1

if [[ -z "${ASH_DATABASE_URL:-}" ]]; then
  echo "ASH_DATABASE_URL unset — skipping live postgres connection"
  echo "OK (parse-only)"
  exit 0
fi

echo "== live open with ASH_DATABASE_URL =="
if [[ "${ASH_POSTGRES_RLS_FORCE:-}" == "1" ]]; then
  echo "skip doctor M3 (RLS force; full suite runs later in postgres-rds-e2e)"
else
  go run ./cmd/cli doctor --suite M3
fi

echo "OK"
