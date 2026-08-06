#!/usr/bin/env bash
# MVP release sign-off gate: static automation + optional Docker/cloud evidence.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
# shellcheck source=_evidence.sh
source "$ROOT/scripts/_evidence.sh"
_ash_go_env_bootstrap "$ROOT"

ash_evidence_init mvp-signoff
EVIDENCE="$ASH_EVIDENCE_DIR"
echo "Evidence dir: $EVIDENCE"

export ASH_REQUIRE_ROLLBACK_DRILL="${ASH_REQUIRE_ROLLBACK_DRILL:-1}"

FAIL=0

run_step() {
  if ash_evidence_step "$@"; then
    return 0
  fi
  FAIL=1
  return 0
}

run_step regression-short bash scripts/regression-short.sh
run_step web-gate bash scripts/web-gate.sh
run_step production-config-gate bash scripts/production-config-gate.sh
run_step scope-freeze-gate bash scripts/scope-freeze-gate.sh
run_step config-env-gate bash scripts/config-env-gate.sh
run_step queue-gate bash scripts/queue-gate.sh
run_step t0-alert-gate bash scripts/t0-alert-gate.sh
run_step t1-metrics-gate bash scripts/t1-metrics-gate.sh
run_step release-sampling-static bash scripts/release-sampling-static.sh
run_step doctor-all go test ./internal/doctor/... -run TestALLSuite -count=1

SKIP_OPENAPI="${ASH_RELEASE_AUDIT_SKIP_OPENAPI:-1}"
if [[ "$SKIP_OPENAPI" == "1" ]]; then
  ash_evidence_optional_step release-window-audit \
    env ASH_RELEASE_AUDIT_SKIP_OPENAPI=1 bash scripts/release-window-audit.sh
else
  run_step release-window-audit bash scripts/release-window-audit.sh
fi

if ash_evidence_has_docker_postgres; then
  run_step postgres-app-gate bash scripts/postgres-app-gate.sh
  H02_STATUS="✅ 本地 Docker"
else
  echo "postgres-app-gate skipped (docker ash-postgres-dev not running)" | tee -a "$EVIDENCE/skipped.txt"
  H02_STATUS="⏸ 需 Docker 或云 RDS"
fi

if [[ -n "${ASH_DATABASE_URL:-}" ]]; then
  run_step cloud-h01-h03 bash scripts/cloud-acceptance-gate.sh
  H01_STATUS="✅ 云 RDS"
else
  echo "cloud-acceptance skipped (ASH_DATABASE_URL unset)" | tee -a "$EVIDENCE/skipped.txt"
  H01_STATUS="⏸ 需 ASH_DATABASE_URL"
fi

run_step rollback-drill bash scripts/rollback-drill-gate.sh
ash_evidence_optional_step worker-local-gate bash scripts/worker-local-gate.sh

DATE_UTC="$(date -u +%Y-%m-%d)"
REPORT="$ROOT/doc/evidence/mvp-signoff-${DATE_UTC}.md"
LATEST="$ROOT/doc/evidence/mvp-signoff-latest.md"

pass_count=0
[[ -f "$EVIDENCE/pass.txt" ]] && pass_count="$(wc -l <"$EVIDENCE/pass.txt" | tr -d ' ')"

write_report() {
  local dest="$1"
  {
    echo "# MVP 发布签字证据（${DATE_UTC}）"
    echo
    echo "- 生成时间（UTC）：$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo "- Git：$(git rev-parse --short HEAD) @ $(git rev-parse --abbrev-ref HEAD)"
    echo "- 自动化步骤通过：**${pass_count}**"
    echo "- 证据目录：\`$EVIDENCE\`（本地，未入库）"
    echo "- 门禁命令：\`make mvp-signoff\`"
    echo
    echo "## 自动化验收"
    echo
    echo "| 门禁 | 状态 |"
    echo "|------|------|"
    echo "| regression-short | $(grep -q '^PASS regression-short' "$EVIDENCE/pass.txt" 2>/dev/null && echo ✅ || echo ❌) |"
    echo "| web-gate (lint+test+build) | $(grep -q '^PASS web-gate' "$EVIDENCE/pass.txt" 2>/dev/null && echo ✅ || echo ❌) |"
    echo "| production-config-gate | $(grep -q '^PASS production-config-gate' "$EVIDENCE/pass.txt" 2>/dev/null && echo ✅ || echo ❌) |"
    echo "| config-env-gate | $(grep -q '^PASS config-env-gate' "$EVIDENCE/pass.txt" 2>/dev/null && echo ✅ || echo ❌) |"
    echo "| queue-gate | $(grep -q '^PASS queue-gate' "$EVIDENCE/pass.txt" 2>/dev/null && echo ✅ || echo ❌) |"
    echo "| t0-alert-gate | $(grep -q '^PASS t0-alert-gate' "$EVIDENCE/pass.txt" 2>/dev/null && echo ✅ || echo ❌) |"
    echo "| rollback-drill | $(grep -q '^PASS rollback-drill' "$EVIDENCE/pass.txt" 2>/dev/null && echo ✅ || echo ⏸/❌) |"
    echo "| release-sampling-static (H-09) | $(grep -q '^PASS release-sampling-static' "$EVIDENCE/pass.txt" 2>/dev/null && echo ✅ || echo ❌) |"
    echo "| Doctor ALL 43/43 | $(grep -q '^PASS doctor-all' "$EVIDENCE/pass.txt" 2>/dev/null && echo ✅ || echo ❌) |"
    echo "| release-window-audit (H-08) | $(grep -q '^PASS release-window-audit' "$EVIDENCE/pass.txt" 2>/dev/null && echo ✅ || echo ⏸/❌) |"
    echo "| H-01 云 RDS | ${H01_STATUS} |"
    echo "| H-02/H-03 ash_app | ${H02_STATUS} |"
    echo
    echo "## MVP 清单映射"
    echo
    echo "| MVP § | 项 | 自动化证据 |"
    echo "|-------|-----|------------|"
    echo "| 3 | 快捷回归 | regression-short |"
    echo "| 3 | 前端 lint/测试/构建 | web-gate |"
    echo "| 3 | 发布审计静态 | release-window-audit |"
    echo "| 5 | H-04~H-09 烟测 | release-sampling-static + regression-short |"
    echo "| 5 | Postgres 生产切换 | H-01~H-03 云验收 |"
    echo "| 10 | T+0 冒烟 | release-sampling-static |"
    echo
    echo "## 待人工签字"
    echo
    echo "见 [\`mvp-signoff-roster.md\`](../checklists/mvp-signoff-roster.md)（\`make signoff-apply\` / \`make signoff-gate\`）。云 RDS：[\`h01-h03-cloud-signoff.md\`](../checklists/h01-h03-cloud-signoff.md)。"
    if [[ -f "$ROOT/doc/evidence/mvp-signatures-latest.md" ]] && grep -q '范围冻结：已冻结' "$ROOT/doc/evidence/mvp-signatures-latest.md" 2>/dev/null; then
      echo ""
      echo "签字记录：[\`mvp-signatures-latest.md\`](../evidence/mvp-signatures-latest.md) ✅"
    fi
    echo
    echo "| 角色 | 姓名 | 日期 |"
    echo "|------|------|------|"
    echo "| 产品负责人 | | |"
    echo "| 技术负责人 | | |"
    echo "| 测试负责人 | | |"
    echo "| 发布负责人 | | |"
  } >"$dest"
}

mkdir -p "$(dirname "$REPORT")"
write_report "$REPORT"
cp "$REPORT" "$LATEST"

if [[ "$FAIL" -ne 0 ]]; then
  echo "mvp-signoff-gate completed with failures; see $EVIDENCE/failures.txt" >&2
  echo "Report: $REPORT"
  exit 1
fi

echo "OK mvp-signoff-gate; report: $REPORT"
