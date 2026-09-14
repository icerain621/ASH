# Agent Chat DSH 对标 — 后端缺失功能清单

> Status: **living backlog**（2026-09-15）  
> Spec: [`2026-09-15-agent-chat-dsh-parity-design.md`](./2026-09-15-agent-chat-dsh-parity-design.md)  
> 已落地：`874b0ce` ListSessions · `6e88718` 三栏 Chat 壳 · **本轮** title/PATCH/DELETE/`stop` 别名 + FE SSE/工具卡/停止/改名关闭 · P2 assistant 流 · P3 Workspace `f711d93` / FE `7c29e5f`
> 原则：不引入 Cordis / 不嵌入 `dsh web`

本文记录 **相对 DSH Agent Chat 仍缺的后端能力**，供续推复刻时排期；FE 可先用现有 API 做近似体验的项另标。

---

## P0 — 直播感 / 工具卡 / 停止（本轮优先）

| 缺口 | 现状 | 建议 API / 行为 |
|------|------|-----------------|
| Session 级 live 流 | ✅ FE：有 `runId` 时 `useRunStream` 合并 transcript；无 run 仍轮询 events。可选 session stream 代理仍未做 | FE 优先 SSE；可选 `GET /agents/sessions/{id}/stream` 代理（无 runId 时 404，空白会话仍靠 turns 轮询） |
| Stop 生成 | ✅ `action:"stop"` 别名 `cancel`；FE IntentBar/Composer 运行中显示停止 | 完成 |
| Tool 事件可见性 | ✅ FE `conversationNodes` + tool bubble；仍依赖 run ledger 事件质量 | 保证 `tool.called`/`tool.result`/`step.*` 对 UI 可见（`ui_only`/`model_visible`）；无新表 |

## P1 — 会话侧栏卫生（DSH WorkspaceBrowser 子集）

| 缺口 | 现状 | 建议 API |
|------|------|----------|
| 会话标题 | ✅ View.`title`；Create(goal)/PromptTurn 空则截断首行 ~48 runes | 完成 |
| 重命名 | ✅ `PATCH /agents/sessions/{id}` `{title}` + 审计 `agent.session_updated`；FE 侧栏改名 | 完成 |
| 关闭/删除 | ✅ soft `DELETE`→`status=closed`；List 默认排除，`?includeClosed=1`；FE 关闭 | 硬删除 / purge 仍不做 |

## P2 — 助手流式正文（打字机对标阻塞项）

| 缺口 | 现状 | 建议 |
|------|------|------|
| Token/文本增量 | ✅ `assistant.delta` `{turnId,text,index}` + `assistant.message` `{turnId,text,stopped,source}`；有 runId 时写入 run ledger（model_visible） | 完成（形状已落地；真 LLM token 流仍后置） |
| 空白会话助手回复 | ✅ 无 runId 时 echo stub `已收到：{prompt}`（`source:"echo"`）存 `View.replies`，`ListEvents`/`synthesizeTurnEvents` 投影 delta+message；有 ACP `acpMessage` 则 `source:"acp"` | echo stub 直至真 LLM stream |

**Stop 说明**：Intent `stop`/`cancel` 仅取消绑定 run；当前 echo/ACP 回包同步完成，故 `stopped:true` 预留给未来 mid-flight 流。

## P3 — Workspace 分组（非 Cordis）

| 缺口 | 建议 |
|------|------|
| ASH 原生 workspace | ✅ `GET/POST /agent-workspaces`；`PATCH` 改名/`sessionIds`；Create session 接受 `workspaceId`；`repoRoot`≈ cwd；FE 侧栏按 Workspace 分组 |

## P4 — Slash / Skills 命令面

| 缺口 | 建议 |
|------|------|
| 命令目录 | `GET /agents/commands` → `{ items: [{ name, description, source }] }` |
| 执行命令 | `actions` `{ action:"command", command:"/foo", args }`；未知命令 fail-closed |

## P5 — Model / Plan / Permission seats

| 缺口 | 建议 |
|------|------|
| 座位字段 | PATCH：`providerKind` / `planId` / `permissionMode` |
| 模型列表 | `GET /agents/models` 或复用 harness profile |
| Plan seat | 挂现有 Goal/Plan id，不做 DSH Cordis plan 面板 |

## 明确不做（仍）

- Cordis / `dsh web` iframe  
- 完整 slash 目录一比一  
- 栏宽拖拽 / workspace DnD（纯 FE 可后补）

## 修订

| 日期 | 说明 |
|------|------|
| 2026-09-15 | 初稿：P0–P5 缺口与建议形状 |
| 2026-09-15 | 落地 P0 FE（SSE/工具卡/Stop）+ P1 title/PATCH/DELETE/`stop` 别名；P2–P5 仍开 |
| 2026-09-15 | 落地 P2：`assistant.delta`/`assistant.message` + 空白会话 echo stub；真 LLM 流仍开 |
| 2026-09-15 | 落地 P3：Agent Workspace API `f711d93` + Chat 侧栏按 Workspace 分组 `7c29e5f` |
