# ACP Provider（Sprint DW）

> Harness `provider.kind=acp_sdk` 通过 `ASH_ACP_ENDPOINT` 对接外部 ACP 控制面；不可用时回退 static 并记 `provider.fallback`。

## 探测

```bash
curl -s "http://127.0.0.1:8080/api/v1/providers/agent?spaceId=local" | jq '.acp,.selection'
```

| 字段 | 期望 |
|------|------|
| `acp.kind` | `acp_sdk` |
| `acp.ok` | 未配置时 `false`；端点可达时 `true` |
| harness=`acp_sdk` 且 `acp.ok=false` | `selection.fallback=true`，`adapter=static` |

## 环境

| 变量 | 说明 |
|------|------|
| `ASH_ACP_ENDPOINT` | HTTP 基址（`/readyz` 或 `/health` 探测；任务 `POST /v1/tasks`） |
| `ASH_ACP_BIN` | 可选提示 |
| `ASH_ACP_E2E=1` | readyz liveGateHints |

## 任务契约（骨架）

`POST {endpoint}/v1/tasks` JSON：`schema=ash.acp.task.v1`，含 `runId`/`prompt`/`repoRoot` 等。  
成功响应：`{"ok":true,"taskId":"...","status":"success"}`。

## 相关

- [`../plan/sprint-dw-acp-provider.md`](../plan/sprint-dw-acp-provider.md)
- [`../plan/v2.1-release-scope.md`](../plan/v2.1-release-scope.md)
- ExecGo：[`execgo-live-smoke.md`](execgo-live-smoke.md)
