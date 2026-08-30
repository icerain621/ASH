# ACP 契约烟测（Sprint DX4）

```bash
make acp-smoke
```

期望：`OK acp-smoke`；启动 `cmd/acp-mock`，`ProbeACP` + `Execute` 成功，`sessionId` 透传。

## 契约

- Schema：[`../appendices/schemas/ash.acp.task.v1.json`](../appendices/schemas/ash.acp.task.v1.json)
- Go：`internal/agentexec/acp_task.go`

## Session turn 转发

Session `providerKind=acp_sdk` 且 `ASH_ACP_ENDPOINT` 可达时，`POST .../turns` 会 best-effort 调用 ACP；`session.turn` 事件含 `acpForwarded` / `acpTaskId`。

## 相关

- [`../plan/sprint-dx4-acp-contract.md`](../plan/sprint-dx4-acp-contract.md)
- [`acp-provider.md`](acp-provider.md) · [`acp-session.md`](acp-session.md)
