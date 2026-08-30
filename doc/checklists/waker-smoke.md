# Waker 烟测（Sprint DX6）

```bash
make waker-smoke
# 可选：ASH_WORKER_URL=http://127.0.0.1:8080 make waker-smoke
```

| 变量 | 说明 |
|------|------|
| `ASH_WAKER_RUN_TTL` | 默认 `2h`；超过则视为 stale |
| `ASH_WAKER_INTERVAL` | Worker 后台巡检；`off` 关闭 |

## 相关

- [`../plan/sprint-dx6-waker.md`](../plan/sprint-dx6-waker.md)
- [`../plan/v2.2-release-scope.md`](../plan/v2.2-release-scope.md)
