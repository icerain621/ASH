# Agent Chat DSH Parity Implementation Plan

> **For agentic workers:** Use subagent-driven-development. Checkbox steps.

**Goal:** ASH Agent 主题一比一复刻 DSH Chat 布局与主交互（三栏、真会话、气泡、sticky composer、Chat|Trajectory、Details），不引入 Cordis。

**Spec:** [`docs/superpowers/specs/2026-09-15-agent-chat-dsh-parity-design.md`](../specs/2026-09-15-agent-chat-dsh-parity-design.md)

**Tech:** Go session 包 · Gin · React · 现有 IntentBar/ThreadTimeline

## Global Constraints

- 空白会话可 Create（已支持）；须 List + FE 主路径
- 保留任务板次级；保留 v5 三主题壳
- 中文 commit；不提交 device-session-smoke-latest.md
- 验证：`go test ./internal/session/... ./internal/api/...` + vitest agent/quest + web-build

---

### Task 1: Backend ListSessions + blank events

**Files:** `internal/session/service.go`, `list` tests, `internal/api/session.go`, `handlers.go`, openapi/swagger

- [ ] `List(spaceID, limit) ([]View, error)` — query audit_log `event_type=agent.session` AND space_id
- [ ] `GET /api/v1/agents/sessions` query spaceId/limit
- [ ] `ListEvents`: if no runId, synthesize items from `view.Turns` as `session.turn` envelopes (so Chat 有内容)
- [ ] Tests + swagger + openapi-check
- [ ] Commit `feat(session): 列出 Agent Session 并支持空白会话事件投影`

---

### Task 2: FE session API + AgentChatShell

**Files:** `session.api.ts`, new components under `modules/agent-session/`, `QuestPage.tsx`, CSS, tests

- [ ] `listAgentSessions()`
- [ ] `AgentChatShell`: 左 SessionHistoryList（API）+ 中 Chat|Trajectory + 右 Details（可关）
- [ ] 新建：`createAgentSession({})` → 选中新 id，不打开任务板
- [ ] `ChatTranscript`：user 右气泡 / assistant·tool·gate 左卡片（从 turns+events 映射）
- [ ] sticky composer 样式；gate takeover 保留
- [ ] TrajectoryTab：ThreadTimeline 或 event list
- [ ] Details：点击节点显示 JSON/摘要
- [ ] QuestPage：默认 shell；任务板仍折叠
- [ ] vitest + Commit `feat(fe): Agent Chat 三栏对标 DSH`

---

### Task 3: Docs + web-build + push

- [ ] 更新三主题规格指针 / v5 program
- [ ] `make web-build`；硬刷新说明
- [ ] Push

---

执行：用户已要求推进 → Subagent-Driven，连续 Task 1→3。
