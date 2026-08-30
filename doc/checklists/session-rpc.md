# Agent Session RPC（Sprint DT）

> 验收：HTTP Session + LF-JSON CLI 可创建会话、提交 turn、桥接 Run 事件。

## 前置

1. Worker 或本地 CLI 数据目录可用
2. 可选：已有 `runId`，或准备 goal 文本 + scenarios

## HTTP 步骤

| # | 动作 | 期望 |
|---|------|------|
| 1 | `POST /api/v1/agents/sessions` `{"runId":"..."}` 或 `{"goal":"...","autoApprove":true,"repoRoot":"."}` | `201`；`id=sess_*`；有 `streamUrl`（若已绑 Run） |
| 2 | `POST /api/v1/agents/sessions/{id}/turns` `{"prompt":"..."}` | `200`；`turns` 增长；Run 上出现 `session.turn` |
| 3 | `GET /api/v1/agents/sessions/{id}/events` | `items` 含事件；`streamUrl=/api/v1/runs/{runId}/stream` |
| 4 | `GET {streamUrl}`（SSE） | 与既有 Run stream 一致 |

## CLI RPC

```bash
printf '%s\n' \
  '{"type":"session.start","goal":"hotfix flaky CI","repoRoot":".","autoApprove":false}' \
  '{"type":"turn.prompt","sessionId":"SESS","prompt":"focus on tests"}' \
| go run ./cmd/cli session rpc --agent static
```

期望 stdout 逐行：`{"type":"event","name":"session.started",...}` / `turn.accepted`。

## 相关

- `internal/session/` · `internal/api/session.go` · `ash session rpc`
- 计划：`doc/plan/sprint-dt-session-rpc.md`
