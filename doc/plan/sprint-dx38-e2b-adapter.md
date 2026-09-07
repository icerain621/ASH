# Sprint DX38 — E2B-class 可选适配器（v2.9 · D3）

> **方案：** HTTP 客户端兼容 `POST /sandboxes` + `/commands` + `DELETE`；超时/取消；非默认  
> **Goal:** `ASH_SANDBOX_REMOTE_BACKEND=e2b` + API key/URL 时 `Available`；httptest 契约测；无新表  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX38-1 | `E2BExecutor` create/run/kill | ✅ |
| DX38-2 | Auth 头、shell join、超时取消 | ✅ |
| DX38-3 | httptest 测 + 替换 DX37 stub | ✅ |
| DX38-4 | sprint / TODO / CHANGELOG | ✅ |

## Env（增量）

| 变量 | 说明 |
|------|------|
| `ASH_SANDBOX_REMOTE_BACKEND=e2b` | 选用 E2B-class 客户端 |
| `ASH_SANDBOX_REMOTE_URL` | 默认 `https://api.e2b.app`；测试/自建网关可覆盖 |
| `ASH_SANDBOX_REMOTE_API_KEY` | `X-API-Key` |
| `ASH_SANDBOX_REMOTE_TEMPLATE` | 默认 `base` |

## 验收

```bash
go test ./internal/sandbox/remote/ -count=1 -run 'TestE2B|TestMock|TestConfig'
```
