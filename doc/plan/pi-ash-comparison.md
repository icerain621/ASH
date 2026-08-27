# Pi 与 ASH 功能比对分析

> 状态：调研稿（2026-08-27）  
> 归属：[`plan/`](README.md)  
> 对照基线：ASH `v0.1.0-mvp`（PRD/ARCH/MVP 范围）  
> 外部项目：本地扫描 `C:\Go_Work\src\pi`（`@earendil-works/pi-coding-agent` v0.84.3）  
> 关联：[pi.dev](https://pi.dev) · [`qoder-ash-comparison.md`](qoder-ash-comparison.md) · [`deepseek-harness-ash-comparison.md`](deepseek-harness-ash-comparison.md) · [`agentic-roadmap-to-qoder.md`](agentic-roadmap-to-qoder.md)

## 1. 目的

梳理 [Pi](https://pi.dev)（`pi` CLI / `pi-monorepo`）与当前 ASH 实现做功能级比对，明确**重叠点**、**差异点**与**可借鉴方向**，供产品/架构决策参考。本文不构成对 Pi 的完整功能承诺或 SLA 背书，以本地仓库扫描与 ASH 现状为准。

---

## 2. 一句话定位

| 项目 | 定位 |
|------|------|
| **Pi (`@earendil-works/pi-coding-agent`)** | **极简、可自扩展的终端 Coding Agent Harness**：核心只做 agent loop + 少量内置工具；MCP、子 Agent、Plan mode、权限弹窗等**刻意不内置**，靠 TypeScript 扩展 / Skills / Pi packages 补齐 |
| **ASH** | **交付编排平台**：Scenario DSL + Run 状态机 + Artifacts 四件套 + Doctor/发布门禁；面向可审计的交付闭环，不是终端 Coding Agent |

Pi 与 ASH 的交集主要在「Agent 调工具改代码」，但 **Pi 卖终端体验与可扩展运行时**，**ASH 卖场景契约与发布证据链**。

---

## 3. 技术栈对照

| 维度 | Pi | ASH |
|------|-----|-----|
| 语言 | TypeScript 5.9 / Node ≥22.19 | Go 1.26 + React 18 / TS |
| 规模 | npm workspaces，~10 发布包（`pi-coding-agent` 等） | Go Worker + `internal/*` + 12 页控制台 |
| 产品形态 | **终端 TUI**（`pi-tui`）为主 | **Web 控制台** + Worker HTTP API |
| 持久化 | JSONL **分支会话树**（`id`/`parentId`）；可选 SQLite FTS | SQLite / Postgres + Run/events 表 |
| 插件/扩展 | TypeScript **Extensions** + **Pi packages**（npm/git） | gRPC Plugin ABI + HMAC 签名 |
| LLM | **`pi-ai`**：30+ Provider 统一流式 API | Model Router + ExecGo 桥接 |
| 分发 | `npm i -g @earendil-works/pi-coding-agent`、`pi.dev/install.sh`、Bun 独立二进制 | Worker `:8080` + `ash` CLI |
| 集成面 | `--mode rpc` JSONL、SDK `createAgentSession()`、实验性 CBOR `pi-protocol` | OpenAPI `/api/v1`、SSE |
| 沙箱 | **无内置权限系统**；文档化容器/Gondolin 模式 | Policy + tool_risk + human 门禁 |
| 测试 | Biome + vitest + `test.sh`（剥离 API key）、evals 包 | Doctor TR0–TR3、mvp-signoff |

---

## 4. 架构哲学（最大差异）

### 4.1 Pi：极简核心 + 扩展优先

**分层包结构：**

```
pi-coding-agent (CLI/SDK)
    → pi-agent-core (Agent / agentLoop / harness)
    → pi-ai (多 Provider LLM)
    → pi-tui (终端 UI)
```

**设计原则**（`packages/coding-agent/README.md` Philosophy）：

- **No MCP in core** — 用 Skills + README 式 CLI 工具，或扩展自行加 MCP
- **No built-in sub-agents / plan mode / permission popups / todos** — 扩展或 Pi package 领域
- **默认工具极少**：`read` / `write` / `edit` / `bash`（+ `grep` / `find` / `ls` / `powershell`）
- **会话 = JSONL 树**：`/tree`、`/fork`、`/clone` 分支，非线性历史
- **Steering / Follow-up**：运行中打断 vs 完成后排队，可配置投递语义
- **Compaction**：上下文溢出时 lossy 摘要（`/compact`）

**运行模式：**

| 模式 | 入口 | 用途 |
|------|------|------|
| Interactive TUI | 默认 | 全功能终端 Agent |
| Print | `-p` | 一次性文本输出 |
| JSON events | `--mode json` | 机器可读事件流 |
| RPC | `--mode rpc` | LF 分隔 JSONL 进程集成 |
| SDK | `createAgentSession()` | 嵌入 Node 应用 |

### 4.2 ASH：场景契约 + 交付门禁

- **Scenario DSL** 为真相源（三场景版本化、Doctor 可测）
- **Run** 为交付单元（状态机、SSE、`waiting_approval`）
- **Artifacts 四件套** + canonical digest
- **Doctor + mvp-signoff** 发布文化

详见 [`ARCH-架构与技术选型.md`](../design/ARCH-架构与技术选型.md)。

---

## 5. 功能域逐项对照

### 5.1 高度重叠

| 能力 | Pi | ASH | 差异侧重 |
|------|-----|-----|----------|
| Coding Agent 多步执行 | `agentLoop` + 内置工具 | Runs + ToolBus + Agent | Pi 通用 loop；ASH 场景步骤 |
| 读/写/编辑/执行命令 | read/write/edit/bash | ToolBus 内置 | Pi 终端 UX 更深（fuzzy edit、PTY） |
| 人工介入 | steering / 扩展可实现确认 | `waiting_approval` | ASH 与 DSL 门禁绑定 |
| 会话持久化 | JSONL session tree | Run + events + checkpoint | Pi 分支会话；ASH 业务 Run |
| 多模型 | `pi-ai` 30+ Provider | Model Router | Pi Provider 面更广、OAuth 登录 |
| CLI | `pi` 全功能 TUI | `ash run` / `doctor` | Pi 即主产品；ASH CLI 偏运维 |
| 可嵌入 | SDK + RPC | OpenAPI + Worker | 均可被外部系统驱动 |
| Skills | Agent Skills 标准 `SKILL.md` | 路线图 Automation 扩展 | Pi 已落地 |
| 上下文文件 | `AGENTS.md` / `CLAUDE.md` 向上遍历 | RAG FTS + citation | ASH 有检索与引用门禁 |

### 5.2 Pi 有、ASH 弱或无

| 能力 | Pi | ASH |
|------|-----|-----|
| **终端 TUI 产品** | `pi-tui` 差分渲染、主题、快捷键 | 无（Web 控制台） |
| **多 Provider 统一层** | `pi-ai` 流式 + handoff + OAuth `/login` | 有限 Provider 适配 |
| **会话分支** | JSONL 树 `/fork` `/tree` | Replay 新 Run，非会话树 |
| **Compaction** | 内置上下文压缩 | Memory TTL，无对话 compaction |
| **扩展生态** | ExtensionAPI + Pi packages（npm/git） | gRPC 插件，无 TS 扩展市场 |
| **RPC/JSON 模式** | 一等公民 `--mode rpc/json` | SSE + REST |
| **实验性远程协议** | `pi-protocol` CBOR + `pi-server`/`pi-client` | 无对等协议层 |
| **Supply-chain 硬化** | shrinkwrap、pin deps、ignore-scripts 安装 | 常规 Go/npm 依赖 |
| **Evals / 会话公开** | `pi-evals`、`pi-share-hf` | Doctor 用例，非行为 eval 数据集 |
| **容器化指南** | Gondolin / Docker / OpenShell 文档 | 无 OS 级沙箱文档 |
| **Thinking levels** | off/minimal/low/…/max，模型轮换 | 无同等终端 UX |

### 5.3 ASH 有、Pi 弱或无

| 能力 | ASH | Pi |
|------|-----|-----|
| **交付场景 DSL** | YAML 三场景 + JSON Schema | 无交付契约层 |
| **Artifacts 四件套** | spec/impl/release_notes/rollback + digest | 会话导出 HTML/JSONL，无交付包模型 |
| **Doctor 质量门禁** | TR0–TR3，43 用例 | vitest + check，非产品 Doctor |
| **Memory 治理链** | L0–L2、评审 merge、TTL、confidence | 无长期 Memory；靠 compaction + 文件上下文 |
| **RAG（代码库）** | FTS SQLite/Postgres | 可选 session FTS（SQLite backend），非代码库 RAG |
| **Citation 门禁** | 改代码须有依据 | 无流程门禁 |
| **CI 诊断** | GitHub fixture + 规则引擎 | 无 |
| **合规/审计** | TR2 export、data-policy、保留期 | telemetry 契约，无合规产品面 |
| **多租户 RLS** | Postgres RLS 41 策略 | 单用户 `~/.pi/agent` |
| **发布治理** | Releases、rollback-drill、mvp-signoff | 无 |
| **Web 运维控制台** | 12 页（Runs/Memory/CI/Scale…） | 终端 only |
| **组织样板** | org-templates | 个人 settings |

---

## 6. Agentic 闭环对照

| 阶段 | Pi | ASH |
|------|-----|-----|
| 委派 | TUI 输入 / `-p` / RPC / SDK | 场景 YAML + inputs（Goal API 路线图） |
| 规划 | 扩展 / Skills（核心拒绝内置 Plan mode） | DSL Spec + 步骤 |
| 执行 | 内置工具 + 扩展工具 | ToolBus + 场景步骤 |
| 门禁 | 无内置；容器或扩展 | citation / tool_risk / human |
| 验证 | 用户/扩展自验 | Doctor + verify（路线图） |
| 交付物 | session 导出、gist 分享 | **Artifacts 四件套** |
| 记忆 | compaction、Skills、上下文文件 | **Memory 评审治理** |
| 迭代 | follow-up、fork 会话 | improve + replay + KPI-11 |

---

## 7. 与 Qoder、DeepSeek Harness 四方关系

| 维度 | Qoder | DeepSeek Harness | **Pi** | ASH |
|------|-------|------------------|--------|-----|
| 主战场 | IDE + Quest | Cordis 全插件运行时 | **终端 Coding Agent** | 交付平台 |
| 产品成熟度 | 商业产品 | rc.7 developer preview | **0.84.3 活跃发布** | MVP 待云签字 |
| MCP | 内置 | 一等公民 | **刻意不做** | ToolBus 桥接 |
| 子 Agent | Experts | subagent 多后端 | **扩展负责** | 路线图 Sub-run |
| Web UI | 丰富 | 插件化 React | **无** | 12 页控制台 |
| 扩展模型 | 插件市场 | Cordis patch | **TS Extension + Pi package** | gRPC 插件 |
| 差异化 | 补全、Wiki | 工具广度、沙箱 | **极简 + TUI + pi-ai** | DSL、Doctor、Artifacts |

**Pi 与 DSH 对比（简）**：二者都是 Agent Harness，但 Pi **更小、更终端、更反 MCP**；DSH **更重、更 Web、更 Cordis 插件化**。ASH 与二者都不同层——**编排与发布治理**。

---

## 8. 成熟度与工程文化

| 信号 | Pi | ASH |
|------|-----|-----|
| 版本 | `0.84.3`（2026-08 活跃 changelog） | `v0.1.0-mvp` |
| 状态 | 生产级终端产品 + 实验性 protocol/server | MVP 功能完成，待发布签字 |
| 文档 | pi.dev/docs、包内 docs、RFC | design/plan/checklists/evidence |
| 社区 | Discord、HF 会话数据集 | 内部交付导向 |
| 贡献 | 新贡献者 issue/PR 默认 auto-close | 常规协作 |
| 开源 | Earendil Works，npm 发布 | 自有仓库 |

Pi 在 **终端 Agent 与 LLM 适配层** 工程成熟度很高；ASH 在 **企业交付与合规门禁** 更完整。

---

## 9. 对 ASH 的可借鉴点

| 优先级 | 借鉴项 | 对应 ASH 动作 |
|--------|--------|--------------|
| **P1** | **RPC/JSON 事件模式** | Session API / 外部集成（路线图波 4） |
| **P1** | **Skills 标准**（`SKILL.md`） | Automation 页 + 场景声明 |
| **P1** | **Compaction 策略** | Run 上下文预算 + 大工具结果 spill |
| **P2** | **Steering / Follow-up 语义** | Run 控制面 cancel/resume 扩展为队列语义 |
| **P2** | **多 Provider `pi-ai` 式统一层** | Model Router 文档化 + OAuth 路径 |
| **P2** | **会话分支 / fork** | Replay 与 Run  lineage 可视化 |
| **P3** | 终端 TUI | 非 ASH 主战场（保持 Web 控制台） |
| **P3** | 反 MCP 哲学 | ASH 已选 MCP 桥接，不采纳 |

**不建议对标**：Pi 的极简核心哲学（ASH 需要场景 DSL 与门禁）、无权限默认信任模型、纯终端产品形态。

---

## 10. 战略结论

```
    IDE / Quest 产品化
           ↑
        Qoder
           |
终端极简 ← Pi ─────→ DSH（Web + Cordis 重型运行时）
           |
           ↓
    ASH（交付 DSL + Artifacts + Doctor + 签字）
```

- **Pi** 适合作为 **终端 Agent UX、pi-ai 式 Provider 层、RPC 集成、Skills/扩展生态** 的参考；不适合作为 ASH 的整体架构模板。
- **ASH** 应继续强化 **Goal→Run→Artifacts→Doctor→签字**；ExecGo/静态 Agent 可类比 Pi 的「嵌入 SDK」，但上层仍用 Scenario DSL 约束。
- **三者分工**：Pi = 开发者终端助手；DSH = 可组合 Agent 平台；ASH = 组织级交付与合规运行面。

---

## 11. 关键文件索引

### Pi（本地扫描路径）

| 用途 | 路径 |
|------|------|
| README | `pi/README.md` |
| Coding Agent | `pi/packages/coding-agent/` |
| Philosophy / No MCP | `pi/packages/coding-agent/README.md` |
| Agent loop | `pi/packages/agent/` |
| LLM 层 | `pi/packages/ai/` |
| TUI | `pi/packages/tui/` |
| 实验协议 | `pi/packages/protocol/`、`pi/packages/server/` |
| 容器化 | `pi/packages/coding-agent/docs/containerization.md` |
| CI | `pi/.github/workflows/ci.yml` |

### ASH

| 用途 | 路径 |
|------|------|
| 架构 | [`../design/ARCH-架构与技术选型.md`](../design/ARCH-架构与技术选型.md) |
| 进度 | [`PLAN-进度与里程碑.md`](PLAN-进度与里程碑.md) |
| Qoder 比对 | [`qoder-ash-comparison.md`](qoder-ash-comparison.md) |
| DSH 比对 | [`deepseek-harness-ash-comparison.md`](deepseek-harness-ash-comparison.md) |
| Agentic 路线 | [`agentic-roadmap-to-qoder.md`](agentic-roadmap-to-qoder.md) |

---

## 12. 修订记录

| 日期 | 说明 |
|------|------|
| 2026-08-27 | 初稿：本地扫描 pi v0.84.3 与 ASH v0.1.0-mvp 比对 |
