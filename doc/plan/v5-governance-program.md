# ASH v5 程序：双核管控 × 厚评审 × 薄交互

> 状态：**P2 GV07–10 + Task 11 硬化已落地**；**FE-42 / FE-16 / BE-46 已落地**；**score_appeal（BE-18 remainder）已落地**（audit-backed `appeal` 队列 · keep/void · 无 SQL 迁移）；**FE-44 文案/空态/权限提示已落地**；**v5 signoff 进行中**  
> 归属：[`plan/`](README.md)  
> 规格：[`../../docs/superpowers/specs/2026-09-13-v5-dual-core-governance-design.md`](../../docs/superpowers/specs/2026-09-13-v5-dual-core-governance-design.md)  
> 图集：[`../diagrams/archify/`](../diagrams/archify/README.md)（`ash-v5-*`）  
> 前置：不打断现行 [`v4.x-program.md`](v4.x-program.md)；**业务代码默认等 v4.0 签字后再开 v5.0 Sprint**

## 0. 与 v4.x 的命名隔离

本程序中的 **A/B/C/D** 含义如下，**不等于** v4.x Auth / 企业 Agentic / Stage-1 / 生态：

| 轨 | 主题 | 说明 |
|----|------|------|
| **A** | 总架构纲领 | 薄交互 + 双核数据化 + 厚评审 |
| **B** | 厚评审后台 | 交互监控 · 量纲评分 · Workbench |
| **C** | 双核管控数据化 | Agent/Memory 登记 · SpacePolicy · 空间测评 |
| **D** | 薄交互 | DSH 模式借鉴（事件真相 / 投影 / 意图 API） |

## 1. 产品意图

> **ASH v5 = 薄交互入口 + 双核数据化资产 + 厚评审治理台**

- 保留 Agent × Memory 双核心与 Go Worker  
- Space `kind=user|team` 管控与测评  
- 评审做厚；交互做薄；人工批准仍为升格唯一闸门  

## 2. 分期（**已确认：薄交互优先**）

| 阶段 | 优先级 | 范围 | 目标 |
|------|--------|------|------|
| **v5.0-design** | — | 规格 + 图 + 排期 | 已交付 |
| **v5.0-thin** | **P0** | **现行智能体薄交互** | visibility · 意图 API · ConversationNode · Quest 去厚 |
| **v5.1-obs** | P1 | Session/Thread 可观测 + 记忆关联 | Timeline · MemoryLink · seal/replay/compare |
| **v5.2-gov** | P2 | 管控登记 + 厚评审评分 | Registry · Policy · Rubric · Workbench |
| **v5.3** | P3 | 治理硬化 | Improve · 多签 · Doctor · signoff |

开工 Sprint：**GV01**（薄交互），不先做 Registry。

## 3. 决议（设计锁定）

| # | 决议 |
|---|------|
| G1 | DSH 只借模式，不引入 Cordis / 不替换为 dsh web |
| G2 | Space 用 `kind` 扩展，不新建 Team 表 |
| G3 | 厚评审在 `internal/evolve` 扩展，不拆微服务 |
| G4 | 评分与测评由事件 derive 计算，禁止 UI 私算 |
| G5 | 新表列为 v5 E5 例外（见规格 §4.3） |
| G6 | 设计已确认；**实现从薄交互 GV01 开工** |
| G7 | **优先现行智能体薄交互**；管控/厚评分后置 |

## 4. 设计交付物索引

| 产物 | 路径 |
|------|------|
| 设计规格 | [`docs/superpowers/specs/2026-09-13-v5-dual-core-governance-design.md`](../../docs/superpowers/specs/2026-09-13-v5-dual-core-governance-design.md) |
| 实现排期 | [`docs/superpowers/plans/2026-09-13-v5-governance-implementation.md`](../../docs/superpowers/plans/2026-09-13-v5-governance-implementation.md) |
| 架构图 | [`ash-v5-governance.html`](../diagrams/archify/ash-v5-governance.html) |
| 厚评审时序 | [`ash-v5-review-thick.html`](../diagrams/archify/ash-v5-review-thick.html) |
| 评分数据流 | [`ash-v5-score-monitor.html`](../diagrams/archify/ash-v5-score-monitor.html) |
| 空间管控流程 | [`ash-v5-space-control.html`](../diagrams/archify/ash-v5-space-control.html) |
| 评审状态机 | [`ash-v5-review-lifecycle.html`](../diagrams/archify/ash-v5-review-lifecycle.html) |

## 5. 下一步

1. **FE-44 已落地**（Reviews/Mobile/Observability/Registry/Quest 中文空态与权限提示）
2. **下一步**：`make v5-signoff` 并勾选 [`../checklists/v5.0-signoff.md`](../checklists/v5.0-signoff.md)
3. 门禁：`make v5-signoff`（含 Doctor TR3）

## 6. 修订记录

| 日期 | 说明 |
|------|------|
| 2026-09-13 | 初稿：设计轨立项；A/B/C/D 定义；图集与规格挂接 |
| 2026-09-13 | 挂接全量改造排期/技术方案；明确暂缓 Agent 打包迁移 |
| 2026-09-13 | 实现计划增补 §0.1：全量 BE/FE 功能点拆分与 Sprint 对照 |
| 2026-09-13 | Interaction Timeline 升格为 Session/Thread；seal/replay/compare 保障评审三性 |
| 2026-09-13 | **确认计划：优先实现现行智能体薄交互（P0 / GV01–03）** |
| 2026-09-13 | 补充 Agent↔DSH 交互对齐；顶栏 **Agent 使用 / 评审管控** 切换；GV01 后端切片落地 |
| 2026-09-13 | **P0 完成**：GV02 ConversationNode/意图条/Quest 去厚；GV03 DeriveModelVisible |
| 2026-09-13 | **GV04**：interaction_threads · FoldThread · MemoryLink · by-run/threads API |
| 2026-09-13 | **P2 + Task 11**：Registry/Policy/Rubric/Workbench · multi_sign · TR3-12 · `make v5-signoff` |
| 2026-09-14 | **FE-16**：Mobile Reviews 薄批准面同步徽章/逾期筛选/rubric/分配闸门 |
| 2026-09-14 | **BE-46 / TR3-13**：PolicyPack merge + scoring/rubric 一致性探针落地；下一步 score_appeal 或 FE 文案打磨 |
| 2026-09-14 | **BE-18 remainder / score_appeal**：audit-backed appeal 队列（keep/void）落地；下一步 FE-44 或 signoff |
| 2026-09-14 | **FE-44**：评审/移动/可观测/登记/Quest 中文空态与权限提示落地；signoff 进行中 |
