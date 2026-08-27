# ASH 面向真实工作的 Agentic 迭代路线图

> 状态：规划稿（2026-08-09）  
> 归属：[`plan/`](README.md)  
> 关联：[`qoder-ash-comparison.md`](qoder-ash-comparison.md) · [`PLAN-进度与里程碑.md`](PLAN-进度与里程碑.md) · [`PRD-需求文档.md`](../design/PRD-需求文档.md) · [`ARCH-架构与技术选型.md`](../design/ARCH-架构与技术选型.md)

## 1. 目的

在 [Qoder 与 ASH 比对](qoder-ash-comparison.md) 基础上，给出 ASH 走向「**面向真实工作的 Agentic**」的迭代方向：**不复制 Qoder IDE**，而是把 Quest 式「委派 → 执行 → 验证 → 迭代」接到现有 **交付 Run + Artifacts 四件套 + Doctor** 之上。

---

## 2. 目标定位（v1.x）

### 2.1 一句话

> **ASH = 交付 Agentic 平台**：用自然语言或 Issue 发起真实交付任务，在可审计场景里跑到四件套，Human 只在门禁点介入。

### 2.2 与 Qoder 的关系

| 维度 | Qoder | ASH 路线 |
|------|-------|----------|
| 主战场 | IDE / 终端编码现场 | 交付运行面（Run / Artifacts / 门禁） |
| Agentic 闭环 | Quest + Editor | Goal → Plan → Run → Verify → Artifacts |
| 差异化 | 补全、工程感知、Repo Wiki | Doctor、digest、Memory 治理、发布/合规 API |
| 不做 | — | IDE 内联补全、QoderWork 式办公、全量 Skill 市场（v1.x） |

### 2.3 ASH 式 Agentic 闭环

```
Goal（自然语言 / Issue / Webhook）
  → Plan（场景路由 + Spec + DSL 步骤）
  → Execute（ToolBus / Agent / MCP）
  → Gate（citation / tool_risk / human）
  → Verify（test / doctor / lint）
  → Artifacts（四件套 + digest）
  → Memory（候选 → 评审 → 合并）
  → Replay / KPI / 发布证据
```

现有能力已覆盖 Execute 之后大部分；弱项在 **Goal/Plan 入口**、**Quest 工作台 UX**、**工程知识产品化**、**多入口委派**。

---

## 3. 能力缺口对照

| Qoder 强项 | ASH 现状 | 本路线图要补 |
|------------|----------|--------------|
| 自然语言委派 + Quest | 必选场景 YAML + inputs | Goal → Plan → Run |
| Repo Wiki / 知识中心 | RAG FTS + Memory 评审 | Repo Profile + Wiki 视图 |
| Experts / Subagent | 场景内固定角色 | 可选 Sub-run（Policy 内） |
| Diff 审查 / 提交 | waiting_approval + 产物 | 审查 UX + 可选 Git 出站 |
| Cloud / CLI / 移动审阅 | Worker + CLI + Web | Session API + 触发器 + 轻量审阅 |

**保留并强化（不对标放弃）**：Scenario DSL、Doctor TR0–TR3、Artifacts digest、Memory 治理链、Releases/Compliance API、Postgres RLS。

---

## 4. 迭代分波

### 波 0：可信底座（P0，1–2 周）

Agentic 会放大不可信输出；须先完成 [`PLAN-进度与里程碑.md`](PLAN-进度与里程碑.md) P0：

| ID | 任务 | 验收 |
|----|------|------|
| P0-1 | `cloud-rds.env` + `make cloud-acceptance` | H-01~H-03 |
| P0-2 | 切换日 backup → migrate → RLS → t0 | runbook §4 证据 |
| P0-3 | 四人真人签字 | `signoff-gate` |
| P0-4/5 | CI 绿 + 切换日 rollback-drill | R-12 |

并行准备 P1：真实 GitHub CI（H-04/05）、ExecGo live（H-06）。

---

### 波 1：Quest 化「委派面」（M4，约 4–6 周）

**目标**：用户说目标，系统出 Plan + Run；控制台成为任务工作台。

| 方向 | 迭代项 | 验收 |
|------|--------|------|
| Goal API | `POST /runs/from-goal`：NL + repoRoot → 场景 + inputs + 可选 Spec | 三场景可从一句话启动 |
| Plan 可见 | Run 详情 Spec/步骤计划；SSE 展示「规划 → 执行」 | 对标 Quest Progress |
| Quest UI | Runs 看板（排队/运行/待审批/完成）、产物/Diff 审查 | 替代「仅 JSON 详情」 |
| 审查闭环 | 单文件/全量 reject；Approve 后继续 | 对标 Quest 审查 |
| CLI 委派 | `ash quest "…" --repo .` | 最小终端委派 |

**原则**：Plan 仍落 **Scenario DSL**（可版本、Doctor 可测）；NL 只做 **路由 + 填槽**。

---

### 波 2：上下文工程 + 工程知识（M4–M5，约 6–8 周）

**目标**：少问人、少 hallucinate；Repo Wiki 走 ASH 路径（非 IDE）。

| 方向 | 迭代项 | 验收 |
|------|--------|------|
| Repo Profile | 自动扫描语言/模块/依赖/测试命令 → `GET /repos/{id}/profile` | Create Run 自动带上下文 |
| Repo Wiki | RAG + Memory → 结构化 Wiki（架构/接口/变更）；控制台「知识」Tab | 语义搜索 + 链到 run/memory |
| contextRefs | Run 绑定 wiki 页、memory id、ci 诊断、file globs | Provenance 可追溯 |
| Space Rules | Space 级 Rules（`.ash/rules` 或 API），与 org-template 联动 | 对标 Qoder Rules |
| 索引增强 | 符号/路径索引（FTS 之上；向量仍 P3） | 大仓检索 P95 可接受 |

**原则**：Wiki 是 Memory/RAG 的**视图**，仍走评审与 TTL。

---

### 波 3：协作 Agent + 验证闭环（M5，约 8–10 周）

**目标**：复杂任务分工，Policy 约束下 Subagent。

| 方向 | 迭代项 | 验收 |
|------|--------|------|
| Sub-run | 父 Run spawn 子 Run（限定工具/目录/预算）；事件树 | hotfix 定界+修复可拆 |
| verify 步骤 | DSL `kind: verify`（test、doctor 子集、lint），可重试 N 次 | 「验证直到过」 |
| Improve 联动 | 失败 Run → proposal → 灰度 replay | KPI-11 稳定率 |
| ExecGo live | M3-05 生产 + 路由策略 | 非 static 探测 |
| Skills 目录 | MCP + 内置 Skill（Automation 页扩展） | 场景可声明 skill 集 |

**原则**：Subagent **不得**绕过 citation/tool_risk/human；危险工具仍默认 deny。

---

### 波 4：多入口 + 企业 Agentic（M6+）

**目标**：离开控制台也能委派、审批、持续运行。

| 方向 | 迭代项 | 验收 |
|------|--------|------|
| Session API | `POST /agents/sessions`：长任务托管、流式事件 | 外部系统集成 |
| Webhook / 触发器 | CI 失败、告警、Issue → 自动开 Run | 与 CI 诊断闭环 |
| 移动/轻量审阅 | 待审批 Push 或 PWA：Plan + Diff + Approve | 对标移动审阅 |
| Waker 雏形 | 定时/事件巡检 Run（TTL、Doctor 子集、KPI 偏差） | 最小持续职责 |
| 企业 Agentic | spawn 策略、配额占位、审计报表 | org-templates 上扩展 |

**v1.x 明确不做**：IDE 补全、QoderWork 办公、Skill Marketplace 全量、向量库主路径。

---

## 5. 架构演进

与 [`ARCH-架构与技术选型.md`](../design/ARCH-架构与技术选型.md) §12 对齐：

| Stage | 形态 | 对应波次 |
|-------|------|----------|
| **0（今）** | Worker 单体 + ToolBus + Web 控制台 | MVP |
| **1** | 外置 RAG Indexer / ExecGo；Plugin gRPC 生产化 | 波 2–3 |
| **2** | Session 服务可选拆分；队列 Worker 水平扩展 | 波 4 |

优先 **进程外插件**，再拆微服务；避免过早分布式。

---

## 6. 建议 Sprint 序列

接续 [`PLAN`](PLAN-进度与里程碑.md) Sprint CY 之后：

| Sprint | 主题 | 产出 |
|--------|------|------|
| **CI–CJ** | P0 发布闭环 | 云 RDS、签字、rollback |
| **DA** | Goal → Run + Plan 事件 | API + CLI 最小委派 |
| **DB** | Quest 工作台 UI v1 | 看板 + 产物/Diff 审查 |
| **DC** | Repo Profile + Wiki 导出 | 知识 Tab + contextRefs |
| **DD** | Space Rules + 场景推荐 | 规则 API + Goal 路由 |
| **DE** | Sub-run + verify 步骤 | DSL 扩展 + Doctor 用例 |
| **DF** | CI/Webhook 触发 Run | 自动化入口 |
| **DG** | Session API / 移动审阅 | 多入口 Agentic |

每 Sprint 要求：**OpenAPI + Doctor/TR 增量 + web-gate**（或等价门禁），不破坏 MVP 发布文化。

---

## 7. 产品原则（防走偏）

1. **Agentic ≠ 更自由的 Chat** — 每次委派绑定场景、Artifacts 契约、可 Replay。
2. **不先做 IDE** — CLI、Webhook、轻插件触达开发者；平台价值在 Run 治理。
3. **Wiki/记忆可审计** — 继承 Memory 评审，不用黑盒「越用越懂」替代。
4. **验证内置** — Doctor + verify 步骤 + KPI-11，闭环可度量。
5. **Human 只在门禁** — citation / tool_risk / human / release；不扩大人工步骤面。

---

## 8. 成功指标（Agentic 阶段）

在现有 KPI（[`kpi-dashboard-definition.md`](kpi-dashboard-definition.md)）上增加或强化：

| 指标 | 含义 | 目标方向 |
|------|------|----------|
| Goal→Run 成功率 | NL 委派一次启动成功 | ≥ 90%（三场景） |
| Plan 采纳率 | 用户未改 Spec 即 Approve 比例 | 上升趋势 |
| 门禁一次通过率 | 无二次 waiting_approval | 对标 KPI-11 场景稳定 |
| Time-to-Artifacts | Goal 到四件套齐全 P95 | 场景基线 -20% |
| Wiki 命中率 | Run 引用 wiki/memory 且 citation 通过 | 可观测 |

---

## 9. 风险与依赖

| 风险 | 缓解 |
|------|------|
| NL 路由漂移 | 场景 DSL 仍为真相源；Goal 仅填槽；Doctor 回归 |
| Subagent 越权 | Policy + 子 Run 工具白名单；R-08 抽测 |
| 与 Qoder 正面 IDE 竞争 | 明确不做补全；卖交付治理 |
| 范围膨胀 | 波次冻结；P3 项（向量/办公/市场）不进 v1.x |

**依赖**：波 1+ 建议在 P0 签字后或本地 `postgres-local-rds-e2e` 稳定后进行；ExecGo live 影响波 3 生产 Agent 质量。

---

## 10. 修订记录

| 日期 | 说明 |
|------|------|
| 2026-08-09 | 初稿：基于 qoder-ash-comparison 与 PLAN v0.2 |
