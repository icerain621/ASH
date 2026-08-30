# Sprint DT：Session RPC/JSON（方案 C）Implementation Plan

> **方案：** 已批准 **C** = HTTP Session API + turns + events 桥接 + `ash session rpc` LF-JSON  
> **Goal：** 外部/CLI 可托管长任务会话，绑定 Run 并转发事件。  
> **状态：** ✅ 本批完成 · **无新表**（会话持久化在 `audit_log`，`event_type=agent.session`）

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DT-1 | `POST/GET /api/v1/agents/sessions` | ✅ |
| DT-2 | `POST .../turns` + `GET .../events`（桥接 run events / streamUrl） | ✅ |
| DT-3 | `ash session rpc` LF-JSON（session.start / turn.prompt / event） | ✅ |
| DT-4 | OpenAPI / 清单 / docs | ✅ |

## 协议（RPC 行）

```json
{"type":"session.start","goal":"...","repoRoot":".","autoApprove":true}
{"type":"turn.prompt","sessionId":"sess_...","prompt":"..."}
→ {"type":"event","name":"session.started","sessionId":"...","payload":{...}}
→ {"type":"event","name":"turn.accepted","sessionId":"...","payload":{...}}
```
