# Qoder 与 ASH 功能比对分析

> 状态：调研稿（2026-08-09）  
> 归属：[`plan/`](README.md)  
> 对照基线：ASH `v0.1.0-mvp`（PRD/ARCH/MVP 范围）  
> 外部资料：[Qoder 产品概述](https://docs.qoder.com/zh) · [Qoder CN IDE](https://docs.qoder.cn/user-guide/what-is-qoder-cn)

## 1. 目的

梳理 [Qoder](https://qoder.com/zh) 全系列产品与能力，与当前 ASH 实现做功能级比对，明确**重叠点**、**差异点**与**可借鉴方向**，供产品/架构决策参考。本文不构成对 Qoder 的完整功能承诺或 SLA 背书，以外部公开文档与 ASH 仓库现状为准。

---

## 2. Qoder 产品矩阵

[Qoder 官方定位](https://docs.qoder.com/zh)：面向真实工作的 **Agentic 平台**，强调「理解 → 规划 → 执行 → 验证 → 迭代」闭环，而非单次补全或对话。

| 产品 | 形态 | 核心用途 |
|------|------|----------|
| **Qoder IDE** | 独立桌面 IDE | Editor（边写边问）+ **Quest**（长程任务委派、看板、进度、产物审查） |
| **Qoder CN IDE** | 国内版桌面 IDE（原通义灵码） | 同上 + 国内模型（GLM/DeepSeek/Kimi 等）、Credits 计费 |
| **JetBrains 插件** | IDE 插件 | 补全、Ask、Agent、MCP、规则，不离开 JetBrains |
| **Qoder CLI** | 终端 Agent | 终端原生协同开发、脚本/自动化 |
| **Cloud Agents** | 云端 API | 配置 Agent/Environment、Session、流式托管执行 |
| **QoderWork** | 办公 Agent | 文档、表格、调研、浏览器、桌面任务 → 本地交付物 |
| **QoderWake** | 数字员工 | Waker：持续性职责、自动化、多阶段流程 |
| **移动端 / 网页版** | 远程审阅 | 离开电脑后查看任务、审计划、处理审批 |
| **企业版** | 组织治理 | 采购、成员、身份、策略、知识、模型、市场、审计 |

### 2.1 Qoder IDE / CN 共性能力

- **代码补全**：行级/函数级、跨文件上下文
- **Ask Mode**：研发问答、工程问答、多模态图片
- **Agent Mode**：自主决策、工程感知、多文件修改、终端命令、MCP
- **Quest 2.0**：任务拆解、Spec、Diff 审查、Commit/PR
- **Experts / Subagent**：多专家并行协作
- **Repo Wiki / 知识中心**：自动沉淀架构、接口、业务知识
- **Memory**：个人/工程记忆
- **MCP / Skills / 插件市场**
- **规则（Rules）**、代码库索引、上下文压缩
- **企业**：统一授权、统计报表、私域知识库、VPC 专属部署

---

## 3. ASH 产品定位（对照基线）

ASH 定位：**交付驱动的编排机器人**，不是 IDE。详见 [`PRD-需求文档.md`](../design/PRD-需求文档.md)、[`ARCH-架构与技术选型.md`](../design/ARCH-架构与技术选型.md)。

| 维度 | ASH |
|------|-----|
| 形态 | Go **Worker** + **CLI** + **React 控制台**（12 页） |
| 核心闭环 | Run → 场景 DSL → 角色步骤 → **Artifacts 四件套** → 可回放 |
| 原则 | 交付闭环、安全审计、可回放、可插拔（Provider/RAG/MCP/观测） |
| 场景 | `feature_delivery` / `hotfix` / `security_patch`（YAML + 人工门禁） |
| 质量 | **Doctor TR0–TR3**（43 用例）、发布/回滚门禁脚本 |

控制台页面：Runs、Memory、Compliance、CI、Metrics、Observability、Releases、Scale、Doctor、Automation、Space 等。

---

## 4. 功能点对照

### 4.1 相同 / 高度重叠

| 能力域 | Qoder | ASH | 说明 |
|--------|-------|-----|------|
| **编程 Agent** | Agent Mode / Quest | Runs + ToolBus + Agent 执行 | 多步改代码、跑命令 |
| **长程任务** | Quest 看板 + 进度 | Run 状态机 + SSE 事件流 | 放手执行 + 事后审查 |
| **工具调用** | 内置工具 + MCP | ToolBus + MCP | 扩展工具生态 |
| **人工审批** | Quest Diff 审查、拒绝变更 | `waiting_approval`（citation/tool_risk/human） | ASH 偏**流程门禁** |
| **工程上下文** | 工程自动感知、索引 | RAG FTS + citation 门禁 | 有依据再改 |
| **记忆** | Memory（对话沉淀） | Memory L0–L2 + 评审合并 + TTL | ASH 重**治理与审计** |
| **规则** | Rules | Scenario DSL + Policy | ASH 为**可版本化场景** |
| **多模型** | 模型选择器 | Model Router | 多 Provider / 降级 |
| **可观测** | 任务进度、产物 | SSE + Waterfall + Prometheus/KPI | ASH 偏**平台指标与门禁** |
| **企业多租户** | 企业版授权/知识库 | Org/Space/RBAC/RLS + org-templates | 组织隔离 |
| **CLI** | Qoder CLI | `ash` CLI（run/doctor/migrate） | 终端入口 |
| **插件扩展** | 插件市场 + MCP | HTTP/gRPC Plugin ABI + 签名 | 扩展生态 |
| **安全** | 代码安全 | Secret 扫描、脱敏、危险工具 deny | 合规导向 |
| **发布相关** | Commit/PR 推送 | release_notes、rollback_plan、Releases 门禁 | ASH 重**证据链** |

### 4.2 Qoder 有、ASH 弱或无

| Qoder 能力 | ASH 现状 | 差距性质 |
|------------|----------|----------|
| **IDE 内联补全** | 无 | 产品形态不同（编码助手 vs 编排平台） |
| **Editor 现场协作** | 无 Web 内嵌 IDE | 体验层 |
| **Quest 独立工作台** | Runs 页有门禁/SSE，非 Quest 级委派 UX | 体验层 |
| **Experts 多 Agent 专家团** | 场景内固定角色，非动态 Subagent | 协作模型 |
| **Repo Wiki / 知识图谱** | RAG + Memory，无自动 Wiki 产品化 | 知识产品形态 |
| **QoderWork** | Out of Scope（P3） | 办公场景未覆盖 |
| **QoderWake 数字员工** | 无 | 持续性自动化未做 |
| **Cloud Agents 托管** | 自托管 Worker | 部署/商业模式 |
| **移动端审阅** | 仅 Web 控制台 | 触达方式 |
| **Credits 计费** | 有组织样板，无计费系统 | 商业产品化 |
| **多模态（图片问答）** | MVP 未覆盖 | 输入形态 |
| **JetBrains 深度集成** | 无 IDE 插件 | 开发者入口 |

### 4.3 ASH 有、Qoder 弱或不突出

| ASH 能力 | 说明 | Qoder 侧 |
|----------|------|----------|
| **Scenario DSL + 三场景** | 可版本化、Doctor 可验 | Quest 偏自然语言任务 |
| **Artifacts 四件套 + digest** | diff/test_report/release_notes/rollback_plan | Quest 有 Diff，非交付契约 |
| **Run Replay / Resume / Checkpoint** | 运行级审计与恢复 | Quest 有快照回滚 |
| **Doctor TR0–TR3** | 43 用例发布准入 | 无对等测试套件 |
| **Memory 候选→评审→merge** | Librarian、证据强制、TTL | Memory 偏个性化 |
| **合规 TR2** | Secret、审计导出、保留期 API | 企业审计形态不同 |
| **CI 诊断闭环** | sync/diagnose/adopt | 非 CI 归因产品 |
| **KPI 看板（含场景稳定率）** | Metrics + t1-metrics-gate | 企业统计非交付 KPI 体系 |
| **发布治理 API** | checklist/gate/rollback-drill | Quest 可 PR，ASH 有门禁脚本 |
| **Postgres RLS 多租户** | rev 20、41 策略、ash_app | 企业 VPC vs 数据面隔离 |
| **OpenAPI + openapicheck** | API-first 契约 | 平台化差异 |
| **Improve proposals** | canary/experiment/promote | 自我迭代闭环 |
| **组织样板 API** | `org-templates` 一键开通 | 企业方案偏销售/部署 |

---

## 5. 架构与定位差异

```
Qoder 生态                         ASH 生态
─────────────                      ─────────
IDE / JetBrains / CLI              Worker 编排服务
    │                                  │
    ▼                                  ▼
Quest 任务工作台                   Scenario DSL
    │                                  │
    ├─ Repo Wiki / 知识中心              ▼
    │                              Artifacts 四件套
Cloud Agents / Wake / Work             │
                                   Doctor 门禁
                                       │
                                   Web 控制台
```

- **Qoder**：以**开发者工作面**（IDE/终端）为中心，AI 嵌入编码现场，Quest 负责长任务。
- **ASH**：以**交付运行面**（Run/Artifacts/门禁/审计）为中心，AI 嵌入**软件交付流程**。

---

## 6. 总结

| 维度 | 相同处 | 不同处 |
|------|--------|--------|
| 产品形态 | Agent、CLI、记忆、MCP、规则 | Qoder = IDE 产品族；ASH = 编排平台 + 控制台 |
| 任务执行 | 多步自主、工具、终端 | Qoder 强 Quest/Experts；ASH 强 DSL + 状态机 + Replay |
| 知识 | 代码理解、检索 | Qoder Repo Wiki；ASH RAG + 评审式 Memory |
| 质量与发布 | Diff 审查、可提交 | Qoder 偏开发体验；ASH 偏 Doctor/门禁/证据 |
| 企业 | 多租户、审计、策略 | Qoder 计费/VPC/SaaS；ASH RLS/组织样板/API 治理 |
| 边界 | 都不只是聊天 | Qoder 还做办公/数字员工/云托管；ASH 专注交付闭环 |

### 6.1 对 ASH 的启示

1. **不必正面做 IDE 补全**——成本高；应强化「场景 Run + 四件套 + Doctor」差异化。
2. **可借鉴 Quest 委派 UX**——Runs 门禁文案（Sprint CS）可继续强化看板/产物审查。
3. **Repo Wiki 缺口**——可在 RAG/Memory 上增加结构化 Wiki 导出，而非重做 IDE。
4. **Cloud Agents**——若做云托管，更接近 Qoder Cloud Agents，需基于现有 Worker/Plugin ABI 演进。

---

## 7. 参考链接

| 资源 | URL |
|------|-----|
| Qoder 官网（中文） | https://qoder.com/zh |
| Qoder 产品概述 | https://docs.qoder.com/zh |
| Qoder IDE 概述 | https://docs.qoder.com/zh/desktop/overview |
| Quest 概览 | https://docs.qoder.com/zh/user-guide/quest/overview |
| Qoder CN IDE | https://docs.qoder.cn/user-guide/what-is-qoder-cn |
| ASH PRD | [`../design/PRD-需求文档.md`](../design/PRD-需求文档.md) |
| ASH ARCH | [`../design/ARCH-架构与技术选型.md`](../design/ARCH-架构与技术选型.md) |
| ASH MVP 范围 | [`mvp-release-scope.md`](mvp-release-scope.md) |

---

## 8. 修订记录

| 日期 | 说明 |
|------|------|
| 2026-08-09 | 初稿：基于公开文档与 ASH v0.1.0-mvp 仓库现状 |
