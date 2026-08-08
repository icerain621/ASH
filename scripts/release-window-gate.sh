#!/usr/bin/env bash
# MVP §8 local release window gate: static bundle + optional live/cloud (no blockers required).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
# shellcheck source=_evidence.sh
source "$ROOT/scripts/_evidence.sh"
_ash_go_env_bootstrap "$ROOT"

ash_evidence_init release-window
EVIDENCE="$ASH_EVIDENCE_DIR"
echo "Evidence dir: $EVIDENCE"

# Release window requires a passed rollback drill on EvaluateGate paths exercised by gates.
export ASH_REQUIRE_ROLLBACK_DRILL="${ASH_REQUIRE_ROLLBACK_DRILL:-1}"

FAIL=0

run_step() {
  if ash_evidence_step "$@"; then
    return 0
  fi
  FAIL=1
  return 0
}

gate_status() {
  local name="$1"
  if grep -q "^PASS ${name}$" "$EVIDENCE/pass.txt" 2>/dev/null; then
    echo "✅"
  elif grep -qE "(^SKIP ${name}|${name} skipped)" "$EVIDENCE/skipped.txt" 2>/dev/null; then
    echo "⏸"
  elif grep -q "^FAIL ${name}" "$EVIDENCE/failures.txt" 2>/dev/null; then
    echo "❌"
  else
    echo "❌"
  fi
}

mvp_signoff_status() {
  local st
  st="$(gate_status mvp-signoff)"
  if [[ "$st" == "✅" ]]; then
    echo "✅"
    return
  fi
  local prior="$ROOT/doc/evidence/mvp-signoff-latest.md"
  if [[ -f "$prior" ]] && grep -q 'regression-short | ✅' "$prior" 2>/dev/null; then
    echo "✅（prior mvp-signoff）"
    return
  fi
  echo "⏸ \`make mvp-signoff\`"
}

echo "== bootstrap sqlite (if missing) =="
bash scripts/bootstrap-local-ash-db.sh | tee -a "$EVIDENCE/bootstrap.log"

run_step config-env-gate bash scripts/config-env-gate.sh
# Soft check by default; set ASH_REQUIRE_EVIDENCE_SHA=1 to hard-fail stale *-latest.md SHAs.
ash_evidence_optional_step evidence-sha-gate bash scripts/evidence-sha-gate.sh
run_step r08-cross-space-gate env ASH_R08_SKIP_POSTGRES="${ASH_R08_SKIP_POSTGRES:-1}" bash scripts/r08-cross-space-gate.sh
run_step t0-alert-gate bash scripts/t0-alert-gate.sh
run_step t1-metrics-gate bash scripts/t1-metrics-gate.sh
run_step data-backup bash scripts/ash-data-backup.sh
run_step pre-migrate-gate bash scripts/pre-migrate-gate.sh
run_step rollback-drill env ASH_ROLLBACK_DRILL_SKIP_DOCTOR="${ASH_ROLLBACK_DRILL_SKIP_DOCTOR:-0}" bash scripts/rollback-drill-gate.sh

if [[ "${ASH_RELEASE_WINDOW_INCLUDE_MVP:-0}" == "1" ]]; then
  SKIP_OPENAPI="${ASH_RELEASE_AUDIT_SKIP_OPENAPI:-1}"
  if [[ "$SKIP_OPENAPI" == "1" ]]; then
    run_step mvp-signoff env ASH_RELEASE_AUDIT_SKIP_OPENAPI=1 bash scripts/mvp-signoff-gate.sh
  else
    run_step mvp-signoff bash scripts/mvp-signoff-gate.sh
  fi
else
  echo "SKIP mvp-signoff (set ASH_RELEASE_WINDOW_INCLUDE_MVP=1 for full signoff)" | tee -a "$EVIDENCE/skipped.txt"
fi

if [[ "${ASH_RELEASE_WINDOW_LIVE:-0}" == "1" ]]; then
  ash_evidence_optional_step worker-local-gate bash scripts/worker-local-gate.sh
  ash_evidence_optional_step worker-production-gate bash scripts/worker-production-gate.sh
else
  echo "SKIP worker-local-gate (set ASH_RELEASE_WINDOW_LIVE=1)" | tee -a "$EVIDENCE/skipped.txt"
  echo "SKIP worker-production-gate (set ASH_RELEASE_WINDOW_LIVE=1)" | tee -a "$EVIDENCE/skipped.txt"
fi

if [[ -n "${ASH_DATABASE_URL:-}" ]]; then
  run_step cloud-acceptance bash scripts/cloud-acceptance-gate.sh
  CLOUD_STATUS="✅"
else
  echo "SKIP cloud-acceptance (ASH_DATABASE_URL unset)" | tee -a "$EVIDENCE/skipped.txt"
  CLOUD_STATUS="⏸ 待环境"
fi

DATE_UTC="$(date -u +%Y-%m-%d)"
REPORT="$ROOT/doc/evidence/release-window-${DATE_UTC}.md"
LATEST="$ROOT/doc/evidence/release-window-latest.md"
pass_count=0
[[ -f "$EVIDENCE/pass.txt" ]] && pass_count="$(wc -l <"$EVIDENCE/pass.txt" | tr -d ' ')"

write_report() {
  local dest="$1"
  {
    echo "# 发布窗口记录（${DATE_UTC}）"
    echo
    echo "- 生成时间（UTC）：$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo "- Git：$(git rev-parse --short HEAD) @ $(git rev-parse --abbrev-ref HEAD)"
    echo "- 门禁命令：\`make release-window-gate\`"
    echo "- 自动化步骤通过：**${pass_count}**"
    echo "- 证据目录：\`$EVIDENCE\`（本地，未入库）"
    echo "- 手册：[\`release-window-runbook.md\`](../checklists/release-window-runbook.md)"
    echo
    echo "## 门禁快照"
    echo
    echo "| 门禁 | 命令 | 状态 |"
    echo "|------|------|------|"
    echo "| 配置核对 | \`make config-env-gate\` | $(gate_status config-env-gate) |"
    echo "| R-08 跨 space | \`make r08-cross-space-gate\` | $(gate_status r08-cross-space-gate) |"
    echo "| T+0 告警 | \`make t0-alert-gate\` | $(gate_status t0-alert-gate) |"
    echo "| T+1 指标 | \`make t1-metrics-gate\` | $(gate_status t1-metrics-gate) |"
    echo "| 数据备份 | \`make data-backup\` | $(gate_status data-backup) |"
    echo "| 迁移前 | \`make pre-migrate-gate\` | $(gate_status pre-migrate-gate) |"
    echo "| 回滚演练 | \`make rollback-drill\` | $(gate_status rollback-drill) |"
    echo "| MVP 签字 | \`make mvp-signoff\` | $(mvp_signoff_status) |"
    echo "| Live Worker | \`make worker-local-gate\` | $(gate_status worker-local-gate) |"
    echo "| JWT Worker | \`make worker-production-gate\` | $(gate_status worker-production-gate) |"
    echo "| 云 RDS | \`make cloud-acceptance\` | ${CLOUD_STATUS} |"
    echo
    echo "## 切换检查"
    echo
    local chk_data chk_pre chk_cloud chk_t0 chk_t1
    chk_data="$(gate_status data-backup)"
    chk_pre="$(gate_status pre-migrate-gate)"
    chk_t0="$(gate_status t0-alert-gate)"
    chk_t1="$(gate_status t1-metrics-gate)"
    if [[ "$CLOUD_STATUS" == "✅" ]]; then chk_cloud="[x]"; else chk_cloud="[ ]"; fi
    echo "- $([[ "$chk_data" == "✅" ]] && echo "[x]" || echo "[ ]") \`make data-backup\`"
    echo "- $([[ "$chk_pre" == "✅" ]] && echo "[x]" || echo "[ ]") \`make pre-migrate-gate\`"
    echo "- ${chk_cloud} \`make cloud-acceptance\`（云环境）"
    echo "- $([[ "$chk_t0" == "✅" ]] && echo "[x]" || echo "[ ]") \`make t0-alert-gate\`"
    echo "- $([[ "$chk_t1" == "✅" ]] && echo "[x]" || echo "[ ]") \`make t1-metrics-gate\`（T+1）"
    echo
    echo "## 回滚触发失败准则（任一即回滚）"
    echo
    echo "- \`migrate verify\` 失败或抽样行数不一致"
    echo "- M3-04 / M3-06 / M3-07 失败"
    echo "- 跨 space 数据泄漏"
    echo "- \`readyz\` 非 postgres 或持续 5xx"
    echo "- P95 查询劣化超过约定 SLO"
    echo
    echo "## 值班 roster（填写）"
    echo
    echo "| 角色 | 姓名 | 联系方式 |"
    echo "|------|------|----------|"
    echo "| 发布负责人 | | |"
    echo "| 技术 on-call | | |"
    echo "| DBA / 平台 | | |"
    echo
    echo "## 待人工"
    echo
    echo "- 发布窗口时间与值班表（§8 roster）"
    echo "- §11 四人签字：[\`mvp-signoff-latest.md\`](mvp-signoff-latest.md)"
    echo "- 云 RDS：[\`h01-h03-cloud-signoff.md\`](../checklists/h01-h03-cloud-signoff.md)"
  } >"$dest"
}

write_report "$REPORT"
cp "$REPORT" "$LATEST"
echo "Report: $LATEST"

if [[ "$FAIL" -ne 0 ]]; then
  echo "release-window-gate FAILED" >&2
  exit 1
fi
echo "OK release-window-gate"
