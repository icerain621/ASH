# ASH 功能 / 接口盘点（原型对照）

> 日期：2026-09-18  
> 对照：[`doc/prototypes/agent-absorb`](../prototypes/agent-absorb/) · 生产 Console `frontend/src` · OpenAPI `doc/api/openapi-ash-v1.yaml` · [`platform-quad-comparison.md`](./platform-quad-comparison.md)  
> 工时单位：**人时**（熟悉仓内同学 · FE+BE 合计切片；不含联调排期缓冲）  
> 状态：`有` 已交付可用 · `待改进` 已有但相对原型/对标不足 · `待开发` 原型或 v6 吸收项尚未产品化

## 1. 总览

| 一级模块 | 能力条目 | 有 | 待改进 | 待开发 | 待投入合计 (人时) |
|----------|----------|----|--------|--------|-------------------|
| Agent 薄交互 | 12 | 8 | 3 | 1 | ~80 |
| 记忆双核 | 13 | 8 | 3 | 2 | ~160 |
| 管控&评审 | 14 | 5 | 3 | 6 | ~640 |
| 平台 / 设置 / 运维 | 12 | 10 | 2 | 0 | ~40 |
| **合计** | **51** | **31** | **11** | **9** | **~920** |

交付优先级（建议波次）：

| 波次 | 范围 | 人时 | 说明 |
|------|------|------|------|
| **W0 · 硬化** | 审批预设深化 · MCP exec · Agent 空态/审批 UX · OpenAPI 漂移 | ~160 | 不改产品边界，先把门禁与 Chat 壳对齐原型 |
| **W1 · v6.0 吸收** | Hooks · execpolicy · Steer/Queue | ~376 | 对标报告 P0 |
| **W2 · v6.1 吸收** | 会话树 fork/compare · Compact · 记忆视角回填 | ~264 | 对标报告 P1 |
| **W3 · v6.2+** | 子代理谱系 UI · Skills/MCP/Hooks 闭环 · Provider/Doctor | ~160 | 对标报告 P1–P2 |

---

## 2. 业务场景归类

| 场景 ID | 场景 | 主模块 | 关键能力 | 现状 |
|---------|------|--------|----------|------|
| S01 | 空会话启动任务（编程/通用） | Agent | 空态 · Composer · 模式 · 模型 · 审批 | **有**（EW03 空态 + agentMode） |
| S02 | 运行中打断 / 后续排队 | 管控 | Steer / Queue | **有**（steer/queue + Chat 投影） |
| S03 | 危险工具审批 | Agent + 管控 | waiting_approval · 审批预设 · SpacePolicy | **有**（once/会话/空间预设 EW04） |
| S04 | 长会话压缩可回放 | 管控 | Compact / spill | **有**（`/compact` + Trajectory/Details） |
| S05 | 分支探索 / 对比 | 管控 | Fork · Compare · 会话树 | **有**（Fork/Compare；独立事件流待深化） |
| S06 | 子代理协作 | 管控 + Runs | sub-run · 谱系 | **有**（Trajectory 派生 + 谱系列表） |
| S07 | 记忆候选薄批 → 厚评审 | 记忆 → 管控 | candidates · review · reviews queue | **有**（薄批 + `?queue=memory&memoryId=` 厚审深链） |
| S08 | 检索已批准记忆 / hit_used | 记忆 | queryMemory | **有** |
| S09 | TTL 到期复核 | 记忆 | ttl-queue / sweep | **有** |
| S10 | 会话 MemoryLink 三性 | 记忆 + 交互 | seal / replay / links | **有**（Agent 深链已接） |
| S11 | 知识注入（RAG/Wiki/LSP） | 记忆·知识 | rag · wiki · lsp | **有** |
| S12 | 策略包 / 资产启停 | 管控 + Space | SpacePolicy · registry assets · Workbench | **有**（v5） |
| S13 | 多签评分 / 申诉 | 管控 | reviews · scores · appeal | **有** |
| S14 | CI 失败诊断 → 反馈闭环 | 运维 | ci · feedback · improve | **有** |
| S15 | 发布门禁 / 合规导出 | 运维 | releases · compliance · audit | **有** |
| S16 | 空间成员与密钥 | 平台 | spaces · secrets · auth | **有** |
| S17 | Hooks 扩展（cite-guard 等） | 管控 | ash.hooks.v1 | **有**（Pre/PostToolUse；cite-guard 可用 Post deny） |
| S18 | 声明式 execpolicy / 沙箱能力位 | 管控 | policy · Doctor | **有**（schema/合并 + 沙箱地板 + Doctor） |

---

## 3. 多级模块 × 接口 × 状态 × 工时 × 优先级

图例：**优先级** P0 > P1 > P2 > P3（P3=观察/可选）。接口路径均相对 `/api/v1`（另有 `/readyz`）。

### 3.1 Agent · 薄交互

| L2 | L3 能力 | 关键接口 / 事件 | 状态 | 人时 | 优先级 | 说明 |
|----|---------|-----------------|------|------|--------|------|
| 会话壳 | 会话 CRUD / SSE / 意图 | `agents/sessions*` · `…/actions` · `…/stream` · `…/events` | 有 | — | — | 生产已落地；GV01/02 薄客户端 |
| 会话壳 | 工作区分组 | `agent-workspaces*` | 有 | — | — | 改名/关闭/搜索已有 |
| 会话壳 | 编程 / 通用模式 | session 元数据或 provider 配置 | 有 | — | — | AgentModeSwitch 已一等 |
| 会话壳 | ASH 空态品牌背景 | —（静态资源） | 有 | — | — | EW03 空态英雄 |
| Composer | 发送 / 暂停合一 · Markdown | session actions + stream | 有 | — | — | DSH 对标主路径完成 |
| Composer | 权限席位 ask/auto/full | `PATCH …/sessions` `permissionMode` | 有 | — | — | 中文席位 EW02 |
| Composer | 模型席位 / Auto 路由 | `agents/models` · model-router | 有 | — | — | 席位 + Provider 健康投影；Auto 路由 API 已有 |
| Composer | + 菜单（@ / Skills / MCP / Tools） | skills · mcp/tools · tools/risk-catalog | 有 | — | — | 设置面板已一等 |
| Details | 节点详情 · Trajectory 深链 | events · interactions | 有 | — | — | Chat 薄；厚面在评审 |
| Quest | 看板 / 门禁 / diff | `quest/*` · `runs/*/diff*` | 有 | — | — | `/quest` 默认入口 |
| 任务板 | 原型「任务板」入口 | — | 有 | — | — | 设置菜单「任务板」→ `/quest`；不新开产品面 |
| 语音输入 | 原型占位 | — | 待开发 | — | P3 | 明确低优 / 可不做 |

**Agent 待投入小计：~72 人时（不含不做项）**

### 3.2 记忆 · 双核

| L2 | L3 能力 | 关键接口 | 状态 | 人时 | 优先级 | 说明 |
|----|---------|----------|------|------|--------|------|
| 记忆体 | 候选列表 / 薄通过拒绝 | `memory/candidates*` · `…/review` | 有 | — | — | MemoryPage；「厚审」→ Reviews |
| 记忆体 | 新建候选 | `POST /memory/candidates` | 有 | — | — | |
| 记忆体 | 分层 L0/L1/L2 浏览 | candidates + records | 有 | — | — | 分层滤镜 + 计数 |
| 记忆体 | 场景/Skill/Tools/项目视角 | query 扩展字段 | 有 | — | — | EW24 标签/scopeRepo 分组 |
| 记忆体 | 检索已批准 | `POST /memory/query` | 有 | — | — | |
| 记忆体 | TTL 队列 / sweep | `memory/ttl-queue` · `ttl-sweep` | 有 | — | — | |
| MemoryLink | 列表 / seal / replay | `interactions/…/memory-links` · seal · replay | 有 | — | — | |
| MemoryLink | Agent → Link 深链 | 前端路由 | 有 | — | — | Chat「记忆」→ `/memory?tab=links&runId=` |
| 资产 | MemoryAsset 登记 / 启停 | `memory/assets*` | 有 | — | — | 厚启停在评审/Space |
| 知识 | RAG profile / 重建索引 | `rag/profile` · `rag/symbols/rebuild` | 有 | — | — | |
| 知识 | Wiki 投影 | `wiki/pages*` | 有 | — | — | `/knowledge` 深链存在、顶栏无独立柱 |
| 知识 | LSP 探针 | `rag/lsp/*` | 有 | — | — | 依赖 gopls 环境 |
| 知识 | 并入记忆 Tab（IA） | — | 有 | 8 | P2 | 三主题已落地；与原型一致化文案 |

**记忆待投入小计：~136 人时**

### 3.3 管控&评审

| L2 | L3 能力 | 关键接口 / 协议 | 状态 | 人时 | 优先级 | 说明 |
|----|---------|----------------|------|------|--------|------|
| 评审台 | 队列 / 指派 / 决定 | `reviews/*` | 有 | — | — | |
| 评分 | rubric / appeal / evaluation | `scores/*` · `spaces/…/evaluation` | 有 | — | — | v5 Workbench |
| 策略 | SpacePolicy / 规则包 | `spaces/…/policy` · `…/rules*` | 有 | — | — | |
| 资产 | AgentAsset Registry | `agents/assets*` | 有 | — | — | |
| 交互观测 | by-run / threads / compare | `interactions/*` | 有 | — | — | |
| Trajectory | 过滤投影台（P01） | events + interactions | 有 | — | — | EW32 过滤器 |
| 会话树 | Fork / Compare（P05） | threads + 新元数据 | 有 | — | — | EW21–EW22 Fork/Compare |
| Compact | /compact 语义 + 回放（P06） | compaction 事件 | 有 | — | — | slash 压缩 + Details 摘要；Run spill 仍自动 |
| Steer/Queue | 运行中打断 vs 排队（P07） | session actions 扩展 | 有 | — | — | EW16–EW17 MVP |
| 审批预设 | once / 会话 / 写入策略（P12） | approvals · SpacePolicy | 有 | — | — | EW04 |
| Hooks | ash.hooks.v1（P13） | 新协议 + 审计 | 有 | — | — | EW11–EW13 + EW41 MCP |
| ExecPolicy | 声明式沙箱能力位（P11） | policy + Doctor | 有 | — | — | EW14–EW15 |
| 子代理谱系 | 图 / 中断 / 回主（P16） | `runs/…/sub-runs` · threads | 有 | — | — | Trajectory 派生 + 回主聚焦；中断仍用既有 cancel |
| MCP 执行 | 工具真实执行收尾（P15） | `mcp/tools*` + ToolBus | 有 | — | — | PreToolUse deny/ask 与 Run 同源 |
| Harness / Improve | 配置演进 | `harness/*` · `improve/*` | 有 | — | — | Automation 内嵌 |

**管控待投入小计：~640 人时**（条目加总约 792，波次并行与复用折减后）

### 3.4 平台 / 设置 / 运维

| L2 | L3 能力 | 关键接口 | 状态 | 人时 | 优先级 | 说明 |
|----|---------|----------|------|------|--------|------|
| 认证 | login / me / sessions / device | `auth/*` | 有 | — | — | |
| 空间 | orgs / spaces / members / scopes | `orgs*` · `spaces*` | 有 | — | — | |
| 密钥 | secrets CRUD / rotate | `secrets*` | 有 | — | — | |
| 设置 IA | 顶栏「设置」聚合运维入口 | — | 有 | — | — | EW81 分组菜单 |
| 运行 | runs 全生命周期 | `runs*` | 有 | — | — | |
| 自动化 | waker · skills · improve | `waker*` · skills · improve | 有 | — | — | |
| CI / 反馈 | diagnoses · feedback | `ci*` · `feedback*` | 有 | — | — | |
| 发布 / 合规 | releases · compliance · audit | 对应前缀 | 有 | — | — | |
| 规模 / 诊断 | scale · doctor | `scale/readiness` · `doctor/*` | 有 | — | — | |
| 可观测 | alerts · trace · metrics | observability · metrics | 有 | — | — | Reviews 内嵌 |
| LLM 实机 | `ASH_LLM_BASE_URL` | — | 待改进 | 8 | P1 | 环境依赖，非功能缺口 |
| OpenAPI 漂移 | legacy `/v1/*` · 缺 `GET /spaces` 等 | openapi yaml | 有 | — | — | EW01+EW101：契约仅 `/api/v1/*` |

**平台待投入小计：~40 人时**

---

## 4. 后端接口分组速查

| 域 | ~Ops | 生产 Console 主消费方 | 相对原型 |
|----|------|----------------------|----------|
| Agents + workspaces | ~17 | Quest / AgentChatShell | 壳已齐；W0–W3 吸收语义已落地 |
| Interactions | ~8 | Agent / Reviews / MemoryLink | 有（Fork/Compare · 谱系 · MemoryLink 深链） |
| Memory + registry | ~15 | MemoryPage / Space / Reviews | 有（四视角分组 + 厚审深链） |
| RAG / Wiki / LSP | ~10 | KnowledgePanel | 有 |
| Runs + Quest | ~30 | Runs / Quest | 有（设置「任务板」→ Quest） |
| Reviews / Harness / Scores | ~21 | ReviewsPage | 有（v5） |
| Spaces / Policy / Auth | ~29 | Space / Login | 有 |
| Platform ops（CI/审计/Doctor/Waker…） | ~40+ | 设置菜单各页 | 有 |
| Hooks / Steer / Compact / ExecPolicy | 中 | Agent Chat / Run | **有**（Pre/PostToolUse · Compact · Steer/Queue · ExecPolicy） |

遗留：契约草稿已去掉无 handler 的 legacy `/v1/*`（EW101）；`openapi-check` 要求计数为 0。

---

## 5. 建议交付顺序（可执行）

1. **W0（~3 周 · 1 人）**：审批预设 + permissionMode 对齐 · MCP exec · Chat 空态/模式 · OpenAPI 卫生  
   → 任务板 [`sprint-ew-w0-harden.md`](sprint-ew-w0-harden.md) · 实现计划 [`docs/superpowers/plans/2026-09-18-w0-harden.md`](../../docs/superpowers/plans/2026-09-18-w0-harden.md)  
2. **W1（~2 月 · 1.5 人）**：Hooks MVP · execpolicy 骨架 · Steer/Queue 协议与 Chat 投影  
   → 任务板 [`sprint-ew-w1-v60-absorb.md`](sprint-ew-w1-v60-absorb.md) · **✅**  
3. **W2（~1.5 月 · 1.5 人）**：会话树 fork/compare · Compact UX · 记忆四视角字段回填  
   → 任务板 [`sprint-ew-w2-v61-absorb.md`](sprint-ew-w2-v61-absorb.md) · **✅**（EW21–EW25）  
4. **W3（~1 月 · 1 人）**：子代理谱系台 · Trajectory 过滤器工作台 · Provider/Doctor 探针  
   → 任务板 [`sprint-ew-w3-v62-absorb.md`](sprint-ew-w3-v62-absorb.md) · **✅**（EW31–EW34）  
5. **P15 闭环**：MCP HTTP execute 走同一 PreToolUse（deny 不可用 token 绕过）  
   → 任务板 [`sprint-ew-w4-p15-loop.md`](sprint-ew-w4-p15-loop.md) · **✅**（EW41–EW42）  
6. **P16 谱系台**：Trajectory 派生子 Run + 回主聚焦  
   → 任务板 [`sprint-ew-w5-p16-lineage.md`](sprint-ew-w5-p16-lineage.md) · **✅**（EW51–EW52）  
7. **MemoryLink 深链**：Agent Chat「记忆」→ 记忆页关联页签  
   → 任务板 [`sprint-ew-w6-memorylink-deeplink.md`](sprint-ew-w6-memorylink-deeplink.md) · **✅**（EW61–EW62）  
8. **P06 Compact**：`/compact` 压缩 transcript + Details 回放摘要  
   → 任务板 [`sprint-ew-w7-compact.md`](sprint-ew-w7-compact.md) · **✅**（EW71–EW72）  
9. **设置 IA**：顶栏「设置」聚合运维入口  
   → 任务板 [`sprint-ew-w8-settings-ia.md`](sprint-ew-w8-settings-ia.md) · **✅**（EW81–EW82）  
10. **模型健康位**：Composer 投影 `model-router/providers`  
   → 任务板 [`sprint-ew-w9-model-health.md`](sprint-ew-w9-model-health.md) · **✅**（EW91–EW92）  
11. **OpenAPI 卫生**：契约草稿去掉无 handler 的 `/v1/*`  
   → 任务板 [`sprint-ew-w10-openapi-hygiene.md`](sprint-ew-w10-openapi-hygiene.md) · **✅**（EW101–EW102）  
12. **PostToolUse**：工具成功后声明式钩子  
   → 任务板 [`sprint-ew-w11-post-tool-use.md`](sprint-ew-w11-post-tool-use.md) · **✅**（EW111–EW112）  
13. **记忆厚审深链 + 任务板入口**：候选 → Reviews；设置 → Quest  
   → 任务板 [`sprint-ew-w12-memory-thick-review.md`](sprint-ew-w12-memory-thick-review.md) · **✅**（EW121–EW122）  
14. **v6.0 冻结**：吸收水位签字门禁  
   → [`v6.0-release-scope.md`](v6.0-release-scope.md) · [`sprint-ew130-v60-signoff.md`](sprint-ew130-v60-signoff.md) · **✅**（EW130）；`make v6-signoff`

明确 **不做**（v6.0 Out）：Cordis、完整 DSH slash 一比一、Hermes 全 IM、YOLO 默认、语音输入占位；**v6.3+**（RPC / Cron / Backend）另开代际。

---

## 6. 修订

| 日期 | 说明 |
|------|------|
| 2026-09-20 | **v6.0 已冻结**（EW130）：W0–W12 吸收收口；`make v6-signoff`；下一代 v6.3 |
| 2026-09-18 | 初版：原型 × Console × OpenAPI × 四象限吸收优先级 |
| 2026-09-18 | W0 → Sprint 板 EW01–EW05 + 实现计划 |
| 2026-09-19 | W0 ✅；W1 → Sprint 板 EW11–EW18 + 实现计划 |
| 2026-09-19 | W1 ✅（EW18 签字）：Hooks / ExecPolicy / Steer-Queue MVP 已落地；上表三行 待开发→部分交付 |
| 2026-09-19 | W2 开工：EW21 会话线程 Fork |
| 2026-09-19 | W3 ✅（EW31–EW34）：谱系、时间线过滤、Doctor `M4-MDL-01` |
| 2026-09-19 | P15 ✅（EW41–EW42）：MCP execute 走 PreToolUse |
| 2026-09-19 | P16 ✅（EW51–EW52）：Trajectory 派生子 Run |
| 2026-09-19 | MemoryLink 深链 ✅（EW61–EW62） |
| 2026-09-19 | P06 ✅（EW71–EW72）：`/compact` + Details 摘要 |
| 2026-09-19 | 设置 IA ✅（EW81–EW82）；同步勾销已交付清单行 |
| 2026-09-19 | 模型健康位 ✅（EW91–EW92） |
| 2026-09-19 | OpenAPI 卫生 ✅（EW101–EW102）：去掉 legacy `/v1/*` |
| 2026-09-19 | PostToolUse ✅（EW111–EW112） |
| 2026-09-19 | 记忆厚审深链 + 任务板 ✅（EW121–EW122）；勾销 S01–S03/S07 与 §4 陈旧表述 |
