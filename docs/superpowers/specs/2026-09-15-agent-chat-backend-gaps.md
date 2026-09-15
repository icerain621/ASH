# Agent Chat DSH 对标 — 后端缺失功能清单

> Status: **living backlog**（2026-09-15）  
> Spec: [`2026-09-15-agent-chat-dsh-parity-design.md`](./2026-09-15-agent-chat-dsh-parity-design.md)  
> 已落地：`874b0ce` ListSessions · `6e88718` 三栏 Chat 壳 · title/PATCH/DELETE/`stop` · P2 assistant 流（真 LLM/Provider）· P3 Workspace · P4/P5 slash+seats · session SSE + skill · **MCP slash** · **FE 视觉密度** · **硬 purge** · **栏宽拖拽**  
> 原则：不引入 Cordis / 不嵌入 `dsh web`

本文记录 **相对 DSH Agent Chat 仍缺的后端能力**，供续推复刻时排期；FE 可先用现有 API 做近似体验的项另标。

---

## 进度总览（2026-09-15）

| 阶段 | 状态 | 说明 |
|------|------|------|
| 基础三栏壳 + 真 Session 列表 | ✅ | Task 1–2 验收全勾 |
| P0 直播感 / 工具卡 / Stop | ✅ | session stream + tool bubble + stop 别名 |
| P1 侧栏卫生 | ✅ | title / rename / soft-close |
| P2 助手流 | ✅ | delta/message + LLM→Provider→echo |
| P3 Workspace 分组 | ✅ | API + FE 分组 |
| P4 Slash / Skill / MCP | ✅ | commands 目录 + command 执行 |
| P5 Model / Plan / Permission seats | ✅ | PATCH + `/agents/models` |
| FE 视觉密度贴 DSH | ✅ | 748px 内容宽 / 气泡 / sticky |
| 硬 purge | ✅ | `DELETE ?purge=1` + FE「删除」确认 |
| 栏宽拖拽 | ✅ | 左右 separator + localStorage |
| mid-flight LLM cancel | ⬜ | Stop 仅绑 run；空白会话 PromptTurn 同步完成 |
| Workspace DnD 调序 | ⬜ | 纯 FE；非阻塞 |
| 完整 Markdown / slash 一比一 | ⬜ | 明确不做完整目录；Markdown 可加深 |

**复刻完成度（主路径）≈ 95%**：DSH Chat 壳与主交互已对齐；剩余为体验加深项。

---

## P0 — 直播感 / 工具卡 / 停止

| 缺口 | 现状 | 建议 API / 行为 |
|------|------|-----------------|
| Session 级 live 流 | ✅ `GET /agents/sessions/{id}/stream` | 完成 |
| Stop 生成 | ✅ `action:"stop"` 别名 `cancel` | 完成 |
| Tool 事件可见性 | ✅ FE tool bubble；依赖 run ledger | 无新表 |

## P1 — 会话侧栏卫生

| 缺口 | 现状 | 建议 API |
|------|------|----------|
| 会话标题 / 重命名 | ✅ | 完成 |
| 关闭/删除 | ✅ soft `DELETE`；硬 `DELETE ?purge=1` → 删 `agent.session` audit 行，审计 `agent.session_purged`；FE「关闭」+「删除」 | 完成 |

## P2 — 助手流式正文

| 缺口 | 现状 | 建议 |
|------|------|------|
| Token/文本增量 | ✅ | 完成 |
| 真 LLM / Provider | ✅ `ASH_LLM_*` → Provider → echo | 完成 |
| mid-flight cancel | ⬜ `stopped:true` 预留；需 PromptTurn 可取消 context | 见后续计划 |

## P3 — Workspace 分组

| 缺口 | 建议 |
|------|------|
| ASH 原生 workspace | ✅ |

## P4 — Slash / Skills / MCP

| 缺口 | 现状 |
|------|------|
| 命令目录 + 执行 | ✅ builtin / skill / MCP |

## P5 — Model / Plan / Permission seats

| 缺口 | 现状 |
|------|------|
| 座位字段 / 模型列表 / planId | ✅ |

## FE 视觉密度 + 栏宽

| 项 | 现状 |
|----|------|
| 中栏内容宽 / 气泡 / 空态 | ✅ |
| 栏宽拖拽 | ✅ `--ash-sidebar-width` / `--ash-details-width`；`localStorage`；separator |

## 明确不做（仍）

- Cordis / `dsh web` iframe  
- 完整 slash 目录一比一  

## 后续计划（按价值）

| 优先级 | 项 | 说明 | 预估 |
|--------|----|------|------|
| **S1** | mid-flight LLM/Provider cancel | PromptTurn 持 `context.Cancel`；`stop` 取消进行中的空白会话生成并写 `assistant.message` `stopped:true` | M |
| **S2** | 助手 Markdown 渲染加深 | Chat 气泡 GFM（代码块/列表）；不引入 Cordis | S |
| **S3** | Workspace session DnD 调序 | FE 拖拽重排 `sessionIds` → PATCH workspace | S |
| **S4** | MCP 设置一等面板 | 控制台登记/启停 mcp_tools（已有 API 可接） | M |
| **S5** | purge 清理 workspace.sessionIds | Purge 时 Detach；避免幽灵 id | S |
| — | 栏宽 / 硬 purge | ✅ 本轮已关 | — |

## 修订

| 日期 | 说明 |
|------|------|
| 2026-09-15 | 初稿：P0–P5 缺口与建议形状 |
| 2026-09-15 | 落地 P0–P5、MCP、视觉密度；仍开 = 硬 purge / 栏宽拖拽 |
| 2026-09-15 | 落地硬 purge + 栏宽拖拽；补充进度总览与 S1–S5 后续计划 |
