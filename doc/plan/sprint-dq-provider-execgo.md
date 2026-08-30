# Sprint DQ：Provider + ExecGo live（方案 C）Implementation Plan

> **方案：** 已批准 **C** = Harness `provider.kind` → Agent 选型 + 不可用回退事件 + live 探测 API  
> **Goal:** Profile 驱动 static/execgo；ExecGo 不健康时回退 static 并发事件；`GET /providers/agent` 暴露探测；H-06 清单对齐。  
> **状态：** ✅ 本批完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DQ-1 | `agentexec.Resolve` + ExecGo `Probe` | ✅ |
| DQ-2 | runs 按 Harness provider 选型；fallback 事件 | ✅ |
| DQ-3 | `GET /api/v1/providers/agent` + openapi | ✅ |
| DQ-4 | docs / H-06 TODO / 单测 | ✅ |

## 行为

- CLI `--agent` / `WithAgentExecutor` **钉死**适配器时，不覆盖 Profile
- 否则读 active Harness `provider.kind`：`static` | `execgo`（`acp_sdk` 暂回退 static）
- `execgo` 且 Probe 失败 → static + `provider.fallback` / `provider.selected`
- 平台默认 Profile provider=`execgo`（与 Worker 默认对齐）
- live 仍需 `ASH_EXECGO_E2E=1 make execgo-live-smoke`（真实 Codex）

## 退出标准

- [x] Profile kind 影响 agent 选型（未钉死时）
- [x] Probe API 可用；openapi-check 绿
- [x] H-06 文档标明代码就绪 / 环境门禁
