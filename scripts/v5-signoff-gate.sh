#!/usr/bin/env bash
# ASH v5.0 sign-off: frozen scope + governance checks (no git tag).
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

FAIL=0

run_step() {
  local name="$1"
  shift
  echo "==> $name"
  if "$@"; then
    echo "OK $name"
    return 0
  fi
  echo "FAIL $name" >&2
  FAIL=1
  return 0
}

run_step scope-freeze-gate bash scripts/scope-freeze-gate.sh

echo "== v5: registry / spacepolicy / scoring / evolve =="
if ! go test ./internal/registry/... ./internal/spacepolicy/... ./internal/scoring/... ./internal/evolve/... ./internal/orgtemplates/... -count=1; then
  FAIL=1
fi

echo "== v5: Doctor TR3 =="
if ! go test ./internal/doctor/... -run TestTR3Suite -count=1; then
  FAIL=1
fi

run_step openapi-check bash -c 'make swagger && make openapi-check'

echo "== v5: frontend workbench / space / scale / improve / mobile =="
# Use npx vitest (same as v3x/v40 gates): `npm test` under GNU make on Win/Git Bash
# often exits 1 before vitest runs (empty stderr); direct vitest is reliable.
if ! (cd frontend && npx vitest --run \
  src/pages/ReviewsPage.test.tsx \
  src/pages/SpacePage.test.tsx \
  src/pages/ScalePage.test.tsx \
  src/components/ImproveProposalsPane.test.tsx \
  src/pages/MobileReviewsPage.test.tsx); then
  FAIL=1
fi

echo "== v5: FE-45 web-build =="
# Same Win/make npm-script quirk: call tsc+vite via npx (see Makefile web-build).
if ! (cd frontend && npx tsc -b && npx vite build); then
  FAIL=1
fi

SCOPE="$ROOT/doc/plan/v5.0-release-scope.md"
if grep -q '已冻结' "$SCOPE"; then
  echo "OK v5.0-release-scope frozen marker"
else
  echo "FAIL v5.0-release-scope.md missing 已冻结" >&2
  FAIL=1
fi

CHECKLIST="$ROOT/doc/checklists/v5.0-signoff.md"
if [[ ! -f "$CHECKLIST" ]]; then
  echo "FAIL missing $CHECKLIST" >&2
  FAIL=1
else
  echo "OK v5.0-signoff checklist present"
fi

TEMPLATE="$ROOT/doc/evidence/v5.0-signatures-template.md"
if [[ ! -f "$TEMPLATE" ]]; then
  echo "FAIL missing $TEMPLATE" >&2
  FAIL=1
else
  echo "OK v5.0 signatures template present"
fi

echo ""
echo "Tag is NOT created by this gate. After human sign-off:"
echo "  git tag -a v5.0.0 -m \"ASH v5.0.0 dual-core governance thin interaction and thick review watermark (GV01–GV06 frozen)\""
echo "  git push origin v5.0.0"

if [[ "$FAIL" -ne 0 ]]; then
  echo "" >&2
  echo "v5-signoff FAILED — see doc/checklists/v5.0-signoff.md" >&2
  exit 1
fi
echo "OK v5-signoff"
