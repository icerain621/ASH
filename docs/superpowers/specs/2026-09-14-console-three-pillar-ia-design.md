# ASH 控制台三大主题 IA — 设计规格

> Status: **approved for planning** (2026-09-14；壳层 §1–§4 已口头确认)  
> Program: [`doc/plan/v5-governance-program.md`](../../../doc/plan/v5-governance-program.md)  
> Prior: [`2026-09-13-v5-dual-core-governance-design.md`](./2026-09-13-v5-dual-core-governance-design.md) §7.4 / §8（双模式顶栏 → 本规格升级为三主题）  
> Non-goal: 不引入 Cordis / 不替换为 dsh web（决议 D2 / G1 仍有效）

## 1. Intent

将控制台从「双模式 chip + 平铺多 Tab」升级为 **三大主题卡片切换**，从布局上突出：

1. **Agent** — DSH 模式借鉴的大 Chat 交互（默认首页）  
2. **记忆** — 记忆体 / 知识分层与多视角编排关联  
3. **评审管控** — 评分、启停、监控观测、编排流程  

次要运维页收入「更多」；登录与 Space 设置进入右上角账号下拉。

**保留** 已落地的 v5 能力（Workbench、rubric、多签、SLA、assign、appeal、Registry、Policy、evaluation、Improve、Mobile 薄批准、voided 测评等）。本规格是 **IA / 布局重组**，不是功能回退。

## 2. Decisions (locked)

| ID | Decision |
|----|----------|
| P1 | 主导航仅三主题：**Agent · 记忆 · 评审管控**（卡片/分段切换） |
| P2 | 次要页采用 **「更多」抽屉/下拉**（方案 1）：运行、自动化、反馈、CI、发布、合规、规模化、诊断等 |
| P3 | **登录 / Space 设置** 放入右上角 **账号管理下拉**，与账号管理一致 |
| P4 | 默认打开 **Agent 首页**；主列为 **大 Chat**；**左侧历史会话列表** |
| P5 | Quest 看板 /「从目标创建」降为 Agent 板块 **次级入口**，不删除 |
| P6 | 旧路由深链保留（`/metrics` `/observability` `/knowledge` `/space` …）；仅不再平铺顶栏 |
| P7 | Mobile `/m/reviews` 薄壳保留；三主题优先桌面 |
| P8 | 不引入 Cordis；不替换控制台为 dsh web |

## 3. Non-goals

- 整站换肤 / 新设计系统重做  
- 新记忆存储模型或新编排引擎  
- 砍已交付 v5 API / Doctor / signoff 证据  
- 强制默认 OIDC / 改 Auth 产品语义（账号下拉仅重组入口）

## 4. Global shell (§1)

```text
┌─ Brand ASH ──────────────────────────── 更多 ▾ · 账号 ▾ ─┐
│  [ Agent ]   [ 记忆 ]   [ 评审管控 ]   ← 唯一主导航（主题卡片）
└──────────────────────────────────────────────────────────┘
│                     板块内容区                            │
```

### 4.1 WorkMode

扩展 `frontend/src/app/layout/workMode.ts`：

```ts
export type WorkMode = "agent" | "memory" | "review";
```

| Mode | 默认路由 | 路径识别 |
|------|----------|----------|
| `agent` | `/quest`（可选别名 `/agent` → 同页） | `/quest`、`/agent`、`/runs`（若从 Agent 内打开可仍标 agent） |
| `memory` | `/memory` | `/memory`、`/knowledge` |
| `review` | `/reviews` | `/reviews`、`/m/reviews`；子页 `/metrics` `/observability` `/automation` 等在「评审管控」上下文打开时高亮 review |

持久化 key：可继续用 `ash.console.workMode`，值域扩展为三态。

### 4.2 账号下拉（右上）

- 登录 / 登出  
- 当前 Space 显示与切换  
- Space 设置（深链现有 `/space`）  
- （可选）个人偏好占位  

### 4.3 「更多」下拉/抽屉

至少收录：运行、自动化、反馈、CI、发布、合规、规模化、诊断。  
可不收录：已并入三主题的记忆/知识/评审/观测主路径（观测默认从评审子导航进，仍可进更多作为捷径）。

## 5. Agent pillar (§2) — default home

### 5.1 Layout (default)

```text
┌─ 历史会话（左栏）─┬─ 大 Chat / ConversationNode 投影 ──────────┐
│ session/thread   │  消息流（只渲染投影，不拼系统 prompt）      │
│ 列表 · 新建会话  │                                            │
│                  ├─ composer：prompt | approve/reject/cancel  │
│                  └─ 菜单：Tools · Skills · MCP（侧栏或顶栏）  │
└──────────────────┴─ 次级：任务板 / 从目标创建（折叠或入口）──┘
```

### 5.2 Behavior

- **默认落地**：打开控制台或点 Agent → 本布局；**大 Chat 为主视线**；左侧显示 **历史会话**（复用/扩展 `AgentSessionPanel` 会话列表与 `GET .../sessions/{id}/threads` 等已有能力）。  
- **门禁**：页内 composer takeover；approve **不**强制跳到评审管控。  
- **Tools / Skills / MCP**：设置菜单保留，不占 Chat 主列。  
- **看板 / 从目标创建 Plan**：次级入口（「任务板」面板或链接），功能保留。  
- 意图 API：`prompt|approve|cancel|reject` 不变。

### 5.3 DSH alignment (pattern only)

| DSH 模式 | ASH |
|----------|-----|
| 会话中心 + 底部 composer | 左历史 + 大 Chat + composer |
| Chat ↔ Trajectory | 页内 Timeline/Events（不跳评审） |
| 无 Cordis 壳 | 继续 `/ui` 路由 |

## 6. Memory pillar (§3)

```text
┌─ 视角：分层 | 场景 | Skill | Tools | 项目 ─────────────────┐
│ Tab：记忆体 | 知识                                           │
│ 左：分层树（L0/L1/L2 或候选/正式）+ 资产登记摘要             │
│ 中：列表 / MemoryLink 关联                                   │
│ 右：详情 · TTL · 范围 · 关联 Runs/Threads                    │
└──────────────────────────────────────────────────────────────┘
```

- **分层**为主轴；场景 / skill / tools / 项目为过滤器或视角。  
- 原 `/knowledge` **并入**本板块 Tab，不再占顶栏。  
- 候选厚批准深链 **评审管控**，记忆页不做完整 Workbench。

## 7. Review / governance pillar (§4)

```text
┌─ 子导航：队列评审 | 启停登记 | 监控观测 | 编排流程 ──────────┐
│ 队列评审 = 现 Reviews Workbench（三栏 · rubric · 多签 · SLA · assign · appeal · Improve）
│ 启停登记 = Agent/Memory 资产启停 + Policy/EffectivePolicy 入口（可复用 Space/Registry 面板）
│ 监控观测 = Metrics + Observability（含 review_sla）嵌入或子路由
│ 编排流程 = Runs / Automation / Scenario 薄管控（深链）
└──────────────────────────────────────────────────────────────┘
```

Space 策略编辑：可从「启停登记」或账号下拉「Space 设置」进入同一 `/space`。

## 8. Route map (preserve deep links)

| 旧顶栏 Tab | 新位置 |
|------------|--------|
| Quest | Agent 默认（大 Chat） |
| 记忆 | 记忆主题 |
| 知识 | 记忆主题内 Tab |
| 评审 | 评审管控 · 队列评审 |
| 指标 / 观测 | 评审管控 · 监控观测（+ 可选「更多」捷径） |
| 运行 / 自动化 | 评审 · 编排 或「更多」 |
| 空间 | 账号下拉 · Space 设置 |
| 登录 | 账号下拉 |
| 反馈 / CI / 发布 / 合规 / 规模化 / 诊断 | 「更多」 |

## 9. Implementation outline (for later plan)

1. **Shell**：`AppLayout` 三主题 + 更多 + 账号下拉；扩展 `workMode`；默认重定向 `/` → Agent Chat 布局。  
2. **Agent**：QuestPage（或拆 `AgentHomePage`）改为左会话列表 + 大 Chat；看板折叠为次级。  
3. **Memory**：MemoryPage 加视角切换 + Knowledge Tab 嵌入。  
4. **Review**：Reviews 外壳加子导航；嵌入/路由 Metrics、Observability、Registry 启停。  
5. **Tests**：layout workMode 三态；默认落地 Agent；关键导航不再渲染平铺 Tab。  
6. **Docs**：本规格；v5 程序「下一步」挂 IA 改造；必要时修订 v5 规格 §7.4「双入口」为「三主题」指针。

## 10. Acceptance

- [ ] 顶栏主视线只有三主题（+ 更多 + 账号），无长平铺 Tab  
- [ ] 默认进入 Agent：左侧历史会话 + 大 Chat composer  
- [ ] Tools / Skills / MCP 菜单位可达  
- [ ] 记忆主题可切换分层/场景等视角；知识不再独立顶栏  
- [ ] 评审管控默认 Workbench；监控/启停/编排可从子导航到达  
- [ ] 登录与 Space 设置仅在账号下拉（或等价右上入口）  
- [ ] 旧 URL 深链仍可用；Mobile `/m/reviews` 仍可用  
- [ ] 既有 v5 API/门禁行为不回退  

## 11. Open points (non-blocking)

- `/agent` 是否作为 `/quest` 正式别名（推荐有，利于文案）  
- 「更多」用下拉 vs 右侧抽屉（实现期二选一，默认下拉）  
- Agent 左栏宽度与可折叠断点（实现期跟现有 CSS 变量）

## 12. Revision

| 日期 | 说明 |
|------|------|
| 2026-09-14 | 初稿：三主题 IA；默认 Agent 大 Chat + 左历史；更多 + 账号下拉；保留 v5 |
