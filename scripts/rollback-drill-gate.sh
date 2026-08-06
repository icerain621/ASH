#!/usr/bin/env bash
# MVP §9: rollback drill evidence (API drill record + post-check baselines).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
# shellcheck source=_evidence.sh
source "$ROOT/scripts/_evidence.sh"
_ash_go_env_bootstrap "$ROOT"

ash_evidence_init rollback-drill
EVIDENCE="$ASH_EVIDENCE_DIR"
START_SEC=$(date +%s)

ash_evidence_step release-governance go test ./internal/api/... -run TestReleaseGovernanceAPI -count=1
ash_evidence_step health-baseline go test ./internal/api/... -run TestHealthEndpointsLatencyBaseline -count=1
ash_evidence_step concurrent-runs go test ./internal/api/... -run TestConcurrentRunsListBaseline -count=1
ash_evidence_step doctor-static go test ./internal/doctor/... -run TestALLSuite -count=1

END_SEC=$(date +%s)
DURATION_MS=$(( (END_SEC - START_SEC) * 1000 ))
MAX_MS="${ASH_ROLLBACK_DRILL_MAX_MS:-300000}"

REPORT="$ROOT/doc/evidence/rollback-drill-latest.md"
mkdir -p "$(dirname "$REPORT")"
{
  echo "# 回滚演练证据（自动化）"
  echo
  echo "- 时间（UTC）：$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "- 耗时（ms）：${DURATION_MS}"
  echo "- SLA 上限（ms）：${MAX_MS}（\`ASH_ROLLBACK_DRILL_MAX_MS\`）"
  echo "- 证据目录：\`$EVIDENCE\`"
  echo "- 门禁：\`make rollback-drill\`"
  echo
  echo "## 步骤"
  echo
  echo "1. 发布治理 API 记录 rollback drill（TestReleaseGovernanceAPI）"
  echo "2. 健康检查 P95 基线（TestHealthEndpointsLatencyBaseline）"
  echo "3. 并发 /runs 列表（TestConcurrentRunsListBaseline）"
  echo "4. Doctor ALL 静态回归（TestALLSuite）"
  echo
  echo "## 回滚触发失败准则（任一即回滚）"
  echo
  echo "- \`migrate verify\` 失败或抽样行数不一致"
  echo "- M3-04 / M3-06 / M3-07 失败"
  echo "- 跨 space 数据泄漏"
  echo "- \`readyz\` 非 postgres 或持续 5xx"
  echo "- P95 查询劣化超过约定 SLO"
  echo
  echo "## 人工补充（生产切换后）"
  echo
  echo "- Worker 切回 SQLite / 上一版本镜像"
  echo "- \`curl /readyz\` 与 KPI 观察 30 分钟"
  echo "- 清单：[\`postgres-rds-e2e.md\`](../checklists/postgres-rds-e2e.md) §8"
} >"$REPORT"

if [[ "$DURATION_MS" -gt "$MAX_MS" ]]; then
  echo "FAIL rollback-drill SLA: ${DURATION_MS}ms > ${MAX_MS}ms" >&2
  exit 1
fi

echo "OK rollback-drill-gate (${DURATION_MS}ms <= ${MAX_MS}ms); report: $REPORT"
