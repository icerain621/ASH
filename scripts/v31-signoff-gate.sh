#!/usr/bin/env bash
# v3.1 release sign-off gate: scope + Doctor ALL/M4 + Quest regression + v3.0 baseline smokes (no git tag).
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

# Quest track regression (DX49–DX53)
run_step quest-webhook go test ./internal/api/ -run 'TestGitHubWebhook' -count=1
run_step quest-events go test ./internal/events/ -run 'ListAfterSpace' -count=1
run_step quest-diffreview go test ./internal/diffreview/... -count=1
run_step quest-frontend bash -c "cd \"$ROOT/frontend\" && npx vitest --run src/pages/QuestPage.test.tsx src/services/sse/questBoardStream.test.tsx"

# Keep v3.0 platform baseline green
run_step rag-hybrid-smoke bash scripts/rag-hybrid-smoke.sh
run_step rag-vector-smoke bash scripts/rag-vector-smoke.sh
run_step sandbox-smoke env ASH_SKIP_SANDBOX="${ASH_SKIP_SANDBOX:-1}" bash scripts/sandbox-smoke.sh
run_step skill-pack-smoke bash scripts/skill-pack-smoke.sh
run_step rag-lsp-smoke bash scripts/rag-lsp-smoke.sh
run_step remote-sandbox-smoke bash scripts/remote-sandbox-smoke.sh

SCOPE="$ROOT/doc/plan/v3.1-release-scope.md"
if grep -q '已冻结' "$SCOPE"; then
  echo "OK v3.1-release-scope frozen marker"
else
  echo "FAIL v3.1-release-scope.md missing 已冻结" >&2
  FAIL=1
fi

CHECKLIST="$ROOT/doc/checklists/v3.1-signoff.md"
if [[ ! -f "$CHECKLIST" ]]; then
  echo "FAIL missing $CHECKLIST" >&2
  FAIL=1
else
  echo "OK v3.1-signoff checklist present"
fi

TEMPLATE="$ROOT/doc/evidence/v3.1-signatures-template.md"
if [[ ! -f "$TEMPLATE" ]]; then
  echo "FAIL missing $TEMPLATE" >&2
  FAIL=1
else
  echo "OK v3.1 signatures template present"
fi

EVIDENCE="$ROOT/doc/evidence/rag-vector-smoke-latest.md"
if [[ ! -f "$EVIDENCE" ]]; then
  echo "FAIL missing $EVIDENCE (run make rag-vector-smoke)" >&2
  FAIL=1
else
  echo "OK rag-vector-smoke evidence present"
fi

REMOTE_EVIDENCE="$ROOT/doc/evidence/remote-sandbox-smoke-latest.md"
if [[ ! -f "$REMOTE_EVIDENCE" ]]; then
  echo "FAIL missing $REMOTE_EVIDENCE (run make remote-sandbox-smoke)" >&2
  FAIL=1
else
  echo "OK remote-sandbox-smoke evidence present"
fi

echo ""
echo "Tag is NOT created by this gate. After human sign-off:"
echo "  git tag -a v3.1.0 -m \"ASH v3.1.0 Quest workbench compose + harden (DX49–DX54 frozen)\""
echo "  git push origin v3.1.0"

if [[ "$FAIL" -ne 0 ]]; then
  echo "" >&2
  echo "v3.1-signoff FAILED — see doc/checklists/v3.1-signoff.md" >&2
  exit 1
fi
echo "OK v3.1-signoff"
