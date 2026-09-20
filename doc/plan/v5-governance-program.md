# ASH v5 程序：双核管控 × 厚评审 × 薄交互

> 状态：**hardening 已完成**；**三主题 IA 已落地**；**Agent Chat DSH 对标续推**；下一步 = **[`v5.0-release-scope.md`](v5.0-release-scope.md) 草案 → GV06 冻结**  
> Status: next step is the v5.0 freeze draft (GV06).  
> 归属：[`plan/`](README.md)  
> 规格：[`../../docs/superpowers/specs/2026-09-13-v5-dual-core-governance-design.md`](../../docs/superpowers/specs/2026-09-13-v5-dual-core-governance-design.md)  
> IA 升级：[`../../docs/superpowers/specs/2026-09-14-console-three-pillar-ia-design.md`](../../docs/superpowers/specs/2026-09-14-console-three-pillar-ia-design.md)  
> Agent Chat 对标：[`../../docs/superpowers/specs/2026-09-15-agent-chat-dsh-parity-design.md`](../../docs/superpowers/specs/2026-09-15-agent-chat-dsh-parity-design.md)（`874b0ce` · `6e88718` · `122b4c4` · `437e5d4`）  
> 后端缺口：[`../../docs/superpowers/specs/2026-09-15-agent-chat-backend-gaps.md`](../../docs/superpowers/specs/2026-09-15-agent-chat-backend-gaps.md)  
> 图集：[`../diagrams/archify/`](../diagrams/archify/README.md)（`ash-v5-*`）  
> 前置：[`v4.x-program.md`](v4.x-program.md) **已收口**；冻结轨见 [`v5.0-release-scope.md`](v5.0-release-scope.md) / [`sprint-gv06-v50-signoff.md`](sprint-gv06-v50-signoff.md)

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

1. **hardening**：evaluation 排除作废分 + BE-49 — **已完成**
2. **控制台三主题 IA**：Tasks 1–4 **已落地**
3. **Agent Chat DSH 对标**：主路径 **已收口**（见 backend-gaps complete+polish）
4. **v4.x**：四代 **已收口**（见 [`v4.x-program.md`](v4.x-program.md)）
5. **现行**：[`v5.0-release-scope.md`](v5.0-release-scope.md) 草案 → **GV06** 冻结 + `make v5-signoff`
6. 门禁：`make v5-signoff`（已存在）；scope-freeze 待 GV06 接入

## 6. 修订记录

| 日期 | 说明 |
|------|------|
| 2026-09-20 | v4.x 收口后挂接 v5.0 冻结草案（GV06）；下一步实现冻结门禁 |
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
| 2026-09-14 | **v5-signoff**：自动门禁通过；checklist 勾选 FE-44 / score_appeal；门禁 FE 改为 `npx vitest` |
| 2026-09-14 | **FE-45 closeout**：`web-build` 入门禁；vitest 扩 Improve/Mobile；手工项改挂自动证据；下一步=可选人工扫一眼或 backlog freeze |
| 2026-09-14 | **hardening + BE-49**：evaluation 排除 voided；HLD/附录 K / Doctor 水位 ALL60·TR3 13；verify-local 纳入 v5 包；下一步=freeze 或 v4.1 |
| 2026-09-14 | **三主题 IA 规格确认**：Agent/记忆/评审管控卡片切换；默认大 Chat+左历史；更多+账号下拉；保留 v5 能力 |
| 2026-09-15 | **三主题 IA Tasks 1–4 落地**：主导航三主题+更多/账号；Agent 大 Chat；记忆多视角+知识 Tab；评审子导航；下一步=可选 polish 或 freeze/v4.1 |
| 2026-09-15 | **Agent Chat DSH 对标落地**：ListSessions + 空白事件投影 `874b0ce`；三栏 FE 壳 `6e88718`；下一步=可选 polish 或 freeze/v4.1 |
