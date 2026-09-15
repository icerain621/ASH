# Agent Chat DSH 对标 — 后端缺失功能清单

> Status: **near-complete**（2026-09-15）  
> Spec: [`2026-09-15-agent-chat-dsh-parity-design.md`](./2026-09-15-agent-chat-dsh-parity-design.md)  
> 已落地：三栏壳 · P0–P5 · MCP slash · 密度 · purge · 栏宽 · mid-flight stop · Markdown · **Workspace DnD** · **Agent MCP 面板**  
> 原则：不引入 Cordis / 不嵌入 `dsh web`

---

## 进度总览

| 阶段 | 状态 |
|------|------|
| 主路径 Chat 对标（P0–P5 + 密度 + purge + 栏宽 + stop + Markdown） | ✅ |
| **S3 Workspace session DnD** | ✅ 组内拖拽 → `PATCH sessionIds` |
| **S4 MCP 设置面板** | ✅ Agent「设置 → MCP」；`PATCH /mcp/tools/:id` 启停；登记表单 |
| 完整 slash 一比一 / Cordis | ❌ 明确不做 |

**复刻完成度（主路径）≈ 99%** — 剩余仅为可选加深（Markdown 表格、跨 workspace 拖拽归属）。

---

## 后续（可选）

| 项 | 说明 |
|----|------|
| 跨 Workspace 拖拽 | drop 到另一组 → Attach + Detach |
| Markdown 表格 / 任务列表 | BubbleMarkdown 加深 |
| Tools 设置入口 | 设置菜单 Tools 仍占位 |

## 修订

| 日期 | 说明 |
|------|------|
| 2026-09-15 | 落地 S1/S2/S5 |
| 2026-09-15 | 落地 S3 DnD + S4 MCP 面板与 PATCH 启停 |
