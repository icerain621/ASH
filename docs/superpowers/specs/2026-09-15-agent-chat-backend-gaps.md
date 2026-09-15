# Agent Chat DSH 对标 — 后端缺失功能清单

> Status: **living backlog**（2026-09-15）  
> Spec: [`2026-09-15-agent-chat-dsh-parity-design.md`](./2026-09-15-agent-chat-dsh-parity-design.md)  
> 已落地：`874b0ce` ListSessions · `6e88718` 三栏 Chat 壳 · **本轮** title/PATCH/DELETE/`stop` 别名 + FE SSE/工具卡/停止/改名关闭 · P2 assistant 流（**真 LLM / Provider**）· P3 Workspace `f711d93` / FE `7c29e5f` · **P4/P5** slash commands + model/permission seats · **session stream SSE** + **skill 命令执行** · **MCP slash 目录+执行** · **FE 视觉密度贴合 DSH**  
> 原则：不引入 Cordis / 不嵌入 `dsh web`

本文记录 **相对 DSH Agent Chat 仍缺的后端能力**，供续推复刻时排期；FE 可先用现有 API 做近似体验的项另标。

---

## P0 — 直播感 / 工具卡 / 停止（本轮优先）

| 缺口 | 现状 | 建议 API / 行为 |
|------|------|-----------------|
| Session 级 live 流 | ✅ `GET /agents/sessions/{id}/stream`；有 runId 复用 run ledger SSE；空白会话轮询 `ListEvents`（~1s，支持 afterSeq / Last-Event-ID）。View.`streamUrl` 恒为会话路径；FE `useSessionStream` 优先 | 完成 |
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
| Token/文本增量 | ✅ `assistant.delta` `{turnId,text,index}` + `assistant.message` `{turnId,text,stopped,source}`；有 runId 时写入 run ledger（model_visible） | 完成 |
| 真 LLM / Provider | ✅ 优先级：`ASH_LLM_*` OpenAI 兼容 Chat Completions（SSE `stream:true`，失败回落非流式）→ `agentexec.Resolve` Provider（static / acp_sdk / execgo）`StdoutSummary` → echo stub | 完成 |
| 空白会话助手回复 | ✅ 无 runId 时：LLM/`source:"llm"`、Provider/`source:<adapter>`、或 echo `已收到：{prompt}`（`source:"echo"`）存 `View.replies`，`ListEvents` 投影 delta+message | 完成 |

### 启用真 LLM（`internal/llmchat`）

| 环境变量 | 说明 |
|----------|------|
| `ASH_LLM_BASE_URL` | **必填**才走 LLM；可为 origin、`.../v1` 或完整 `.../chat/completions` |
| `ASH_LLM_API_KEY` | 可选；设置则带 `Authorization: Bearer` |
| `ASH_LLM_MODEL` | 默认 `gpt-4o-mini` |
| `ASH_LLM_TIMEOUT` | Go duration，默认 `60s` |

未设置 `ASH_LLM_BASE_URL` 时：有 `providerKind` 则跑 generalized provider；否则 echo。

**Stop 说明**：Intent `stop`/`cancel` 仅取消绑定 run；当前 LLM/provider/echo 回包仍在 PromptTurn 内同步完成，故 `stopped:true` 预留给未来 mid-flight cancel。

## P3 — Workspace 分组（非 Cordis）

| 缺口 | 建议 |
|------|------|
| ASH 原生 workspace | ✅ `GET/POST /agent-workspaces`；`PATCH` 改名/`sessionIds`；Create session 接受 `workspaceId`；`repoRoot`≈ cwd；FE 侧栏按 Workspace 分组 |

## P4 — Slash / Skills / MCP 命令面

| 缺口 | 现状 | 建议 |
|------|------|------|
| 命令目录 | ✅ `GET /agents/commands` → `{ items: [{ name, description, source }] }`（builtin + skills + space `mcp_tools`，status 非空且非 `disabled`） | 完成 |
| 执行命令 | ✅ `action:"command"`：`/help` `/clear` + skill + **MCP**（`toolbus` `mcp.call`；成功/失败均 chat 可见，`source:"mcp"`；有 runId 时 `tool.called`/`tool.result` ui_only）；未知仍 409 | 完成 |

## P5 — Model / Plan / Permission seats

| 缺口 | 现状 | 建议 |
|------|------|------|
| 座位字段 | ✅ PATCH：`providerKind` / `planId` / `permissionMode`；View.`permissionMode` | 完成 |
| 模型列表 | ✅ `GET /agents/models`（static / acp_sdk / execgo） | 完成 |
| Plan seat | ✅ 挂 `planId` 文本座位；不做 DSH Cordis plan 面板 | 完成 |

## FE 视觉密度（DSH-aligned，无 Cordis）

| 项 | 现状 |
|----|------|
| 中栏内容宽 | ✅ `--ash-chat-content-width: 748px` 居中；composer ≈ content+32px，侧边 16px |
| 气泡 / 空态 | ✅ user pill 右对齐；assistant 更软边框 + line-height 1.5；默认隐藏 raw event type；空态 hero 居中 |
| 侧栏 / sticky | ✅ 更密行；transcript 底 padding 避挡 sticky composer |

## 明确不做（仍）

- Cordis / `dsh web` iframe  
- 完整 slash 目录一比一  

## 仍开（高价值）

| 缺口 | 说明 |
|------|------|
| 硬删除 / purge | soft-close only；物理 purge 仍不做 |
| 栏宽拖拽 / workspace DnD | 纯 FE 可后补 |

## 修订

| 日期 | 说明 |
|------|------|
| 2026-09-15 | 初稿：P0–P5 缺口与建议形状 |
| 2026-09-15 | 落地 P0 FE（SSE/工具卡/Stop）+ P1 title/PATCH/DELETE/`stop` 别名；P2–P5 仍开 |
| 2026-09-15 | 落地 P2：`assistant.delta`/`assistant.message` + 空白会话 echo stub；真 LLM 流仍开 |
| 2026-09-15 | 落地 P3：Agent Workspace API `f711d93` + Chat 侧栏按 Workspace 分组 `7c29e5f` |
| 2026-09-15 | 落地 P4/P5：`/agents/commands` + Intent `command`；`/agents/models` + PATCH seats；FE 命令菜单与座位行；真 LLM 流仍开 |
| 2026-09-15 | 落地 P2 余量：`internal/llmchat` + PromptTurn LLM→Provider→echo；gaps 标注真流已落地；session stream 路由仍可选 |
| 2026-09-15 | 落地 session stream SSE + skill 命令执行；FE 优先会话级 SSE；MCP exec 仍开 |
| 2026-09-15 | 落地 MCP slash 目录+执行；FE Agent Chat 视觉密度贴合 DSH；仍开 = 硬 purge / 栏宽拖拽 |
