# Sprint DX6：Waker 雏形（v2.2 · 方案 C）

> **方案：** 已批准 **C** = DX5 落地后开 Waker：过期 Run 巡检 queue/sweep + 可选后台  
> **Goal:** 无新表；`GET /waker/queue` + `POST /waker/sweep`；`ASH_WAKER_*`  
> **状态：** 本批完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX6-1 | `internal/waker` Queue/Sweep + 单测 | ✅ |
| DX6-2 | API + OpenAPI + Worker `ASH_WAKER_INTERVAL` | ✅ |
| DX6-3 | `make waker-smoke` + 清单 / v2.2 草案 | ✅ |

## 行为

- 候选：`status IN (running, waiting_approval)` 且 `updated_at` 早于 TTL（默认 `ASH_WAKER_RUN_TTL=2h`）
- sweep：`dryRun` 默认 true；`dryRun=false` 仅 **flagged** 计数 + 审计 `waker.sweep_completed`（不自动 cancel）
- 后台：`ASH_WAKER_INTERVAL`（如 `5m`）启动 Worker 周期巡检日志

## 退出标准

- [x] 包测 + API 测
- [x] `make waker-smoke`
