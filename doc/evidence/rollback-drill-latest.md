# 回滚演练证据（自动化）

- 时间（UTC）：2026-08-27T14:04:59Z
- 耗时（ms）：26000
- SLA 上限（ms）：300000（`ASH_ROLLBACK_DRILL_MAX_MS`）
- 证据目录：`/c/Go_Work/src/ash/.ash/evidence/mvp-signoff-20260827T135920Z`
- 门禁：`make rollback-drill`

## 步骤

1. 发布治理 API 记录 rollback drill（TestReleaseGovernanceAPI）
2. 健康检查 P95 基线（TestHealthEndpointsLatencyBaseline）
3. 并发 /runs 列表（TestConcurrentRunsListBaseline）
4. Doctor ALL 静态回归（跳过：`ASH_ROLLBACK_DRILL_SKIP_DOCTOR=1`）

## 回滚触发失败准则（任一即回滚）

- `migrate verify` 失败或抽样行数不一致
- M3-04 / M3-06 / M3-07 失败
- 跨 space 数据泄漏
- `readyz` 非 postgres 或持续 5xx
- P95 查询劣化超过约定 SLO

## 人工补充（生产切换后）

- Worker 切回 SQLite / 上一版本镜像
- `curl /readyz` 与 KPI 观察 30 分钟
- 清单：[`postgres-rds-e2e.md`](../checklists/postgres-rds-e2e.md) §8
