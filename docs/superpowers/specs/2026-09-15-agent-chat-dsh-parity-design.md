# ASH Agent Chat × DSH 视觉/交互对标 — 设计规格

> Status: **landed** (2026-09-15；Task 1 `874b0ce` · Task 2 `6e88718`)  
> Prior: [`2026-09-14-console-three-pillar-ia-design.md`](./2026-09-14-console-three-pillar-ia-design.md) · DSH 源码 `deepseek-harness` `ui-layout` / `ui-conversation` / `ui-workspace` / `ui-trajectory`  
> Non-goal: **不引入 Cordis、不嵌入 `dsh web`**（G1/D2）；Goal→Plan 看板保留为次级

## 1. Intent

在 ASH 控制台 Agent 主题内，**一比一复刻 DSH Agent Chat 页的布局与主交互**（三栏壳、真会话历史、气泡 Chat、sticky composer、Chat|Trajectory、Details），底层仍用 ASH Session/Run/Events API。

## 2. Decisions

| ID | Decision |
|----|----------|
| C1 | **新建会话** = 空白 Agent Session（无强制 Goal→Plan）；可直接打字发 `prompt` |
| C2 | 左栏数据源 = `GET /agents/sessions`（audit `agent.session`），不再用 Quest board 冒充历史 |
| C3 | 三栏：Sidebar(会话) \| Conversation(Chat\|Trajectory) \| Details(可关) |
| C4 | Chat = 气泡/Markdown 风格节点；Trajectory = 事件时间线（复用 ThreadTimeline / session events） |
| C5 | Composer sticky 底部；gate 时 takeover 覆盖输入（保留 approve/reject/cancel） |
| C6 | Goal→Plan / 看板 = 「任务板」次级（已有折叠），不挡 Chat |
| C7 | Tools 在 Chat 流内可见（事件投影为 tool 卡）；Skills/MCP 设置入口保留并逐步加深 |

## 3. Layout (target)

```text
┌─ Sidebar ~280 ──┬─ Conversation (flex) ──────────┬─ Details ~320 ─┐
│ Brand/New       │ Header: title · Chat|Trajectory│ Tool/事件详情   │
│ Session list    │ Transcript (bubbles)           │ 可关闭          │
│                 │ sticky Composer / Gate takeover│                 │
└─────────────────┴────────────────────────────────┴─────────────────┘
         └─ 次级：任务板（折叠）
```

## 4. API

| Method | Path | Behavior |
|--------|------|----------|
| GET | `/api/v1/agents/sessions?spaceId=&limit=` | 列当前 space 的 `agent.session`，按 updatedAt 降序 |
| POST | `/api/v1/agents/sessions` | **已支持**无 runId/goal 的空白会话；保持 |
| POST | `.../actions` `prompt` | 空白会话亦写入 turn；有 run 时继续挂 run events |
| GET | `.../events` | 无 runId 时返回由 turns/session 合成的投影条目（或空 + FE 用 `view.turns`） |

权限：list/get 用 `runs:read` 或现有 session 读权限；create 保持 `runs:create`。

## 5. Frontend modules

| Module | Role |
|--------|------|
| `AgentChatShell` | 三栏布局 |
| `SessionHistoryList` | 左栏；选中 sessionId；新建 |
| `ChatTranscript` | 气泡渲染（user/assistant/tool/gate） |
| `ChatComposer` | sticky IntentBar 升级样式 |
| `TrajectoryPane` | 事件 ledger |
| `DetailsPane` | 选中节点详情 |
| `QuestPage` | 组装 shell + 折叠任务板 |

## 6. Acceptance

- [x] 默认 Agent：三栏 Chat 壳（Details 默认可开或可关，但入口存在）
- [x] 新建 → 空白会话 → 直接发送意图，不强制打开 Goal 表单
- [x] 左栏为真 session 列表（刷新后仍在）
- [x] Chat 为气泡流（非纯 event-line 运维列表）
- [x] Chat \| Trajectory 页内 Tab
- [x] Composer sticky；gate takeover
- [x] 任务板默认折叠且可达
- [x] 不引入 Cordis / dsh web iframe

## 7. Out of scope (本轮)

- DSH `/` slash 命令目录完整复刻  
- Workspace 多级树 / session DnD 调序（栏宽拖拽已落地）  
- mid-flight LLM cancel（预留 `stopped`）  
- MCP 一等设置面板  

## 8. Revision

| 日期 | 说明 |
|------|------|
| 2026-09-15 | 初稿：对标验收 + 空白会话默认 + API/FE 切面 |
| 2026-09-15 | **落地**：BE ListSessions + 空白事件投影 `874b0ce`；FE 三栏壳 `6e88718`；验收项全部勾选 |
| 2026-09-15 | 续推：P0–P5 / MCP / 密度 / 硬 purge / 栏宽拖拽；详见 backend-gaps |
