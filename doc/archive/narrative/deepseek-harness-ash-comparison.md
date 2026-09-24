# DeepSeek Harness 与 ASH 功能比对分析

> 状态：调研稿（2026-08-27）  
> 归属：[`plan/`](README.md)  
> 对照基线：ASH `v0.1.0-mvp`（PRD/ARCH/MVP 范围）  
> 外部项目：本地扫描 `C:\Go_Work\src\deepseek-harness`（`@deepseek-ai/dsh` v0.1.0-rc.7）  
> 关联：[DeepSeek Harness README](https://github.com/deepseek-ai/deepseek-harness) · [`qoder-ash-comparison.md`](qoder-ash-comparison.md) · [`agentic-roadmap-to-qoder.md`](agentic-roadmap-to-qoder.md)

## 1. 目的

梳理 [DeepSeek Harness](https://github.com/deepseek-ai/deepseek-harness)（`dsh`）与当前 ASH 实现做功能级比对，明确**重叠点**、**差异点**与**可借鉴方向**，供产品/架构决策参考。本文不构成对 DSH 的完整功能承诺或 SLA 背书，以本地仓库扫描与 ASH 现状为准。

---

## 2. 一句话定位

| 项目 | 定位 |
|------|------|
| **DeepSeek Harness (`dsh`)** | 基于 **Cordis** 的**插件化 Agent 运行时**：一切皆插件，可替换 agent loop、工具、持久化、沙箱；面向开发者本地/自动化使用 |
| **ASH** | **交付编排平台**：Scenario DSL + Run 状态机 + Artifacts 四件套 + Doctor/发布门禁；面向可审计的交付闭环，不是 IDE |

两者都在做「Agent 跑真实任务」，但 **DSH 卖运行时可组合性**，**ASH 卖交付治理与证据链**。

---

## 3. 技术栈对照

| 维度 | DeepSeek Harness | ASH |
|------|------------------|-----|
| 语言 | TypeScript 6 / Node 22+ | Go 1.26 + React 18 / TS |
| 规模 | pnpm monorepo，~40 能力组、200+ `@deepseek-ai/dsh-*` 包 | Go 单体 Worker + `internal/*` 包边界 |
| 前端 | Vite + React，Host/Client 双 TS 编译面 + Typert RPC | Vite + React，TanStack Router/Query |
| 持久化 | JSONL / SQLite session log（事件溯源） | SQLite / Postgres + GORM + SQL 迁移 rev 20 |
| 插件 | Cordis `cordis.patch.yml` 分层组合 | gRPC Plugin ABI + HMAC 签名 |
| 分发 | `npx @deepseek-ai/dsh web`（默认 :3080） | Worker `:8080` + CLI + `/ui` |
| SDK | TS JSON-RPC stdio、Python SDK、ACP | OpenAPI `/api/v1`、CLI |
| 沙箱 | Landlock / bwrap / Seatbelt / E2B POC | Policy + tool_risk + human 门禁（无容器沙箱） |
| 测试文化 | 包级 100% 覆盖率目标、50+ verify gates、snapshot replay | Doctor TR0–TR3、mvp-signoff 发布门禁 |

---

## 4. 架构哲学（最大差异）

### 4.1 DSH：事件溯源 + 全插件

- **Session log 是唯一真相**：`model-visible ⟺ logged`，历史由 `deriveMessages()` 从日志推导，不重复存一份
- **无特权核心**：agent loop、工具注册、LLM 适配器都是 Cordis 插件，用 `cordis.patch.yml` 热补丁
- **能力接缝（Capability seams）**：`ctx.fs`、`ctx.tools`、`ctx.sandbox` 等 Service/Provider 可整体替换
- **Profile → Bundle → Plugin**：`web` / `headless` 等 profile 叠加 `dsh-base` + 可选 bundle

核心包（`packages/core/`）：

| 包 | ctx 键 | 职责 |
|----|--------|------|
| session | `ctx.sessions` | 追加式 `SessionEvent` 日志 |
| system-prompt | `ctx.systemPrompt` | Prompt 段与 tool schema 组装 |
| tools | `ctx.tools` | 工具注册与 guarded 执行管道 |
| agent | `ctx.agents` | Agent 接口与 `agent/*` 事件 |
| agent-loop | `ctx.agentLoop` | 默认驱动（`ReactLoopAgent`） |
| llm | `ctx.llm` | 流式适配器接缝 |

### 4.2 ASH：场景契约 + 交付门禁

- **Scenario DSL 是真相源**：三场景（`feature_delivery` / `hotfix` / `security_patch`）版本化、Doctor 可测
- **Run 是交付单元**：状态机、SSE 事件、`waiting_approval`（citation / tool_risk / human）
- **Artifacts 四件套**：spec / implementation / release_notes / rollback_plan + canonical digest
- **Doctor + mvp-signoff**：43 用例 + 云 RDS / 签字 / rollback-drill 等发布文化

详见 [`ARCH-架构与技术选型.md`](../design/ARCH-架构与技术选型.md)、[`PLAN-进度与里程碑.md`](PLAN-进度与里程碑.md)。

---

## 5. 功能域逐项对照

### 5.1 高度重叠

| 能力 | DSH | ASH | 差异侧重 |
|------|-----|-----|----------|
| Agent 多步执行 | `ReactLoopAgent` + turn/step | Runs + Agent/ToolBus | DSH 偏通用 loop；ASH 偏场景步骤 |
| 工具调用 | read/write/bash/grep/MCP… | ToolBus 内置 + MCP | DSH 工具面更广（LSP、Code Mode、Ralph） |
| 人工审批 | `ctx.approval` + permission preset | `waiting_approval` | ASH 与 DSL 门禁深度绑定 |
| 子 Agent | subagent / fork / ACP / Claude Code / Codex | 场景内固定角色（Sub-run 在路线图） | DSH 已生产化多后端 |
| 会话/任务持久化 | JSONL/SQLite session | Run + events 表 + checkpoint | DSH 事件溯源；ASH 业务 Run 模型 |
| Web 控制台 | 插件化 React client | 12 页控制台 | DSH UI 可插拔；ASH 偏运维/合规页 |
| CLI | `dsh --profile headless "task"` | `ash run` / `doctor` | 均支持无头执行 |
| 多模型 | `ctx.llm` 适配器 | Model Router | 均可换 Provider |
| 可观测 | OTLP（默认关）、session telemetry | Prometheus + KPI + derive replay | ASH 有 KPI-11、发布 KPI 对账 |
| MCP | 一等公民 `mcp__server__tool` | ToolBus MCP 桥接 | DSH 集成更深 |
| Skills | 文件系统 skill loader | 路线图（Automation 页扩展） | DSH 已落地 |

### 5.2 DSH 有、ASH 弱或无

| 能力 | DSH | ASH |
|------|-----|-----|
| **插件组合/HMR** | Profile + bundle + patch 分层 | 固定 Worker 二进制 + 可选 gRPC 插件 |
| **Plan mode** | `exit_plan_mode`、稳定 tool catalog | Spec 在场景内，无独立 Plan 模式 |
| **Code Mode** | `run_code` 生成程序调工具 | 无 |
| **Ralph 工作流** | 多轮 fresh-agent + workspace 记忆 | improve 提案 + replay（不同形态） |
| **后台 Job** | `job_list` / bash PTY / subagent jobs | 队列 TTL sweep，无交互式 PTY 作业台 |
| **终端套件** | `terminal_*` 持久 shell | 无 |
| **Web 搜索** | `web_search` / `web_fetch` | 无（RAG 限代码库 FTS） |
| **上下文压缩** | compaction + tool-result pruner | Memory TTL + feedback 衰减 |
| **沙箱** | Landlock / Seatbelt / E2B | Policy deny，无 OS 级沙箱 |
| **Hook 桥接** | Claude Code / Codex `hooks.json` | 无 |
| **自举扩展** | `cordis_*` 动态挂载插件 | 无 |
| **ACP / Python SDK** | 官方多语言自动化面 | OpenAPI + Go CLI |
| **LSP 工具** | 可选 `lsp` provider | 无 |
| **内置 RAG** | 无（靠 MCP memory 示例） | FTS SQLite/Postgres 一等能力 |

### 5.3 ASH 有、DSH 弱或无

| 能力 | ASH | DSH |
|------|-----|-----|
| **交付场景 DSL** | YAML 三场景 + JSON Schema + 版本 | 无等价「交付契约」层 |
| **Artifacts 四件套 + digest** | canonical JSON、路径策略 | 会话产物无统一交付包模型 |
| **Doctor 质量门禁** | TR0–TR3 / M2 / M3，43 用例 | Vitest + gates，偏工程 CI 非产品 Doctor |
| **Memory 治理链** | L0–L2、评审 merge、TTL、confidence | 无内置 Memory 子系统；靠 MCP memory 示例 |
| **Citation 门禁** | 改代码须有依据 | 无同等流程门禁 |
| **CI 诊断** | GitHub fixture + 规则引擎 + adopt | 无 |
| **合规/审计** | TR2 export、脱敏、data-policy、保留期 | telemetry 可选，无合规产品面 |
| **多租户 RLS** | Postgres RLS 41 策略 | workspace 分组，非企业 RLS 模型 |
| **发布治理** | Releases API、灰度记录、rollback-drill | 无 MVP 发布 runbook 文化 |
| **组织样板** | org-templates + Space 页 | settings/credentials 个人向 |
| **Postgres 生产切换** | migrate e2e、cloud-acceptance | SQLite session 为主 |

---

## 6. Agentic 闭环对照

| 阶段 | DSH | ASH |
|------|-----|-----|
| 委派 | headless 一句话 / Web UI / ACP | 场景 YAML + inputs（Goal API 在路线图） |
| 规划 | Plan mode + goal 包 | DSL Spec + 步骤计划（可见性待加强） |
| 执行 | 全工具链 + subagent + sandbox | ToolBus + 场景步骤 |
| 门禁 | approval + permission preset | citation / tool_risk / human |
| 验证 | 依赖 agent 自验 + 测试文化 | Doctor + verify 步骤（路线图） |
| 交付物 | session 导出、workspace | **Artifacts 四件套** |
| 记忆 | compaction / skills / MCP | **Memory 评审治理** |
| 迭代 | Ralph / schedule / feedback | improve + replay + KPI-11 |

DSH 的闭环在**运行时与工具广度**；ASH 的闭环在**交付契约与可签字证据**。

---

## 7. 成熟度与工程文化

| 信号 | DSH | ASH |
|------|-----|-----|
| 版本 | `0.1.0-rc.7`，明确 breaking changes | `v0.1.0-mvp` tag，范围冻结 |
| 状态标签 | Developer preview | MVP 功能完成，待云签字发布 |
| 文档 | 220+ md、双语、生成 catalog | design/plan/progress/checklists 体系 |
| CI | 50+ gates、包覆盖率、Wine Windows | mvp-signoff / release-window-gate |
| 开源 | DeepSeek 官方 MIT | 自有仓库 |

DSH **工程成熟度更高**（测试与插件生态）；ASH **发布与合规成熟度更高**（Doctor、清单、证据链）。

---

## 8. 与 Qoder 三角关系

| | Qoder | DeepSeek Harness | ASH |
|--|-------|------------------|-----|
| 主战场 | IDE + Quest 产品 | Agent **运行时/框架** | **交付平台** |
| 差异化 | 补全、Repo Wiki、企业版 | Cordis 可组合、工具全、沙箱 | DSL、Doctor、Artifacts、RLS |
| 与 ASH 关系 | 终端产品竞品参考 | **技术架构参考**（插件、session、subagent） | 不复制 IDE，强化交付 Agentic |

DSH 比 Qoder 更接近 ASH 的「编排」问题，但 DSH 仍缺**交付契约与发布治理**；Qoder 比 DSH 更接近**产品化 Quest/IDE**。详见 [`qoder-ash-comparison.md`](qoder-ash-comparison.md)。

---

## 9. 对 ASH 的可借鉴点

| 优先级 | 借鉴项 | 对应 ASH 动作 |
|--------|--------|--------------|
| **P1** | Session 事件模型 +「可见即已记录」 | Run 事件协议与 derive 对齐，减少双写 |
| **P1** | Subagent 多后端（in-process / ACP / SDK） | 路线图波 3 Sub-run |
| **P1** | Plan mode（稳定 schema + 行为策略） | Sprint DA Goal→Plan |
| **P2** | Compaction / spill 大输出 | Run 上下文预算 + 工具结果落盘 |
| **P2** | Skills 文件 loader | Automation 页 + 场景声明 skill 集 |
| **P2** | MCP memory 可选集成 | 不内置向量库，接 MCP 即可 |
| **P3** | Cordis 式 patch 组合 | 过重；保持 Scenario DSL + 轻量 org-template |
| **P3** | Code Mode / LSP / PTY 终端 | 非 ASH 主战场 |

**不建议对标**：DSH 的全插件运行时、Landlock 沙箱、IDE 级工具全集——会稀释 ASH「交付治理」定位。迭代路线见 [`agentic-roadmap-to-qoder.md`](agentic-roadmap-to-qoder.md)。

---

## 10. 战略结论

```
         工具广度 / 运行时灵活性
                    ↑
         DeepSeek Harness
                    |
    Qoder ←---------+---------→ （缺位：IDE 补全）
   (产品化 Quest)   |
                    ↓
              ASH（交付契约 + 门禁 + 证据）
```

- **DSH** 适合作为 **Agent 运行时与插件架构** 的参考实现（session log、subagent、approval、sandbox、MCP）。
- **ASH** 应继续强化 **Goal→Run→Artifacts→Doctor→签字** 链，而不是重写为 TypeScript Cordis monorepo。
- 二者可共存：ASH Worker 将来可把 **ExecGo/ACP/SDK 子进程** 当作一种 subagent provider（类似 DSH 的 `dsh-sdk` children），上层仍用 Scenario DSL 与四件套约束。

---

## 11. 关键文件索引

### DeepSeek Harness（本地扫描路径）

| 用途 | 路径 |
|------|------|
| README | `deepseek-harness/README.md` |
| 架构 | `deepseek-harness/docs/architecture.md` |
| 工具目录 | `deepseek-harness/docs/tool-catalog.md` |
| 默认插件树 | `deepseek-harness/packages/bundle/base/cordis.patch.yml` |
| CLI 入口 | `deepseek-harness/apps/cli/src/bin.ts` |
| Agent loop | `deepseek-harness/packages/core/agent-loop/` |
| 测试策略 | `deepseek-harness/docs/testing.md` |
| CI gates | `deepseek-harness/scripts/run-gates.ts` |

### ASH

| 用途 | 路径 |
|------|------|
| 架构 | [`../design/ARCH-架构与技术选型.md`](../design/ARCH-架构与技术选型.md) |
| 进度 | [`PLAN-进度与里程碑.md`](PLAN-进度与里程碑.md) |
| Qoder 比对 | [`qoder-ash-comparison.md`](qoder-ash-comparison.md) |
| Agentic 路线 | [`agentic-roadmap-to-qoder.md`](agentic-roadmap-to-qoder.md) |

---

## 12. 修订记录

| 日期 | 说明 |
|------|------|
| 2026-08-27 | 初稿：本地扫描 deepseek-harness v0.1.0-rc.7 与 ASH v0.1.0-mvp 比对 |
