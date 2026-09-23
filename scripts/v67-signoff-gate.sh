#!/usr/bin/env bash
# ASH v6.7 sign-off: harness profile echoes the sandbox backend catalog (no git tag).
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

echo "== v6.7: harness sandbox catalog =="
if ! go test ./internal/api/ -count=1 -run 'TestHarnessProfileAPILifecycle'; then
  FAIL=1
fi

SCOPE="$ROOT/doc/plan/v6.7-release-scope.md"
if grep -q '已冻结' "$SCOPE"; then
  echo "OK v6.7-release-scope frozen marker"
else
  echo "FAIL v6.7-release-scope.md missing 已冻结" >&2
  FAIL=1
fi

CHECKLIST="$ROOT/doc/checklists/v6.7-signoff.md"
if [[ ! -f "$CHECKLIST" ]]; then
  echo "FAIL missing $CHECKLIST" >&2
  FAIL=1
else
  echo "OK v6.7-signoff checklist present"
fi

TEMPLATE="$ROOT/doc/evidence/v6.7-signatures-template.md"
if [[ ! -f "$TEMPLATE" ]]; then
  echo "FAIL missing $TEMPLATE" >&2
  FAIL=1
else
  echo "OK v6.7 signatures template present"
fi

echo ""
echo "Tag is NOT created by this gate. After human sign-off:"
echo "  git tag -a v6.7.0 -m \"ASH v6.7.0 harness echoes sandbox backend catalog (VX71–VX72 frozen)\""
echo "  git push origin v6.7.0"

if [[ "$FAIL" -ne 0 ]]; then
  echo "" >&2
  echo "v6.7-signoff FAILED — see doc/checklists/v6.7-signoff.md" >&2
  exit 1
fi
echo "OK v6.7-signoff"
