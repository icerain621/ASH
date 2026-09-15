# Agent Chat DSH 对标 — 后端缺失功能清单

> Status: **complete (主路径)**（2026-09-15）  
> Spec: [`2026-09-15-agent-chat-dsh-parity-design.md`](./2026-09-15-agent-chat-dsh-parity-design.md)  
> 原则：不引入 Cordis / 不嵌入 `dsh web`

## 进度

主路径 Chat 对标（三栏 / SSE / LLM / seats / MCP slash / 密度 / purge / 栏宽 / mid-flight stop / Markdown / Workspace 组内+跨组 DnD / Tools+MCP 设置面板）**已收口**。

| 项 | 状态 |
|----|------|
| 跨 Workspace 拖拽 | ✅ Detach API + FE Attach/Detach/重排 |
| Tools 设置入口 | ✅ 内置工具风险目录面板 |
| Cordis / 完整 slash 一比一 | ❌ 明确不做 |

## 可选加深（非阻塞）

| 项 | 说明 |
|----|------|
| Markdown 表格 / 任务列表 | BubbleMarkdown |
| Tools 运行时开关 | 超出风险目录只读展示 |

## 修订

| 日期 | 说明 |
|------|------|
| 2026-09-15 | S3/S4 |
| 2026-09-15 | 跨组 DnD + Tools 面板 + Detach API；主路径标 complete |
