# ASH 总体设计（SDD）· 薄入口

> 更新：2026-10-08 · 归属：[`design/`](README.md)  
> Updated: 2026-10-08.

本文件是**现行设计入口**；协议与 Schema 仍以 appendices / OpenAPI 为准。  
This file is the **living design entry**; protocols and schemas stay in appendices / OpenAPI.

| 主题 | 现行依据 |
|------|----------|
| 模块与包边界 | 根 [`CLAUDE.md`](../../CLAUDE.md) · `internal/*` |
| 事件 / DSL / Memory | [`../appendices/`](../appendices/README.md) |
| HTTP 契约 | [`../api/openapi-ash-v1.yaml`](../api/openapi-ash-v1.yaml) |
| 吸收对照 | [`../plan/platform-quad-comparison.md`](../plan/platform-quad-comparison.md) |
| 历史 HLD / ARCH | [`../archive/narrative/`](../archive/narrative/) |

## 建构要点（现行） / Architecture notes

- **Worker + CLI + Vite 控制台**；会话意图经 API，UI **只渲投影**（P02）。  
- **双核**：执行（Runs/Harness/Sandbox）× 演进（Feedback/Reviews/Memory）。  
- **v6 缝**：Hooks · Notifier · Ingress；Harness 回显 `sandboxBackends`（v6.7）。  
- **冻结后**：`reasoningEffort` / `providerKind` / `llmModel` 写入事件投影，不新表。

## 待补 / TODO

完整 SDD 章节（部署视图、时序、威胁模型摘要）可按模块分批扩写；扩写前请先确认要覆盖的模块清单。  
Expand chapter-by-chapter after agreeing which modules need long-form design docs.
