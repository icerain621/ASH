# Sprint DX3：ACP ↔ Session 互通（v2.1 · 方案 C）

> **方案：** 已批准 **C** = Session 可声明 provider + Run 在 `acp_sdk` 选型时挂/建 Session  
> **Goal:** `sess_*` 与 ACP/AgentTask 对齐；事件 `session.linked`；无新表  
> **状态：** 本批完成 · **无新表**（仍写 `audit_log` event_type=`agent.session`）

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX3-1 | Session `providerKind` + Probe 写入 View/Meta | ✅ |
| DX3-2 | `FindByRunID` / `EnsureForRun` | ✅ |
| DX3-3 | Run 执行链路：`session.linked` + AgentTask.SessionID=`sess_*` | ✅ |
| DX3-4 | ACP 任务体带 `sessionId`；OpenAPI / 清单 | ✅ |

## 行为

- `POST /agents/sessions` 可传 `providerKind`（`static`|`execgo`|`acp_sdk`）；ACP/ExecGo 探测失败则 `providerFallback=true`
- Harness/`SelectProvider` 请求 `acp_sdk` 时：`EnsureForRun` → 事件 `session.linked`；`provider.selected.sessionId`
- `executeAgentStep` / `ACPExecutor` 使用 `sess_*`（Metadata + Result.SessionID）
- RPC `session.start` 支持 `providerKind`

## 退出标准

- [x] Create + EnsureForRun 单测
- [x] Run link 单测发出 `session.linked`
- [x] OpenAPI 字段齐全；清单 `acp-session.md`
