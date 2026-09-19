#!/usr/bin/env bash
# v4.1 release sign-off: frozen scope + enterprise Agentic checks (no git tag).
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
run_step openapi-check bash scripts/openapi-check.sh
run_step v41-quotas-audit go test ./internal/spacepolicy/ ./internal/runs/ ./internal/api/ -count=1 -run 'Quota|Quotas|AuditReport|SubRun|SpawnHonors'
run_step v41-compliance-ui bash -c "cd \"$ROOT/frontend\" && npx vitest --run src/pages/CompliancePage.test.tsx"

SCOPE="$ROOT/doc/plan/v4.1-release-scope.md"
if grep -q '已冻结' "$SCOPE"; then
  echo "OK v4.1-release-scope frozen marker"
else
  echo "FAIL v4.1-release-scope.md missing 已冻结" >&2
  FAIL=1
fi

CHECKLIST="$ROOT/doc/checklists/v4.1-signoff.md"
if [[ ! -f "$CHECKLIST" ]]; then
  echo "FAIL missing $CHECKLIST" >&2
  FAIL=1
else
  echo "OK v4.1-signoff checklist present"
fi

TEMPLATE="$ROOT/doc/evidence/v4.1-signatures-template.md"
if [[ ! -f "$TEMPLATE" ]]; then
  echo "FAIL missing $TEMPLATE" >&2
  FAIL=1
else
  echo "OK v4.1 signatures template present"
fi

echo ""
echo "Tag is NOT created by this gate. After human sign-off:"
echo "  git tag -a v4.1.0 -m \"ASH v4.1.0 enterprise Agentic quotas audit-report spawn policy (DX67–DX72 frozen)\""
echo "  git push origin v4.1.0"

if [[ "$FAIL" -ne 0 ]]; then
  echo "" >&2
  echo "v4.1-signoff FAILED — see doc/checklists/v4.1-signoff.md" >&2
  exit 1
fi
echo "OK v4.1-signoff"
