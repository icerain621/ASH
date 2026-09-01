# Waker 烟测（Sprint DX6 / DX12）

```bash
make waker-smoke
# 可选：ASH_WORKER_URL=http://127.0.0.1:8080 make waker-smoke
```

包测覆盖 queue/sweep、`EnsureStaleRunDuty`、`RunDueDuties`、`/waker/status` 与 duties。进程内断言已足够；设 `ASH_WORKER_URL` 时额外 curl queue 与 status。

| 变量 | 说明 |
|------|------|
| `ASH_WAKER_RUN_TTL` | 默认 `2h`；超过则视为 stale |
| `ASH_WAKER_INTERVAL` | Worker 后台巡检；`off` / 空则关闭 ticker |
| `ASH_WAKER_ALLOW_CANCEL=1` | 允许 `action=cancel`（仍需 confirm） |

## Duties（DX12）

默认按 space 确保 `stale_run` 职责；ticker 读 `waker_duties` 到期行，**只 report/flag**，写入 `waker_duty_runs`，永不 cancel。

```bash
curl -sf "$ASH_WORKER_URL/api/v1/waker/status?spaceId=local&recent=3"
curl -sf "$ASH_WORKER_URL/api/v1/waker/duties?spaceId=local"
curl -s -X POST "$ASH_WORKER_URL/api/v1/waker/duties/${DUTY_ID}/run" \
  -H 'Content-Type: application/json' \
  -d '{"dryRun":true}'
```

`doctor_subset` / `kpi_drift` 探针属 **DX13**；本烟测不要求执行。

## Cancel（DX7）

```bash
export ASH_WAKER_ALLOW_CANCEL=1
curl -s -X POST "$ASH_WORKER_URL/api/v1/waker/sweep" \
  -H 'Content-Type: application/json' \
  -d '{"dryRun":false,"action":"cancel","confirm":"CANCEL_STALE_RUNS","maxAge":"2h"}'
```

后台 ticker **不会** cancel。见 [`../plan/sprint-dx7-waker-cancel.md`](../plan/sprint-dx7-waker-cancel.md)。

## 相关

- [`../plan/sprint-dx6-waker.md`](../plan/sprint-dx6-waker.md)
- [`../plan/sprint-dx12-waker-duties.md`](../plan/sprint-dx12-waker-duties.md)
- [`../plan/v2.2-release-scope.md`](../plan/v2.2-release-scope.md)（已冻结）
- [`../plan/v2.4-release-scope.md`](../plan/v2.4-release-scope.md)（草案 · 未冻结）
- 签字：[`v2.2-signoff.md`](v2.2-signoff.md) / `make v2.2-signoff`（v2.4 签字属 DX14）
