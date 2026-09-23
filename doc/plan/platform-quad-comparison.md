# 四产品平台综合分析：DSH · Pi · Codex · Hermes → ASH

> 状态：**A 分析报告（评审稿）** · 2026-09-17  
> 归属：[`plan/`](README.md)  
> 方法：**模式抽取 + 定位过滤 + 附录矩阵 + 全功能点 + 使用场景**  
> 下游：**B** → v6 演进规格（本报告签字后再写）  
> 前置：[`v5-governance-program.md`](v5-governance-program.md)（薄交互 / 厚评审 / 双核管控已大体落地）  
> 单品旧稿：[`deepseek-harness-ash-comparison.md`](deepseek-harness-ash-comparison.md) · [`pi-ash-comparison.md`](pi-ash-comparison.md) · [`qoder-ash-comparison.md`](qoder-ash-comparison.md)

---

## 0. 方法与证据源

### 0.1 方法

| 层 | 做法 | 产出 |
|----|------|------|
| **定位过滤** | 先钉死 ASH 北极星；凡把 ASH 拉成 IDE / 全插件 Harness / IM 机器人的一律 **Non-absorb** | §1、§8 Non-goals |
| **模式抽取** | 不抄功能表；抽可迁移机制，映射到 ASH 包边界 | §3 模式目录 |
| **功能点全量** | 各产品功能清单作证据，不作扩面清单 | §5 |
| **使用场景** | 主用户路径对照，标 ASH 已有 / 缺口 / 不跟 | §6 |
| **附录矩阵** | 能力维 × 五产品，供评审扫一眼 | 附录 A |

### 0.2 扫描基线

| 产品 | 路径 | 主文档 |
|------|------|--------|
| **DSH** | `C:\Go_Work\src\deepseek-harness` | `docs/platform-blueprint/README.md`（包 `0.1.6-alpha.1`） |
| **Pi** | `C:\Go_Work\src\pi` | `doc/platform/README.md`（v0.85.1） |
| **Codex** | `C:\Go_Work\src\codex` | `analysis/README.md` |
| **Hermes** | `C:\Go_Work\src\hermes-agent` | `docs/platform-analysis/INDEX.md` |
| **ASH** | 本仓库 | `doc/design/*` · v5 规格 · Doctor/OpenAPI |

本文不构成对外部产品的 SLA 承诺；以本地扫描与 ASH 现行设计为准。

---

## 1. 定位过滤（ASH 北极星）

### 1.1 ASH 一句话

> **可审计的交付编排平台**：Scenario DSL + Run 状态机 + Agent×Memory 双核 + Doctor/证据链；交互做薄，评审/管控做厚。

### 1.2 通过门（Absorb 必须同时满足）

1. **强化交付闭环**：契约、门禁、Artifacts、签字、回放、KPI 之一  
2. **可落在现有包边界**：不引入 Cordis 式全插件树、不替换 Go Worker  
3. **可观测 / 可评审**：事件可 derive、可进厚评审或 Doctor  
4. **不稀释差异化**：不与 Qoder/IDE、DSH Web、Hermes IM 主战场正面硬刚

### 1.3 明确不吸收（Non-absorb）

| 来源 | 不吸收项 | 原因 |
|------|----------|------|
| DSH | Cordis / HMR / Slot SPA 替换控制台 | 运行时哲学冲突；v5 已决议 |
| DSH | Code Mode / LSP 全集 / PTY 终端产品化 | 非交付主战场 |
| Pi | 以 TUI 为主产品面 | ASH 主面是 Web + OpenAPI |
| Pi | 「核心刻意无权限」哲学 | 与 ASH 门禁文化相反 |
| Codex | Desktop/IDE 源码级对标、ChatGPT 账户体系 | 外链产品；非本仓边界 |
| Codex | Pets / 语音等体验枝节 | 无治理增益 |
| Hermes | 十余 IM 通道全量适配 | 运维面爆炸；可选单通道 POC 另议 |
| Hermes | YOLO / 弱隔离默认 | 与 fail-closed 冲突 |
| 全部 | 公网 Marketplace、Active-Active | v5 Non-goals 延续 |

---

## 2. 五产品定位与优缺点

### 2.1 定位对照

| 产品 | 一句话 | 主用户 | 真相源 | 卖点 |
|------|--------|--------|--------|------|
| **DSH** | 一切皆插件的 Agent 运行时 | 开发者本地 / 自动化 | SessionEvent 日志 | 可组合性、工具广度、沙箱缝 |
| **Pi** | 极简 CLI Coding Agent Harness | 终端重度用户 | JSONL 会话树 | 小核心、扩展、pi-ai、TUI |
| **Codex** | 企业级 CLI/TUI + app-server | 个人与团队编码 | Thread + rollout JSONL | 沙箱、审批、Hooks、daemon/远程 |
| **Hermes** | 多通道 Agent + Dashboard | 个人助手 / 跨 IM 运维 | SessionDB + config | Gateway、Cron、Skills Hub、记忆插件 |
| **ASH** | 交付编排 + 双核治理 | 团队交付 / 合规运维 | Scenario DSL + Run/events | DSL、Doctor、Memory 评审、RLS、Artifacts |

```
        工具/运行时灵活性
                 ↑
        DSH · Codex(sandbox)
                 |
   Pi(极简) -----+----- Hermes(通道)
                 |
                 ↓
        ASH（契约 · 门禁 · 证据 · 双核）
```

### 2.2 优缺点摘要

#### DSH

| 优点 | 缺点 |
|------|------|
| 事件溯源「可见即已记录」、derive 投影清晰 | Cordis 学习成本高，组合爆炸 |
| Profile/Bundle/Patch 可替换执行世界 | 缺交付契约 / Doctor / 企业 RLS |
| 工具与沙箱缝完整（Landlock/SSH/E2B） | Web UI 强绑定插件 Slot，难当企业控制台 |
| ACP/SDK/Webhook 多入口 | Memory 非一等公民 |

#### Pi

| 优点 | 缺点 |
|------|------|
| 核心极小，扩展边界干净 | 无内置权限/沙箱（安全靠外部） |
| 会话树 fork/tree/compact 体验一流 | 无 Web 运维面、无多租户 |
| `pi-ai` Provider 面广；RPC/SDK 易嵌入 | 刻意不做 MCP/子 Agent 内置，能力靠生态 |
| 供应链硬化（pin/ignore-scripts） | 与「交付治理」正交 |

#### Codex

| 优点 | 缺点 |
|------|------|
| 跨平台沙箱 + execpolicy + 审批门成熟 | 核心绑定 OpenAI 生态与账户 |
| Hooks 生命周期完整、可审计扩展点 | Desktop/IDE 不在本仓，协议面复杂 |
| app-server 统一客户端；Skills/MCP/Plugins | 记忆/项目能力偏实验 |
| `doctor`、OTEL、daemon 远程控制 | 无 Scenario DSL / Artifacts 四件套 |

#### Hermes

| 优点 | 缺点 |
|------|------|
| Gateway 多 IM 一等公民；Cron 投递 | 通道维护成本高；安全默认偏宽松 |
| Skills Hub + 自主沉淀技能 | 无企业 RLS / 发布签字文化 |
| 记忆可插拔（mem0 等，至多 1 个） | Dashboard 偏本机运维，非组织治理 |
| 终端 backends 多样（docker/ssh/modal…） | 工具面杂，与交付契约弱耦合 |

#### ASH（自评）

| 优点 | 缺点 |
|------|------|
| DSL + Artifacts + Doctor + 证据链 | Agent 工具广度弱于 DSH/Codex |
| Memory L0–L2 评审治理独有 | 会话树 / Hooks / 多通道仍弱 |
| v5：薄交互 + 厚评审 + Space 管控 | Provider/沙箱 OS 级隔离仍阶段性 |
| 多租户 RLS、Org 样板 | 扩展生态（Hooks/Skills Hub）不及 Codex/Hermes |

---

## 3. 模式目录（抽取 → 映射 → 决策）

图例：`Absorb` = 建议进 v6 规格 · `Partial` = 已有雏形需加深 · `Observe` = 跟踪不落地 · `Reject` = 定位过滤拒绝

| ID | 模式 | 最佳来源 | ASH 现状 | 决策 | 建议落点 |
|----|------|----------|----------|------|----------|
| P01 | Append-only 事件真相 + derive 投影 | DSH | v5 visibility / ConversationNode / FoldThread | **Partial** | 硬化「模型可见 ⟺ 已记录」；减少双写 |
| P02 | UI 只发意图、只渲投影 | DSH | v5 薄交互 / Quest 去厚 | **Partial** | Agent Chat 对标收尾；禁止 FE 私算分 |
| P03 | Capability seams（fs/tools/sandbox 可替换） | DSH | ToolBus + Sandbox + Provider | **Partial** | 缝接口文档化；不引入 Cordis |
| P04 | Profile → Bundle 组合 | DSH | Harness Profile | **Partial** | Profile 版本评审已有；不叠加 patch 热更 |
| P05 | JSONL 会话树（fork/tree/clone） | Pi | Run replay；interaction threads | **Absorb** | Thread 分支元数据 + compare；非另起 TUI |
| P06 | Compaction + spill 大结果 | Pi / DSH | compaction 事件 / spill 已有 | **Partial** | 对话面 `/compact` 语义 + 评审可回放摘要 |
| P07 | Steering / follow-up 队列 | Pi | Session interrupt/queue 部分 | **Absorb** | 运行中 steer vs 完成后 queue 语义写进协议 |
| P08 | 极简核心 + 扩展包 | Pi | gRPC Plugin + Skills | **Observe** | 保持 Go 边界；Skills 深化即可 |
| P09 | 统一 LLM Provider 层 | Pi (`pi-ai`) | Model Router + ASH_LLM | **Partial** | Provider 目录 + 健康探针进 Doctor |
| P10 | RPC/JSONL 嵌入模式 | Pi / Codex | OpenAPI + SSE；ACP 已有 | **Absorb** | 可选 `ash session --mode rpc` 供 CI/嵌入 |
| P11 | 跨平台 OS 沙箱 + execpolicy | Codex | Policy + sandbox POC / Landlock 路线 | **Absorb** | execpolicy 声明式 + Doctor 能力位 |
| P12 | 工具审批门（once / always / reject） | Codex / DSH | `waiting_approval` | **Partial** | 预设档 + 一次/永久允许写入 SpacePolicy |
| P13 | Hooks 生命周期（Pre/Post Tool…） | Codex | 弱 / CI webhook 有限 | **Absorb** | `ash.hooks.v1`：可审计、可评审、默拒危险 |
| P14 | app-server / daemon 多客户端 | Codex | 单 Worker HTTP | **Observe** | 保持单体；ACP/OpenAPI 足够 |
| P15 | Skills + Hooks + MCP 三件套 | Codex | Skills 目录 + MCP 桥 + 无 Hooks | **Absorb** | Hooks 补齐；MCP exec 收尾 |
| P16 | 多 Agent 图 / subagent | Codex / DSH | Sub-run / ACP | **Partial** | 谱系写入 interaction_threads |
| P17 | Messaging Gateway 多通道 | Hermes | Webhook / CI 有限 | **Observe→可选** | v6 仅设计「单通道适配器缝」；不全量 IM |
| P18 | Cron + 投递到通道 | Hermes | Waker + Webhook 入库 | **Partial** | Cron 与 Waker 对齐；Webhook 非重复入库也走同一 Notifier（VX61）。无 IM |
| P19 | Skills Hub + 自主沉淀 | Hermes | skill catalog（无公网市场） | **Partial** | 组织内 Hub；沉淀须经记忆/编排评审 |
| P20 | 可插拔 Memory Provider | Hermes | 内置 Memory Core | **Reject 替换** / **Observe 桥** | 不以外置 mem0 替换双核；可 MCP 旁路 |
| P21 | 多终端 Backend（docker/ssh/…） | Hermes / DSH | Sandbox backends 路线 | **Partial** | Backend 枚举进 Harness Profile |
| P22 | Dashboard 本机运维八页 | Hermes | 三主题控制台更厚 | **Reject 复制** | 借「运维动作状态」卡片模式即可 |
| P23 | Scenario DSL + Artifacts + Doctor | ASH | 已有 | **Keep** | 差异化护城河 |
| P24 | Memory L0–L2 + 厚评审评分 | ASH v5 | 已有 | **Keep** | 继续 Harden，勿被外部「轻记忆」带回退 |

**吸收优先级（移交 B / v6）：**

| 优先级 | 模式 ID | 一句话 |
|--------|---------|--------|
| **P0** | P13, P11, P07 | Hooks · execpolicy/沙箱声明 · steer/queue 语义 |
| **P1** | P05, P06, P15, P12 | 会话树分支 · compaction UX · Skills/MCP/Hooks 闭环 · 审批预设 |
| **P2** | P10, P16, P18, P21 | RPC 嵌入 · 子代理谱系 · Cron 投递 · Backend 枚举 |
| **P3** | P17, P09, P19 | 单通道 Gateway 缝 · Provider 目录 · 组织 Skills Hub |

---

## 4. ASH 现行设计对照（v2 / v5）

| ASH 层 | 已具备 | 相对四产品仍弱 | v6 建议方向 |
|--------|--------|----------------|-------------|
| 入口 | Web 三主题、CLI、Webhook、ACP、Session | 无 Pi 级 RPC 模式；无 Hermes 级 IM | P10；P17 仅缝 |
| Agent Core | Run、Harness、ToolBus、Sandbox POC、Skills | Hooks 缺失；execpolicy 弱于 Codex；工具广度弱 | P13、P11、P15 |
| Memory Core | L0–L2、评审、RAG、hit_used | 无「外置记忆超市」——**应保持** | Keep P24 |
| 治理 | DSL、Gates、Doctor、RLS、Releases | — | Keep P23 |
| 演进 | Feedback、Improve、评分、Workbench | 交互监控可再对齐 DSH Trajectory | P01/P02 硬化 |
| 薄交互 | ConversationNode、意图 API、Agent Chat 对标中 | fork/tree、steer 语义未产品化 | P05、P07 |

结论：v5 已吃掉 DSH 的「薄交互 + 事件投影」主课；**v6 主课应转向 Codex 的可审计扩展点（Hooks/execpolicy）与 Pi 的会话语义（树/compact/steer），Hermes 只借「适配器缝 + Cron」而非通道全集。**

---

## 5. 功能点全量清单（证据）

### 5.1 DSH（摘自 platform-blueprint §4）

**用户面**

- Web / Headless / SDK / ACP 多入口  
- 工作区与会话（创建/重命名/排序/归档/搜索）  
- 流式对话、推理展示、工具卡；Chat ↔ Trajectory  
- `@file` / `@session` / `/` 命令与 Skills；附件/图片/Present  
- 模型与 API Key、Onboarding；权限预设 + 一次性审批  
- Plan Mode、Goal、Todo、Jobs；子代理谱系与续写/中断  
- 右侧栏：文件树 / 预览 / 终端；Agent Presets、插件配置卡  
- 会话导出 ZIP、消息反馈；主题/字号/语言  
- Schedule UI（默认 disabled）；Desktop Electron（不走 loopback）

**模型工具（类）**

- Shell：`bash` `pwsh`  
- FS：`read` `write` `edit` `glob` `grep` `str_replace_editor` `present` …  
- Terminal：`terminal_open/list/read/send/signal/close`  
- Jobs：`job_list` `job_output` `job_kill`  
- Web：`web_search` `web_fetch`  
- Subagent：`subagent` `subagent_fork` `send_message` `interrupt_agent` …  
- Workflow / PTC：`workflow` `run_code`  
- Plan/Goal/Schedule/Todo；Skill/LSP/Ask；Session query；MCP 动态工具  
- Experimental：`cordis_*` `spawn_teammate` browser/computer-use …

**平台缝**

- LLM 适配、沙箱模式、审批策略、设置/凭证、持久化/投影/Spill/Checkpoint、Telemetry、HMR、Typert Remote、Webhook→Session、SSH 远程世界、Hooks 桥接

### 5.2 Pi（摘自 doc/platform/features.md）

**运行与分发**：TUI / print / JSON / RPC / SDK；`pi update`；offline；遥测  

**认证与模型**：`/login` 订阅+API key；`/model` `/thinking`；scoped-models；`models.json`；registerProvider  

**工具**：默认 `read` `write` `edit` `bash`；可选 grep/find/ls/powershell；`!`/`!!` 用户 shell  

**会话**：JSONL 树；`-c/-r/--fork`；`/new/resume/name/session/tree/fork/clone/export/import/share`  

**上下文**：AGENTS.md / CLAUDE.md / SYSTEM.md；skills；prompt 模板  

**Compaction**：自动阈值/overflow；`/compact`；branch summary  

**信任**：首次项目信任 → `trust.json`；无内置沙箱  

**扩展**：TS 扩展；`pi install|remove|list|update`；`/reload`  

**体验**：steering / follow-up；折叠 thinking/tools；主题；粘贴图片；`@` 文件  

**集成**：RPC JSONL；SDK AgentSession；实验远程 CBOR/chord  

### 5.3 Codex（摘自 analysis/01-modules-and-features.md）

**表面**：CLI、TUI、exec、app-server、daemon/remote-control、exec-server、TS/Python SDK、MCP；Desktop/IDE/Web 外链  

**CLI 子命令**：agents、exec、review、login、mcp、plugin、app-server、remote-control、app、resume/fork/archive、queue、cloud、sandbox、doctor、update、features、apply、execpolicy…  

**TUI slash（节选）**：`/model` `/permissions` `/approve` `/memories` `/skills` `/hooks` `/review` `/worktree` `/compact` `/plan` `/goal` `/agents` `/subagents` `/mcp` `/apps` `/plugins` `/daemon` …  

**平台能力**：ChatGPT/API 认证；thread 生命周期；turn steer/interrupt；shell/apply_patch/MCP；Seatbelt/bwrap/MXC 沙箱；Skills/Plugins/Marketplace/Hooks；agent-graph；cloud-tasks；OTEL/feedback/doctor  

**Hooks 事件**：PreToolUse、PermissionRequest、PostToolUse、Pre/PostCompact、SessionStart/End、UserPromptSubmit、SubagentStart/Stop、Stop、Interrupt  

### 5.4 Hermes（摘自 platform-analysis/MODULES.md）

**通道**：Classic CLI、Ink TUI、Messaging Gateway（Telegram/Discord/Slack/WhatsApp/Signal/Matrix/…/钉钉/飞书/企微/微信/QQ…）、OpenAI API Server、Webhook、Dashboard、ACP、MCP serve、Cron  

**Agent 核心**：多轮 tool-calling、预算、中断/steer/queue；压缩；子代理 `delegate_task`；Todo/Memory/Session FTS；Trajectory/batch  

**Toolsets**：web/terminal/file/vision/image_gen/skills/browser/tts/todo/memory/session_search/clarify/code_execution/delegation/cronjob/messaging/homeassistant/moa/rl/飞书…  

**终端 Backends**：local · docker · ssh · modal · daytona · singularity  

**记忆/技能**：Builtin MEMORY + 外置 provider（至多 1）；Skills Hub ~70+；自主沉淀  

**安全运维**：危险命令审批、DM pairing、YOLO、容器隔离、Webhook HMAC、Profiles、doctor、backup、OpenClaw 迁移、hooks  

### 5.5 ASH（现行能力摘要，对照用）

**入口**：Worker HTTP `:8080`、`/ui` 三主题控制台、CLI（run/doctor/…）、Webhook/CI、ACP provider、Session Chat（v5 薄交互）  

**Agent**：Scenario DSL 三场景、Run 状态机、SSE、Harness Profile、ToolBus、Sandbox 路线、Skills 目录/catalog、Sub-run/ACP、waiting_approval（citation/tool_risk/human）、Artifacts 四件套  

**Memory**：L0–L2、候选评审、TTL/confidence、RAG FTS/Hybrid/vector 路线、hit_used、Wiki/Skills contextRefs  

**治理**：Doctor TR/M 套件、RLS、Org 样板、Releases/signoff、data-policy  

**演进 v5**：Interaction threads、评分 Rubric、SpacePolicy、Registry、Review Workbench、Improve、score_appeal  

**相对缺口（相对 §5.1–5.4）**：Hooks 生命周期、声明式 execpolicy、会话树 fork/UX、steer/queue 产品语义、RPC 嵌入模式、IM Gateway、OS 级沙箱完备度、工具广度（web_search/PTY/LSP 等）

---

## 6. 使用场景对照

### 6.1 场景索引

| ID | 场景 | DSH | Pi | Codex | Hermes | ASH |
|----|------|-----|----|-------|--------|-----|
| S1 | 首启到第一次任务 | Web Onboarding→workspace→session | 信任选择→login→TUI | login→TUI thread | setup→model→通道就绪 | Org/Space→Quest/Chat 或 `ash run` |
| S2 | 日常改代码对话 | Chat+工具卡+Trajectory | TUI+steering | TUI+/diff | 任意通道 chat | Agent Chat / Run 步骤 |
| S3 | 敏感工具审批 | Permission preset + once/always | 无内置（扩展/容器） | `/permissions` + approval | 危险命令审批 / YOLO | waiting_approval + SpacePolicy |
| S4 | 计划后执行 | Plan mode / Goal | 扩展/Skills（核心无） | `/plan` `/goal` | Todo / 长任务 | Scenario Spec + Goal 路线 |
| S5 | 长上下文续聊 | compaction/spill | `/compact` + 树摘要 | `/compact` + Hooks | 自动压缩 | compaction 事件；对话 UX 弱 |
| S6 | 分支探索 | 子代理 fork | `/fork` `/tree` | `/fork` `/side` | branch/rollback slash | replay 新 Run；缺会话树 |
| S7 | 子代理委派 | subagent 多后端 | 扩展 | `/subagents` agent-graph | `delegate_task` | ACP / Sub-run |
| S8 | 扩展点介入 | Cordis patch / hooks 桥 | TS Extension | **Hooks 一等** | hooks + plugins | Plugin ABI；**缺 Hooks** |
| S9 | 定时/后台 | Jobs/Schedule | — | queue/cloud | **Cron+投递** | Waker；投递弱 |
| S10 | 跨端/远程 | SSH 世界 / ACP | 实验 CBOR | **daemon remote-control** | Gateway 多 IM | ACP；无 IM |
| S11 | 知识沉淀 | MCP memory 示例 | Skills/上下文文件 | `/memories`（实验） | Memory 插件 + 技能沉淀 | **Memory 评审主路径** |
| S12 | 交付签字 | session 导出 | export/share | review/exec | dashboard 运维 | **Artifacts + Doctor + signoff** |

### 6.2 ASH 应跟进的场景（定位过滤后）

| 场景 | 动作 | 不跟 |
|------|------|------|
| S3 | 审批预设 once/always 写入策略包 | YOLO 默认 |
| S5–S6 | compact UX + thread fork/compare | 纯 TUI 产品 |
| S8 | Hooks 可审计生命周期 | Cordis 热补丁 |
| S9 | Cron 与通知适配器 | 全 IM 投递矩阵 |
| S11 | 保持 Memory 评审；Skills 沉淀须过评审 | 外置 mem 替换双核 |
| S12 | 继续加深（护城河） | — |

### 6.3 典型路径（文字版）

**DSH S1**：`dsh web` → Cookie URL → Onboarding Key → 选 workspace → `session.create` → Send → Turn（工具卡/审批）。  

**Pi S2**：`pi` → 信任项目 → 输入 → agentLoop → 工具流；Esc 中止；Enter steer / Alt+Enter follow-up；`/compact` 压缩。  

**Codex S3**：工具触发 → PermissionRequest Hook → TUI 审批 → once/always/reject → PostToolUse Hooks → 续 turn。  

**Hermes S9**：`hermes cron` 建 job → scheduler → Agent 跑 → `deliver` 到 Telegram/飞书等。  

**ASH S12**：选 Scenario → Run → gates → Artifacts digest → Doctor/TR → Releases/signoff；并行 Memory 候选进评审队列。

---

## 7. 优缺点综合结论

1. **DSH** 是「运行时教科书」：事件模型与薄 UI 已被 v5 吸收；余下价值在 seams 文档化，而非 Cordis。  
2. **Pi** 是「会话语义教科书」：树、compact、steer/queue 最值得产品化进 ASH Session/Thread。  
3. **Codex** 是「安全与扩展教科书」：Hooks + execpolicy + 审批预设是 v6 最高 ROI。  
4. **Hermes** 是「通道与定时教科书」：只借适配器缝与 Cron，不借 IM 全集与 YOLO。  
5. **ASH** 护城河仍在 DSL/Doctor/Memory/RLS/厚评审；演进应 **加固护城河周边的可审计执行面**，而不是扩成第五个 Coding Agent。

---

## 8. A→B 交接清单（v6 规格输入）

### 8.0 Agent / 管控&评审 交互原型（已绘 · 收敛）

[`doc/prototypes/agent-absorb/`](../prototypes/agent-absorb/README.md) · [`index.html`](../prototypes/agent-absorb/index.html)

| 主题 | 内容 |
|------|------|
| **Agent** | 仅主 Chat 壳（P02）；状态灯深链 |
| **记忆** | 记忆体（分层/视角/候选/TTL/Link/资产）+ 知识（RAG/Wiki/LSP）；厚评审深链管控台 |
| **管控&评审**（原「评审管控」） | Trajectory · 会话树 · Compact · Steer/Queue · 审批 · Hooks · 谱系（P01/P05–P07/P12–P13/P16） |

原则：**交互做薄 · 管控做厚**；不在 Agent 面堆叠原 1–7 菜单。

### 8.1 建议程序名

**ASH v6：可审计执行面** = Hooks + ExecPolicy/Sandbox 声明 + Session 语义（树/compact/steer） + 可选适配器缝  

### 8.2 必须写进 v6 规格的决策草案（待 B 确认）

| ID | 草案 |
|----|------|
| V6-D1 | 不引入 Cordis；不替换控制台为 DSH Web |
| V6-D2 | Hooks 为平台一等事件，默认 fail-closed，变更走编排评审 |
| V6-D3 | ExecPolicy 与 SpacePolicy/Harness Profile 合并解析（更严者优先） |
| V6-D4 | Session/Thread 支持 fork 元数据与 compact 摘要事件；UI 只投影 |
| V6-D5 | Gateway 仅定义 `Notifier`/`IngressAdapter` 接口；首期 0～1 个参考实现 |
| V6-D6 | 外置 Memory Provider 不得替换 Memory Core；仅 MCP 旁路 |
| V6-D7 | 业务 Sprint 与 v4.x/v5 freeze 节奏协调（规格可先行） |

### 8.3 建议分期（规格级，非排期承诺）

| 阶段 | 主题 | 模式 | 状态 |
|------|------|------|------|
| v6.0 | Hooks + 审批预设 + execpolicy 骨架 | P13, P12, P11 | **已冻结**（含下两行主题；见 [`v6.0-release-scope.md`](v6.0-release-scope.md)） |
| v6.1 | Session 树 / compact / steer 语义与投影 | P05, P06, P07 | 已并入 v6.0 冻结水位 |
| v6.2 | Skills/MCP/Hooks 闭环 + 子代理谱系 | P15, P16 | 已并入 v6.0 冻结水位 |
| v6.3 | RPC 嵌入 + Cron 通知适配器 + Backend 枚举 | P10, P18, P21 | **已冻结**（[`v6.3-release-scope.md`](v6.3-release-scope.md) · VX31–VX36） |
| v6.4 | 单通道 Ingress；Provider 目录；组织 Skills Hub | P17, P09, P19 | **已冻结**（[`v6.4-release-scope.md`](v6.4-release-scope.md) · VX41–VX46） |
| v6.5 | 已有 GitHub Webhook 经 Ingress 缝 | P17 | **已冻结**（[`v6.5-release-scope.md`](v6.5-release-scope.md) · VX51–VX52） |
| v6.6 | Webhook 成功入库后走 Notifier | P18 | **已冻结**（[`v6.6-release-scope.md`](v6.6-release-scope.md) · VX61–VX62） |

> 2026-09-20：v6.0–v6.2 主题经 EW W0–W12 落地后，以 **v6.0** 一代冻结（EW130 / `make v6-signoff`）。  
> 2026-09-20: v6.0–v6.2 themes shipped via EW W0–W12 are frozen as **v6.0** (EW130).

---

## 附录 A · 能力矩阵（五产品）

图例：● 强 / ◐ 中或局部 / ○ 弱或无 / — 刻意不做

| 能力维 | DSH | Pi | Codex | Hermes | ASH |
|--------|-----|----|-------|--------|-----|
| 插件/扩展运行时 | ● | ● | ● | ◐ | ◐ |
| 事件溯源会话 | ● | ● | ● | ◐ | ● |
| 会话树 fork | ◐ | ● | ● | ◐ | ○ |
| Compaction | ● | ● | ● | ● | ◐ |
| 工具广度 | ● | ◐ | ● | ● | ◐ |
| OS 沙箱 | ● | ○ | ● | ◐ | ◐ |
| 审批/权限产品 | ● | — | ● | ◐ | ● |
| Hooks 生命周期 | ◐ | ◐ | ● | ◐ | ○ |
| MCP | ● | — | ● | ● | ◐ |
| Skills | ● | ● | ● | ● | ● |
| 子 Agent | ● | — | ● | ● | ◐ |
| 多 IM 通道 | ○ | ○ | ○ | ● | ○ |
| Cron/调度 | ◐ | ○ | ◐ | ● | ◐ |
| Web 控制台 | ● | ○ | ○* | ● | ● |
| TUI | ○ | ● | ● | ● | ○ |
| 交付 DSL/Artifacts | ○ | ○ | ○ | ○ | ● |
| Doctor/发布门禁 | ○ | ○ | ◐ | ◐ | ● |
| Memory 治理评审 | ○ | ○ | ◐ | ◐ | ● |
| 多租户 RLS | ○ | ○ | ○ | ○ | ● |
| 厚评分/评审台 | ○ | ○ | ○ | ○ | ● |

\* Codex Desktop/Web 在仓外。

---

## 附录 B · 关联文档

| 文档 | 说明 |
|------|------|
| `deepseek-harness/docs/platform-blueprint/*` | DSH 蓝图 |
| `pi/doc/platform/*` | Pi 平台扫描 |
| `codex/analysis/*` | Codex 分析包 |
| `hermes-agent/docs/platform-analysis/*` | Hermes 分析包 |
| [`HLD-双核心-v2.md`](../design/HLD-双核心-v2.md) | ASH 双核 |
| [`v5-governance-program.md`](v5-governance-program.md) | v5 程序 |
| [`2026-09-13-v5-dual-core-governance-design.md`](../../docs/superpowers/specs/2026-09-13-v5-dual-core-governance-design.md) | v5 规格 |

---

## 附录 C · 修订记录

| 日期 | 说明 |
|------|------|
| 2026-09-17 | A 初稿：四产品模式抽取 + 定位过滤 + 全功能点 + 场景 + 矩阵；交接 v6 输入 |
| 2026-09-17 | 挂接 Agent 吸收交互原型 C 包（`doc/prototypes/agent-absorb/`） |
| 2026-09-17 | 原型收敛：Agent 仅主壳；1–7 →「管控&评审」；主题更名 |
| 2026-09-17 | 原型补全记忆全页（记忆体 10 + 知识 3 子页） |
