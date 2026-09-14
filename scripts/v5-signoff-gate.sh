#!/usr/bin/env bash
# ASH v5 governance subset signoff (Tasks 1–5 + 11 hardening probes + FE-45).
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

echo "== v5: frontend workbench / space / scale / improve / mobile =="
# Use npx vitest (same as v3x/v40 gates): `npm test` under GNU make on Win/Git Bash
# often exits 1 before vitest runs (empty stderr); direct vitest is reliable.
cd frontend
npx vitest --run \
  src/pages/ReviewsPage.test.tsx \
  src/pages/SpacePage.test.tsx \
  src/pages/ScalePage.test.tsx \
  src/components/ImproveProposalsPane.test.tsx \
  src/pages/MobileReviewsPage.test.tsx

echo "== v5: FE-45 web-build =="
# Same Win/make npm-script quirk: call tsc+vite via npx (see Makefile web-build).
npx tsc -b
npx vite build

echo "v5-signoff OK — also review doc/checklists/v5.0-signoff.md (optional human smoke)"
