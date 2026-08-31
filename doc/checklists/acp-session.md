# ACP ↔ Session（Sprint DX3）

> Run 在 `provider.kind=acp_sdk` 时自动挂/建 `sess_*`；Session 也可主动声明 `providerKind`。

## Session 声明 provider

```bash
curl -s -X POST "$ASH_WORKER_URL/api/v1/agents/sessions" \
  -H 'Content-Type: application/json' \
  -d '{"repoRoot":".","providerKind":"acp_sdk","spaceId":"local"}'
```

期望：`providerKind=acp_sdk`；未配 `ASH_ACP_ENDPOINT` 时 `providerFallback=true`、`providerAdapter=static`。

## Run 自动链接

1. Harness active profile：`provider.kind=acp_sdk`
2. 启动含 agent 步骤的 Run
3. 事件流出现 `session.linked` 与 `provider.selected.sessionId`
4. `AgentTask.sessionId` 为 `sess_*`（非 run id）

## Turn → ACP（DX4）

`providerKind=acp_sdk` 且端点可达时，`POST .../turns` best-effort 转发；见 [`acp-smoke.md`](acp-smoke.md)。

## 相关

- [`../plan/sprint-dx3-acp-session.md`](../plan/sprint-dx3-acp-session.md)
- [`acp-provider.md`](acp-provider.md) · [`session-rpc.md`](session-rpc.md) · [`acp-smoke.md`](acp-smoke.md)
