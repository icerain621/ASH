#!/usr/bin/env bash
# H-01~H-03 cloud RDS acceptance with evidence archive (requires ASH_DATABASE_URL).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
# shellcheck source=_evidence.sh
source "$ROOT/scripts/_evidence.sh"
_ash_go_env_bootstrap "$ROOT"

if [[ -z "${ASH_DATABASE_URL:-}" ]]; then
  echo "ASH_DATABASE_URL is required for H-01 cloud acceptance" >&2
  echo "See doc/checklists/h01-h03-cloud-signoff.md" >&2
  exit 2
fi

ash_evidence_init cloud-h01-h03
EVIDENCE="$ASH_EVIDENCE_DIR"
echo "Evidence dir: $EVIDENCE"

export ASH_EVIDENCE_DIR="$EVIDENCE"
export ASH_DATA_DIR="${ASH_DATA_DIR:-.ash}"
export ASH_SQLITE_PATH="${ASH_SQLITE_PATH:-$ASH_DATA_DIR/ash.db}"
export ASH_MIGRATE_E2E="${ASH_MIGRATE_E2E:-1}"
export ASH_POSTGRES_RLS="${ASH_POSTGRES_RLS:-1}"
export ASH_POSTGRES_RLS_FORCE="${ASH_POSTGRES_RLS_FORCE:-1}"
export ASH_SCHEMA_MODE="${ASH_SCHEMA_MODE:-sql}"

ash_evidence_step h01-rds-e2e bash scripts/postgres-rds-e2e.sh

SIGNOFF="$ROOT/doc/evidence/cloud-h01-h03-latest.md"
mkdir -p "$(dirname "$SIGNOFF")"
{
  echo "# H-01~H-03 云环境验收证据"
  echo
  echo "- 生成时间（UTC）：$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "- Git：$(git rev-parse --short HEAD) @ $(git rev-parse --abbrev-ref HEAD)"
  echo "- 证据目录：\`$EVIDENCE\`（本地，未入库）"
  echo "- RDS host：\`${ASH_DATABASE_URL%%@*}@...\`（脱敏）"
  echo
  echo "## 验收项"
  echo
  echo "| 项 | 状态 | 说明 |"
  echo "|----|------|------|"
  echo "| H-01 云 RDS 全链路 | ✅ | \`postgres-rds-e2e\` + migrate verify + Doctor ALL |"
  if [[ -n "${ASH_DATABASE_APP_URL:-}" ]]; then
    echo "| H-02 RLS + ash_app | ✅ | M3-06/07 + RLS 集成测试 |"
    echo "| H-03 生产 Worker 配置 | ⏸ | 需 Worker 指向 \`ASH_DATABASE_APP_URL\` 后 §5 readyz 签字 |"
  else
    echo "| H-02 RLS + ash_app | ⏸ | 未设 \`ASH_DATABASE_APP_URL\` |"
    echo "| H-03 生产 Worker 配置 | ⏸ | 未设 \`ASH_DATABASE_APP_URL\` |"
  fi
  echo
  echo "## 签字（云验收完成后填写）"
  echo
  echo "| 角色 | 姓名 | 日期 |"
  echo "|------|------|------|"
  echo "| 技术负责人 | | |"
  echo "| 测试负责人 | | |"
  echo "| 发布负责人 | | |"
  echo
  echo "清单：[\`h01-h03-cloud-signoff.md\`](../checklists/h01-h03-cloud-signoff.md)"
} >"$SIGNOFF"

echo "OK cloud acceptance gate; signoff stub: $SIGNOFF"
