# Sprint DX37 — Remote Executor 契约（v2.9 · D1/D2）

> **方案：** `internal/sandbox/remote` 扩展点 + env 探测 + mock 后端；**不**改变默认路由  
> **Goal:** `Decision.Executor=remote` 可 dispatch；`ASH_SANDBOX_REMOTE` 默认关；无新表  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX37-1 | `remote.Backend` + ConfigFromEnv + Status | ✅ |
| DX37-2 | `MockExecutor` Dispatch | ✅ |
| DX37-3 | runs dispatch `case "remote"`（含 landlock/remote 接线） | ✅ |
| DX37-4 | 单测 + sprint / TODO / CHANGELOG | ✅ |

## Env

| 变量 | 说明 |
|------|------|
| `ASH_SANDBOX_REMOTE` | `0`/`off`（默认）关闭；`1`/`on`/`auto` 启用探测 |
| `ASH_SANDBOX_REMOTE_BACKEND` | `mock`（DX37 默认）\| `e2b`（DX38） |
| `ASH_SANDBOX_REMOTE_URL` | 远程 API base（e2b） |
| `ASH_SANDBOX_REMOTE_API_KEY` | 密钥（勿入库） |
| `ASH_SANDBOX_REMOTE_TIMEOUT_SEC` | 默认 60 |

## 验收

```bash
go test ./internal/sandbox/remote/ ./internal/sandbox/ -count=1
ASH_SKIP_SANDBOX=1 make sandbox-smoke
```
