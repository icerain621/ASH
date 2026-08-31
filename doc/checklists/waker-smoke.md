# Waker 烟测（Sprint DX6）

```bash
make waker-smoke
# 可选：ASH_WORKER_URL=http://127.0.0.1:8080 make waker-smoke
```

| 变量 | 说明 |
|------|------|
| `ASH_WAKER_RUN_TTL` | 默认 `2h`；超过则视为 stale |
| `ASH_WAKER_INTERVAL` | Worker 后台巡检；`off` 关闭 |
| `ASH_WAKER_ALLOW_CANCEL=1` | 允许 `action=cancel`（仍需 confirm） |

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
- [`../plan/v2.2-release-scope.md`](../plan/v2.2-release-scope.md)
