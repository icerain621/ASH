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

echo "== release sampling api smoke =="
go test ./internal/api/ -run 'TestReleaseSampling' -count=1

echo "== ci fixture api smoke =="
go test ./internal/api/ -run 'TestCISyncRunsWithFixture|TestReleaseSamplingCIFixture' -count=1

echo "== execgo M3-05 static smoke =="
go test ./internal/doctor/ -run TestM3ExecGoLiveSmoke -count=1

echo "== secret rotate H-07 static smoke =="
go test ./internal/api/ -run TestSecretRotateRepoConnectionH07 -count=1

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

LOCAL_EXECGO="$ROOT/.ash/execgo/execgo/bin/execgocli"
if command -v execgocli >/dev/null 2>&1 || [[ -x "$LOCAL_EXECGO" ]]; then
  echo "== execgo-health (optional) =="
  if ! make execgo-health; then
    echo "WARN: execgo-health failed (optional; see doc/checklists/execgo-live-smoke.md)"
  fi
else
  echo "== skip execgo-health (make execgo-bootstrap or install execgocli) =="
fi

echo "== frontend build =="
(cd frontend && npm install && npm run build)

echo "OK"
