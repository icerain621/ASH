# 计划归属（plan）

**本目录是排期、范围、风险与待办的唯一现行源。**

| 文件 | 用途 | 更新频率 |
|------|------|----------|
| [`PLAN-进度与里程碑.md`](PLAN-进度与里程碑.md) | 里程碑完成度 + P0–P3 计划 | 里程碑/周会 |
| [`TODO.md`](TODO.md) | 未完成短清单 | 任务完成即删条目 |
| [`mvp-release-scope.md`](mvp-release-scope.md) | MVP 范围冻结（`scope-freeze-gate`） | 冻结变更评审 |
| [`v2-release-scope.md`](v2-release-scope.md) | **v2** 范围冻结（`scope-freeze-gate` / `make v2-signoff`） | v2 GA / DV |
| [`v2.1-release-scope.md`](v2.1-release-scope.md) | **v2.1** 范围冻结（`make v2.1-signoff`） | ACP 桥接 / DX5 |
| [`v2.2-release-scope.md`](v2.2-release-scope.md) | **v2.2** 范围冻结（`make v2.2-signoff`） | Waker / DX8 |
| [`v2.3-release-scope.md`](v2.3-release-scope.md) | **v2.3** 范围冻结（`make v2.3-signoff`） | RAG Hybrid / DX9–DX11 |
| [`v2.4-release-scope.md`](v2.4-release-scope.md) | **v2.4** 范围冻结（`make v2.4-signoff`） | Waker duties / DX12–DX14 |
| [`v2.5-release-scope.md`](v2.5-release-scope.md) | **v2.5** 范围冻结（`make v2.5-signoff`） | Waker probes / ctags / vector / Landlock / DX15–DX19 |
| [`v2.6-release-scope.md`](v2.6-release-scope.md) | **v2.6** 范围冻结（`make v2.6-signoff`） | tree-sitter / 沙箱硬化 / skill packs / DX20–DX24 |
| [`v2.7-release-scope.md`](v2.7-release-scope.md) | **v2.7** 范围冻结（`make v2.7-signoff`） | vector embed / sandbox 证据 / skill catalog / thin LSP / DX25–DX30 |
| [`v2.8-release-scope.md`](v2.8-release-scope.md) | **v2.8** 范围冻结（`make v2.8-signoff`） | 完整 LSP 切片 hover/def/refs / DX31–DX36 |
| [`v2.9-release-scope.md`](v2.9-release-scope.md) | **v2.9** 范围冻结（`make v2.9-signoff`） | 可选远程沙箱 E2B-class / DX37–DX42 |
| [`v3.x-program.md`](v3.x-program.md) | **v3.x** 程序（E1–E7 已确认；分冻 v3.0/3.1/3.2） | 向量 / Quest / IdP·网关 |
| [`v3.0-release-scope.md`](v3.0-release-scope.md) | **v3.0** 范围冻结（`make v3.0-signoff`） | Chroma+Milvus + prefer=vector；保留 Qdrant / DX43–DX48 |
| [`v3.1-release-scope.md`](v3.1-release-scope.md) | **v3.1** 范围（已冻结；DX49–DX54） | Quest 委派面 UX / 硬化 |
| [`v3.2-release-scope.md`](v3.2-release-scope.md) | **v3.2** 范围（已冻结；DX55–DX60） | IdP OIDC / Session 网关 |
| [`v4.x-program.md`](v4.x-program.md) | **v4.x** 程序（F1–F7；分冻 v4.0–v4.3） | Auth / 企业 Agentic / Stage-1 / 生态 |
| [`v4.0-release-scope.md`](v4.0-release-scope.md) | **v4.0** 范围（**已冻结**；DX61–DX66） | Auth 硬化（RS256 / refresh / 吊销 / 门闸） |
| [`v4.1-release-scope.md`](v4.1-release-scope.md) | **v4.1** 范围（**已冻结**；DX67–DX72） | 企业 Agentic（配额 / 审计报表 / spawn） |
| [`v4.2-release-scope.md`](v4.2-release-scope.md) | **v4.2** 范围（**草案**；DX73–DX78） | Stage-1（插件 gRPC 生产路径 / 进程内 Indexer） |
| [`v5-governance-program.md`](v5-governance-program.md) | **v5** 双核管控 × 厚评审 × 薄交互（设计评审中） | 空间管控 / 评分 / Workbench / 薄交互 |
| [v5 实现排期](../../docs/superpowers/plans/2026-09-13-v5-governance-implementation.md) | **v5** 全量改造排期与技术方案（GV01–12） | 可观测运行 / 记忆关联 / 评分管控 |
| [`risk-register.md`](risk-register.md) | 风险台账 | 周会 |
| [`kpi-dashboard-definition.md`](kpi-dashboard-definition.md) | KPI 口径 | 口径变更时 |
| [`qoder-ash-comparison.md`](qoder-ash-comparison.md) | Qoder 与 ASH 竞品/能力比对（调研稿） | 外部产品重大变更或 ASH 范围调整时 |
| [`deepseek-harness-ash-comparison.md`](deepseek-harness-ash-comparison.md) | DeepSeek Harness 与 ASH 能力比对（调研稿） | DSH 重大版本或 ASH Agentic 架构调整时 |
| [`pi-ash-comparison.md`](pi-ash-comparison.md) | Pi 与 ASH 能力比对（调研稿） | Pi 重大版本或 ASH 集成/CLI 调整时 |
| [`platform-quad-comparison.md`](platform-quad-comparison.md) | **四产品综合分析**（DSH·Pi·Codex·Hermes→ASH；v6 输入） | 外部平台大版本或 ASH v6 规格启动时 |
| [`ash-feature-inventory.md`](ash-feature-inventory.md) | 原型对照：功能/接口状态 · 人时 · 交付波次 | 排期或吸收项评审时 |
| [`sprint-ew-w0-harden.md`](sprint-ew-w0-harden.md) | **W0 硬化** Sprint 板（EW01–EW05）✅ | 已完成 |
| [`sprint-ew-w1-v60-absorb.md`](sprint-ew-w1-v60-absorb.md) | **W1 v6.0 吸收** Sprint 板（EW11–EW18） | 开工勾选 |
| [`sprint-ew-w2-v61-absorb.md`](sprint-ew-w2-v61-absorb.md) | **W2 v6.1 吸收** Sprint 板（EW21–EW25） | 开工勾选 |
| [`sprint-ew-w3-v62-absorb.md`](sprint-ew-w3-v62-absorb.md) | **W3** Sprint 板（EW31–EW34） | ✅ |
| [`sprint-ew-w4-p15-loop.md`](sprint-ew-w4-p15-loop.md) | **P15** MCP execute 走 Hooks | ✅ |
| [`sprint-ew-w5-p16-lineage.md`](sprint-ew-w5-p16-lineage.md) | **P16** Trajectory 派生子 Run | ✅ |
| [`sprint-ew-w6-memorylink-deeplink.md`](sprint-ew-w6-memorylink-deeplink.md) | Agent → MemoryLink 深链 | ✅ |
| [`sprint-ew-w7-compact.md`](sprint-ew-w7-compact.md) | **P06** `/compact` 语义 | ✅ |
| [`sprint-ew-w8-settings-ia.md`](sprint-ew-w8-settings-ia.md) | 顶栏「设置」聚合运维 | ✅ |
| [`sprint-ew-w9-model-health.md`](sprint-ew-w9-model-health.md) | Composer Provider 健康位 | ✅ |
| [`sprint-ew-w10-openapi-hygiene.md`](sprint-ew-w10-openapi-hygiene.md) | OpenAPI 去掉 legacy `/v1/*` | ✅ |
| [`sprint-ew-w11-post-tool-use.md`](sprint-ew-w11-post-tool-use.md) | Hooks PostToolUse | ✅ |
| [`../prototypes/agent-absorb/`](../prototypes/agent-absorb/README.md) | 吸收交互原型（Agent 薄壳 + **管控&评审** 工作台） | 对照吸收项或 IA 演进时 |
| [`agentic-roadmap-to-qoder.md`](agentic-roadmap-to-qoder.md) | 面向真实工作的 Agentic 迭代路线图（原则） | 里程碑 M4+ 或 Sprint DA+ 排期时 |
| [`v2-dual-core-evolution-plan.md`](v2-dual-core-evolution-plan.md) | **v2 双核心演进 + v1→v2 开发计划（Sprint DH–DV）** | v2 范围冻结 / M4 启动 |

## 归属边界

- 不写协议细节（→ `appendices/` / `api/`）  
- 不写操作步骤长文（→ `checklists/`）  
- 不存门禁输出（→ `evidence/`）  

真相源优先级：`PLAN` ≫ `TODO` ≫ CHANGELOG（事实）≫ archive（历史）。
