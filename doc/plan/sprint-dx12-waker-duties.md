# Sprint DX12：Waker duties 账本（v2.4 · 草案）

> **方案：** 已批准 duty ledger：将 TTL stale-run 巡检迁入可调度、可审计的持续职责平面  
> **Goal:** SQL **29** / RLS **50**；默认 `stale_run`；ticker 读 ledger；`/waker/status` + duties；后台永不 cancel  
> **状态：** 本批完成 · **新表 2**（`waker_duties` / `waker_duty_runs`）

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX12-1 | Schema + SQL 29 + RLS 50 + GORM models | ✅ |
| DX12-2 | Ensure `stale_run` + `RunDueDuties` + ticker 读 ledger | ✅ |
| DX12-3 | `GET /waker/status` · `GET /waker/duties` · `POST /duties/:id/run` + OpenAPI | ✅ |
| DX12-4 | `make waker-smoke` 扩展 + v2.4 范围草案 / 水位 | ✅ |

## 行为

- 空账本：首次 `/status` 或 Worker ticker 按 space **ensure** 默认 `stale_run`（`space_id` 含 `local`）
- 到期调度：`enabled AND next_run_at <= now` → `stale_run` 走 report/flag（`canceled=0`）；未知 kind 记 `skipped`（DX13 占位）
- 保留 DX6 `GET /waker/queue` 与 `POST /waker/sweep`；DX7 cancel 闸门不变
- Out（本 Sprint）：`doctor_subset` / `kpi_drift` 探针、控制台面板、范围冻结

## 退出标准

- [x] `go test ./internal/waker/ ./internal/api/ -count=1 -run 'TestWaker|TestEnsure|TestRunDue|TestQueue'`
- [x] `make waker-smoke`
- [x] `v2.4-release-scope.md` 草案（§1–§6；**未冻结**）
