# Agent Chat DSH 对标 — 后端缺失功能清单

> Status: **living backlog**（2026-09-15）  
> Spec: [`2026-09-15-agent-chat-dsh-parity-design.md`](./2026-09-15-agent-chat-dsh-parity-design.md)  
> 已落地：三栏壳 · P0–P5 · MCP · 视觉密度 · 硬 purge · 栏宽拖拽 · **mid-flight stop** · **purge Detach** · **气泡 Markdown**  
> 原则：不引入 Cordis / 不嵌入 `dsh web`

---

## 进度总览（2026-09-15）

| 阶段 | 状态 |
|------|------|
| 基础三栏壳 + 真 Session 列表 | ✅ |
| P0–P5 / MCP / 视觉密度 / purge / 栏宽 | ✅ |
| **S1 mid-flight LLM cancel** | ✅ `flights` 注册表；空白会话 `stop` 取消进行中生成；`stopped:true`；取消时不回落 Complete |
| **S5 purge Detach workspace** | ✅ purge 时 `Detach` workspace.sessionIds |
| **S2 气泡 Markdown** | ✅ 轻量 fences / `code` / **bold** / *italic*（无新依赖） |
| S3 Workspace session DnD | ⬜ |
| S4 MCP 设置一等面板 | ⬜ |

**复刻完成度（主路径）≈ 97%**

---

## P0–P5（摘要）

均已完成，详见历史修订。Stop：有 run 取消 run；无 run 取消 in-flight PromptTurn。

## FE 视觉 / 栏宽 / Markdown

| 项 | 现状 |
|----|------|
| 密度 / 栏宽拖拽 | ✅ |
| 气泡 Markdown | ✅ `BubbleMarkdown` |

## 明确不做（仍）

- Cordis / `dsh web` iframe  
- 完整 slash 目录一比一  

## 后续计划

| 优先级 | 项 | 说明 |
|--------|----|------|
| **S3** | Workspace session DnD | FE 拖拽重排 → PATCH sessionIds |
| **S4** | MCP 设置面板 | 登记/启停 mcp_tools |
| — | Markdown 表格/任务列表 | 可选加深 |
| — | S1/S2/S5 | ✅ 本轮 |

## 修订

| 日期 | 说明 |
|------|------|
| 2026-09-15 | 落地硬 purge + 栏宽拖拽 |
| 2026-09-15 | 落地 S1 mid-flight stop、S5 purge Detach、S2 气泡 Markdown |
