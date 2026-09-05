#!/usr/bin/env bash
# v2.5 release sign-off gate: scope + Doctor ALL/M4 + Waker + RAG hybrid/vector + sandbox (no git tag).
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
run_step doctor-all go test ./internal/doctor/... -run TestALLSuite -count=1
run_step doctor-m4 go test ./internal/doctor/... -run TestM4Suite -count=1
run_step waker-smoke bash scripts/waker-smoke.sh
run_step rag-hybrid-smoke bash scripts/rag-hybrid-smoke.sh
run_step rag-vector-smoke bash scripts/rag-vector-smoke.sh
# Docker optional; Landlock segment optional via ASH_SANDBOX_LANDLOCK=1
run_step sandbox-smoke env ASH_SKIP_SANDBOX="${ASH_SKIP_SANDBOX:-1}" bash scripts/sandbox-smoke.sh

SCOPE="$ROOT/doc/plan/v2.5-release-scope.md"
if grep -q '已冻结' "$SCOPE"; then
  echo "OK v2.5-release-scope frozen marker"
else
  echo "FAIL v2.5-release-scope.md missing 已冻结" >&2
  FAIL=1
fi

CHECKLIST="$ROOT/doc/checklists/v2.5-signoff.md"
if [[ ! -f "$CHECKLIST" ]]; then
  echo "FAIL missing $CHECKLIST" >&2
  FAIL=1
else
  echo "OK v2.5-signoff checklist present"
fi

TEMPLATE="$ROOT/doc/evidence/v2.5-signatures-template.md"
if [[ ! -f "$TEMPLATE" ]]; then
  echo "FAIL missing $TEMPLATE" >&2
  FAIL=1
else
  echo "OK v2.5 signatures template present"
fi

echo ""
echo "Tag is NOT created by this gate. After human sign-off:"
echo "  git tag -a v2.5.0 -m \"ASH v2.5.0 Waker probes + ctags + vector + Landlock (DX15–DX19 frozen)\""
echo "  git push origin v2.5.0"

if [[ "$FAIL" -ne 0 ]]; then
  echo "" >&2
  echo "v2.5-signoff FAILED — see doc/checklists/v2.5-signoff.md" >&2
  exit 1
fi
echo "OK v2.5-signoff"
