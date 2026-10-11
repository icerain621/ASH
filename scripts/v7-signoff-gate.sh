#!/usr/bin/env bash
# ASH v7 sign-off: P0–P12 smoke + doc anchors (no git tag).
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

check_v7_scope() {
  local scope="$1"
  local label="$2"
  if [[ ! -f "$scope" ]]; then
    echo "FAIL missing $scope" >&2
    FAIL=1
    return 0
  fi
  local heading
  for heading in "## 1. 发布目标" "## 2. 包含范围" "## 3. 不包含范围" "## 4. 验收" "## 5. 评审签字"; do
    if ! grep -qF "$heading" "$scope"; then
      echo "FAIL $label missing section: $heading" >&2
      FAIL=1
      return 0
    fi
  done
  if ! grep -qE '验收测试已绿|P0 验收测试已绿|已绿，待签字|已冻结' "$scope"; then
    echo "FAIL $label missing green/frozen status marker" >&2
    FAIL=1
    return 0
  fi
  echo "OK $label scope structure"
}

echo "== v7 scopes P0–P13 =="
n=0
while [[ "$n" -le 13 ]]; do
  check_v7_scope "$ROOT/doc/plan/v7.${n}-release-scope.md" "v7.${n}"
  n=$((n + 1))
done

PROGRAM="$ROOT/doc/plan/v7.x-program.md"
if [[ -f "$PROGRAM" ]] && grep -qE 'P0–P1[0-9].*已绿|P0-P1[0-9].*green' "$PROGRAM"; then
  echo "OK v7.x-program watermark"
else
  echo "FAIL v7.x-program.md missing P0–P1x green watermark" >&2
  FAIL=1
fi

run_step openapi-check bash scripts/openapi-check.sh

echo "== v7: Doctor BOOT / QUALITY / TR0 =="
if ! go test ./internal/doctor/ -count=1 -timeout 360s -run 'TestBOOTSuite|TestQUALITYSuite|TestTR0Suite'; then
  FAIL=1
fi

echo "== v7: replan + lenses + verify_replan =="
if ! go test ./internal/runs/ ./internal/rules/ ./internal/harness/drive/ ./internal/observability/lenses/ \
  -count=1 -timeout 240s \
  -run 'TestReplan|TestMaybePropose|TestCountReplan|TestParseAndValidate_verifyReplan|TestAdmit|TestContextPacked|TestUnregistered'; then
  FAIL=1
fi

echo "== v7: review / memory-eval / agent templates =="
if ! go test ./internal/review/ ./internal/api/ ./internal/agenttpl/ \
  -count=1 -timeout 300s \
  -run 'TestMemoryEval|TestAgentTemplate|TestSubmit'; then
  FAIL=1
fi

echo "== v7: frontend PF07 / templates / timeline =="
if ! (cd frontend && npx vitest --run \
  src/modules/agent-session/components/SessionHistoryList.test.tsx \
  src/modules/interactions/components/ThreadTimeline.test.tsx \
  src/pages/AgentTemplatesPage.test.tsx); then
  FAIL=1
fi

CHECKLIST="$ROOT/doc/checklists/v7-signoff.md"
if [[ ! -f "$CHECKLIST" ]]; then
  echo "FAIL missing $CHECKLIST" >&2
  FAIL=1
else
  echo "OK v7-signoff checklist present"
fi

TEMPLATE="$ROOT/doc/evidence/v7-signatures-template.md"
if [[ ! -f "$TEMPLATE" ]]; then
  echo "FAIL missing $TEMPLATE" >&2
  FAIL=1
else
  echo "OK v7 signatures template present"
fi

SCOPE="$ROOT/doc/plan/v7.13-release-scope.md"
if grep -qE '已冻结|验收测试已绿' "$SCOPE"; then
  echo "OK v7.13-release-scope frozen/green marker"
else
  echo "FAIL v7.13-release-scope.md missing frozen/green marker" >&2
  FAIL=1
fi

echo ""
echo "Tag is NOT created by this gate. After human sign-off:"
echo "  git tag -a v7.0.0 -m \"ASH v7.0.0 component agent / pluggable memory / three lenses (P0–P13)\""
echo "  git push origin v7.0.0"

if [[ "$FAIL" -ne 0 ]]; then
  echo "" >&2
  echo "v7-signoff FAILED — see doc/checklists/v7-signoff.md" >&2
  exit 1
fi
echo "OK v7-signoff"
