#!/usr/bin/env bash
# ASH v5 governance subset signoff (Tasks 1–5 + 11 hardening probes).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== v5: registry / spacepolicy / scoring / evolve =="
go test ./internal/registry/... ./internal/spacepolicy/... ./internal/scoring/... ./internal/evolve/... ./internal/orgtemplates/... -count=1

echo "== v5: Doctor TR3 =="
go test ./internal/doctor/... -run TestTR3Suite -count=1

echo "== v5: openapi =="
make swagger
make openapi-check

echo "== v5: frontend workbench / space / scale =="
cd frontend
npm test -- --run src/pages/ReviewsPage.test.tsx src/pages/SpacePage.test.tsx src/pages/ScalePage.test.tsx

echo "v5-signoff OK — also review doc/checklists/v5.0-signoff.md manual items"
