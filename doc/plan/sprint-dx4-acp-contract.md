# Sprint DX4：ACP 任务契约硬化（v2.1 · 方案 C）

> **方案：** 已批准 **C** = `ash.acp.task.v1` 校验 + mock 烟测 + Session turn→ACP 转发  
> **Goal:** 契约可测、本地可冒烟、Session 提示可透传 ACP（best-effort）  
> **状态：** 本批完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX4-1 | `ACPTaskV1` / `ACPTaskResultV1` Validate + Execute 接线 | ✅ |
| DX4-2 | `cmd/acp-mock` + `make acp-smoke` | ✅ |
| DX4-3 | Session `PromptTurn` → ACP forward（健康时） | ✅ |
| DX4-4 | schema JSON + 清单 / TODO | ✅ |

## 行为

- 出站任务必须 `schema=ash.acp.task.v1`，且 `prompt` 或 `issue` 非空
- 入站结果校验 schema（若有）与 status 枚举
- `make acp-smoke`：单测 + 临时 mock + `TestACPSmokeAgainstEnv`
- Session `providerKind=acp_sdk` 且探测 OK：turn 转发 ACP；失败不阻断 turn 落盘，事件带 `acpError`/`acpSkipped`

## 退出标准

- [x] 契约单测
- [x] `make acp-smoke` 绿
- [x] Session forward 单测
