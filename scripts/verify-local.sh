#!/usr/bin/env bash
# Local verification (Git Bash on Windows, bash on Linux/macOS)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "== go test (authz, doctor, api, security, store migration) =="
go test ./internal/authz ./internal/doctor ./internal/api ./internal/security ./internal/store -run 'TestMigration|TestMigrator|TestDualWrite|TestMigrationCatalog' -count=1
go test ./internal/authz ./internal/doctor ./internal/api ./internal/security -count=1

echo "== doctor M2 + M3 + TR3 =="
go run ./cmd/cli doctor --suite M2
go run ./cmd/cli doctor --suite M3
go run ./cmd/cli doctor --suite TR3

echo "== swagger regen =="
bash scripts/regenerate-swagger.sh

echo "== frontend build =="
(cd frontend && npm run build)

echo "OK"
