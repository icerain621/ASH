# 回滚演练证据（自动化）

- 时间（UTC）：2026-07-05T11:58:03Z
- 耗时（ms）：62000
- 证据目录：`/c/Go_Work/src/ash/.ash/evidence/rollback-drill-20260705T115701Z`
- 门禁：`make rollback-drill`

## 步骤

1. 发布治理 API 记录 rollback drill（TestReleaseGovernanceAPI）
2. 健康检查 P95 基线（TestHealthEndpointsLatencyBaseline）
3. 并发 /runs 列表（TestConcurrentRunsListBaseline）
4. Doctor ALL 静态回归（TestALLSuite）

## 人工补充（生产切换后）

- Worker 切回 SQLite / 上一版本镜像
- `curl /readyz` 与 KPI 观察 30 分钟
- 清单：[`postgres-rds-e2e.md`](../checklists/postgres-rds-e2e.md) §8
