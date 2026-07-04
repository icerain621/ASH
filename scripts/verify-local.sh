#!/usr/bin/env bash
# Local verification (Git Bash on Windows, bash on Linux/macOS)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "== go test (authz, doctor, api, security, store migration, pluginhealth, opsenv) =="
go test ./internal/authz ./internal/doctor ./internal/api ./internal/security ./internal/store -run 'TestMigration|TestMigrator|TestDualWrite|TestMigrationCatalog' -count=1
go test ./internal/pluginhealth ./internal/alerts ./internal/opsenv -count=1
go test ./internal/authz ./internal/doctor ./internal/api ./internal/security -count=1

echo "== regression-short =="
make regression-short

echo "== doctor M2 + M3 + TR3 + ALL (static) =="
go run ./cmd/cli doctor --suite M2
go run ./cmd/cli doctor --suite M3
go run ./cmd/cli doctor --suite TR3
go run ./cmd/cli doctor --suite ALL --agent static

echo "== metrics KPI regression =="
go test ./internal/metrics -run 'TestOverview' -count=1

echo "== openapi-check =="
bash scripts/openapi-check.sh

if docker ps --format '{{.Names}}' 2>/dev/null | grep -qx ash-postgres-dev; then
  echo "== postgres RLS e2e (optional, docker detected) =="
  make postgres-rls-e2e
else
  echo "== skip postgres-rls-e2e (start docker: make postgres-up) =="
fi

echo "== frontend build =="
(cd frontend && npm install && npm run build)

echo "OK"
