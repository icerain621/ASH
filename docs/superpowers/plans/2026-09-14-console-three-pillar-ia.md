# 控制台三主题 IA Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 ASH 控制台主导航改为 Agent · 记忆 · 评审管控 三主题卡片切换；默认 Agent 大 Chat + 左侧历史会话；次要页进「更多」；登录/Space 进账号下拉；保留全部已落地 v5 能力。

**Architecture:** 扩展 `workMode` 三态；重写 `AppLayout` 顶栏；`/` 默认进 `/quest`；Quest 页改为左会话列表 + 大 Chat（看板次级）；Memory 加视角/知识 Tab；Reviews 加板块子导航嵌入观测/启停。不引入 Cordis。

**Tech Stack:** React 18 · TanStack Router/Query · Vite · 现有 `AgentSessionPanel` / Reviews / Memory / Registry 组件

**Spec:** [`docs/superpowers/specs/2026-09-14-console-three-pillar-ia-design.md`](../specs/2026-09-14-console-three-pillar-ia-design.md)

## Global Constraints

- 不删除旧路由深链（`/metrics` `/knowledge` `/space` 等仍可打开）
- 不回退 v5 API/Workbench/rubric/多签/SLA/assign/appeal
- Mobile `/m/reviews` 保持薄壳
- 中文 UI 文案；`data-testid` 覆盖主导航与 Agent 壳
- Commit 消息中文；不提交 `doc/evidence/device-session-smoke-latest.md`
- 验证：相关 vitest + `make web-build`（或 npx tsc/vite）

---

### Task 1: Shell — 三主题 + 更多 + 账号下拉

**Files:**
- Modify: `frontend/src/app/layout/workMode.ts`
- Modify: `frontend/src/app/layout/workMode.test.ts`
- Modify: `frontend/src/app/layout/AppLayout.tsx`
- Modify: `frontend/src/app/router.tsx`（`/` → `/quest`）
- Modify: `frontend/src/styles/globals.css`（主题卡片样式；弱化/隐藏旧 `.tabs` 平铺）
- Create: `frontend/src/app/layout/AppLayout.test.tsx`（若尚无）

**Interfaces:**
- Produces: `WorkMode = "agent" | "memory" | "review"`；`workModeFromPath` 识别三主题路径；`persistWorkMode` / `readPersistedWorkMode` 支持三态

- [x] **Step 1:** 扩展 `workMode.ts` + 单测（agent/memory/review 路径映射；旧 `"review"` 持久化兼容）
- [x] **Step 2:** 重写 `AppLayout`：三主题分段（链接 `/quest` `/memory` `/reviews`）；「更多」下拉含运行/自动化/反馈/CI/发布/合规/规模化/诊断；右上「账号」下拉含登录、Space 设置（`/space`）、当前 spaceId 展示；**移除**顶栏长 Tab 列表
- [x] **Step 3:** `indexRoute` 改为 `Navigate to="/quest"`
- [x] **Step 4:** vitest：`workMode.test.ts` + layout 冒烟（三主题 testid、无平铺 tabs）
- [x] **Step 5:** Commit `feat(fe): 控制台三主题壳层与账号/更多下拉`

**Verify:**
```bash
cd frontend && npx vitest run src/app/layout/workMode.test.ts src/app/layout/AppLayout.test.tsx
```

---

### Task 2: Agent — 大 Chat + 左历史会话

**Files:**
- Modify: `frontend/src/pages/QuestPage.tsx`（主布局重组）
- Modify: `frontend/src/modules/agent-session/components/AgentSessionPanel.tsx`（可选：变高填充）
- Modify: `frontend/src/styles/globals.css`（`.agent-home` 左右栏）
- Modify/Create: `frontend/src/pages/QuestPage.test.tsx`
- Optional alias: `frontend/src/app/router.tsx` 增加 `/agent` → 同 `QuestPage`

**Interfaces:**
- Consumes: `getQuestBoard` 作为历史会话数据源（runId/plan 条目）；`AgentSessionPanel({ runId })`
- Produces: 左栏选中 `selectedRunId`；无选中时显示空态「选择或新建会话」+ 保留「从目标创建」入口

- [x] **Step 1:** Quest 默认视图：`data-testid="agent-home"`；左栏 `agent-session-history` 列出看板各列条目（标题+状态）；点击绑定右侧 `AgentSessionPanel`
- [x] **Step 2:** 右侧为大 Chat 区（现有 ConversationThread + IntentBar）；Tools/Skills/MCP：顶栏或右上菜单链到现有 skills/相关页或占位菜单项（`data-testid="agent-settings-menu"`，含 Tools/Skills/MCP 三项；Skills 可链 `/automation` 或文档说明——优先链已有技能相关路由若存在，否则按钮 + toast/muted「在会话侧栏配置」占位但三项可见）
- [x] **Step 3:** 看板与「从目标创建」收入可折叠「任务板」次级区（默认折叠，`data-testid="agent-task-board-toggle"`）
- [x] **Step 4:** vitest：渲染左历史 + 大 Chat 壳；任务板默认折叠
- [x] **Step 5:** Commit `feat(fe): Agent 首页大 Chat 与左侧历史会话`

**Verify:**
```bash
cd frontend && npx vitest run src/pages/QuestPage.test.tsx src/modules/agent-session/
```

**Note:** 暂无新增 `GET /agents/sessions` 列表 API；历史 = Quest board 投影。后续可换真会话列表。

---

### Task 3: Memory — 视角切换 + 知识 Tab

**Files:**
- Modify: `frontend/src/pages/MemoryPage.tsx`
- Modify: `frontend/src/pages/MemoryPage.test.tsx`（若有）或新建
- Reuse: `KnowledgePage` 组件嵌入或抽取共享内容

- [x] **Step 1:** Memory 顶加 Tab：`记忆体 | 知识`（`data-testid="memory-pillar-tabs"`）；知识 Tab 渲染现有 Knowledge 页面内容（import 组件或 iframe 式复用 export）
- [x] **Step 2:** 视角切换：`分层 | 场景 | Skill | Tools | 项目`（`data-testid="memory-perspective"`）；分层为默认；其它视角先做过滤器 UI（按现有 list 字段能滤则滤，否则显示提示但仍切换高亮）
- [x] **Step 3:** 深链「去评审」按钮指向 `/reviews`（候选批准）
- [x] **Step 4:** vitest + Commit `feat(fe): 记忆板块多视角与知识并入`

---

### Task 4: Review — 板块子导航

**Files:**
- Modify: `frontend/src/pages/ReviewsPage.tsx`（或新建 `ReviewPillarLayout` 包装）
- Modify: `frontend/src/app/router.tsx`（可选：`/reviews/metrics` 等子路径；或页内 state 切换）
- Reuse: MetricsPage / ObservabilityPage / RegistryAssetsPanel

- [x] **Step 1:** Reviews 顶加子导航：`队列评审 | 启停登记 | 监控观测 | 编排流程`（`data-testid="review-pillar-nav"`）
- [x] **Step 2:** 默认队列评审 = 现有 Workbench 全文保留
- [x] **Step 3:** 启停登记 = 嵌入 `RegistryAssetsPanel` + EffectivePolicy 摘要（从 Space API）；监控观测 = 嵌入或 Link 到 Metrics/Observability 内容；编排流程 = 链到 `/runs` `/automation` 的内嵌说明+按钮
- [x] **Step 4:** vitest + Commit `feat(fe): 评审管控板块子导航`

---

### Task 5: 文档收口 + web-build + push

**Files:**
- Modify: `doc/plan/v5-governance-program.md`（IA 进行中）
- Modify: spec status → `approved / implementing`
- Optional: v5 规格 §7.4 加指针到三主题规格

- [x] **Step 1:** `cd frontend && npx vitest run src/app/layout src/pages/QuestPage.test.tsx src/pages/ReviewsPage.test.tsx src/pages/MemoryPage.test.tsx`（按实际文件调整）
- [x] **Step 2:** `make web-build`
- [x] **Step 3:** 更新程序状态；Commit docs；`git push origin HEAD`

---

## Spec coverage checklist

| Spec | Task |
|------|------|
| P1 三主题 | T1 |
| P2 更多 | T1 |
| P3 账号下拉 | T1 |
| P4 默认大 Chat+左历史 | T1 index + T2 |
| P5 看板次级 | T2 |
| P6 深链保留 | T1（不删路由） |
| §3 记忆 | T3 |
| §4 评审子导航 | T4 |
| 保留 v5 | T4 默认队列不改行为 |

## Execution

用户已要求推进：采用 **Subagent-Driven**，按 Task 1→5 连续执行，任务间不暂停询问。
