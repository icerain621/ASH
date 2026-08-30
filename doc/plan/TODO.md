# ASH 待办 / 技术债（短清单）

> 更新：2026-08-30  
> **完整计划与设计完成度**见 [`PLAN-进度与里程碑.md`](PLAN-进度与里程碑.md)。  
> 归属：[`plan/`](README.md)  
> 完成项请写入 `CHANGELOG.md` 并从本文件删除；历史 Sprint AY–CE 细节以 CHANGELOG 为准。

## 基线

| 项 | 值 |
|----|-----|
| Tag | `v0.1.0-mvp` |
| Doctor | ALL **53/53** · M3 11/11 · M4 6/6 · M5 4/4 · TR3 10/10 |
| Schema | SQL rev **27**（+run sub-run lineage）· RLS **46** |
| 结论 | v1 自动化门禁达 MVP 可发布水位；**v2 功能 Sprint DH–DV 已收口**（tag `v2.0.0` 待人工）；**v2.1 开篇 DW（ACP）✅** |

---

## v2 — 双核心 / Harness（现行开发）

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| DH | Harness 骨架 | Schema + 000021 + API + `make harness-smoke` | ✅ |
| DI | Loop Adapter | `harness.*` 事件 + NoopRouter stub + HAR-02 | ✅ |
| DX | Sandbox POC | process/Docker + `sandbox-smoke` + danger/`off` 拒绝 | ✅ |
| DY | 演进平面 | feedback 全类型 + `/reviews/queue` + decide | ✅ |
| DZ | 编排 UI / promote 硬化 | Reviews 页 + promote 闸门 + scenario patch + rollback | ✅ |
| DJ | Goal→Run + Plan | from-goal + approve + `ash quest` + Runs Quest UI | ✅ |
| DK | Quest 工作台 v1 | 看板 + 深 Diff 批注 + 步骤评分 + contextRefs | ✅ |
| DL | Repo Profile + Wiki | 即时 Profile/Wiki API + `/ui/knowledge` + contextRefs（无新表） | ✅ |
| DM | Space Rules + 路由 | DB + `.ash/rules.yaml` 双向同步 + from-goal 注入 | ✅ |
| DN | Skills 目录 | SKILL.md 扫描 + 场景绑定 + `skill:` contextRefs | ✅ |
| DO | Sub-run | spawn + 深度/白名单 + `/tree` + Quest 树 | ✅ |
| DP | verify 步骤 | `kind: verify` + 重试 + improve 草稿 | ✅ |
| DQ | Provider + ExecGo live | harness provider→agent + fallback 事件 + `/providers/agent` | ✅ 代码就绪；H-06 live 脚本就绪 |
| DX2 | isolated 强制 | hotfix/security 地板 + KPI-19 + `sandbox.minMode` | ✅ |
| DR | compaction + Doctor M4/M5 | spill + compaction 事件 + M4/M5 探针；ALL 53 | ✅ |
| DS | Webhook 触发 CI→Run | HMAC + diagnose + optional autoRun hotfix；清单 `ci-webhook.md` | ✅ |
| DT | Session RPC/JSON | `/agents/sessions` + turns/events + `ash session rpc` | ✅ |
| DU | 演进 KPI + 移动审阅 | KPI-17/18 + Metrics 演进区 + `/ui/m/reviews` | ✅ |
| DV | v2 scope 冻结 + 签字 | `v2-release-scope.md` + `make v2-signoff`；tag 人工 | ✅ |
| DW | ACP provider 骨架（v2.1） | `ACPExecutor` + Probe + `/providers/agent.acp` + fallback | ✅ |
| 详排 | — | [`v2-dual-core-evolution-plan.md`](v2-dual-core-evolution-plan.md) · v2 功能收口；v2.1 见 [`v2.1-release-scope.md`](v2.1-release-scope.md) | — |

---

## P0 — 发布闭环（需环境 / 人工）

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| H-01 | 云 RDS `make cloud-acceptance` | migrate v20 + Doctor M3/ALL | ⏸ 待 `cloud-rds.env`（本地 `postgres-local-rds-e2e` ✅ Sprint CW，含 M3-04） |
| H-02 | 云 RLS + `ash_app` | M3-06/07；本地 `postgres-app-gate` ✅ | ⏸ 云待签（本地已复验） |
| H-03 | 生产 Worker `ASH_DATABASE_APP_URL` + RLS | `/readyz` dialect=postgres | ⏸ 云待签（本地已复验） |
| BI-5 | 切换日备份→停 SQLite→migrate→t0 | runbook §4 证据 | ⏸ 生产日待执行 |
| S-11 | 四人**真人**签字（替换占位） | `signoff.env` + `signoff-gate` | ⚠️ dry-run 已过 |
| GIT | 远端 `main` 同步 / CI 绿确认 | Actions | ⚠️ 按仓库状态 |

```bash
cp config/cloud-rds.env.example config/cloud-rds.env
bash scripts/source-cloud-rds-env.sh
make cloud-acceptance
make mvp-signoff
```

---

## P1 — 生产可信度

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| H-04/05 | 真实 GitHub CI sync + diagnose | `ASH_CI_LIVE=1 make ci-live-smoke`；禁 fixture；见 [`p1-live-credibility.md`](../checklists/p1-live-credibility.md) | ✅ 脚本就绪；跑通仍需 token/Worker |
| H-06 | ExecGo live | `ASH_EXECGO_E2E=1 make execgo-live-smoke`（证据 + 可选 `/providers/agent`） | ✅ 脚本就绪；跑通仍需 ExecGo/Codex |
| BJ-3 | T+1 生产 KPI 对账 | vs `t1-metrics-gate` 偏差 &lt;5% | ⏸ 需上线后 |
| PRD-8 | 数据分级与保留期 | 附录 J + `data-policy` + retention apply | ✅ Sprint CF |
| R-08 | 发布窗口跨 space / RLS 抽测 | 无越权 | ✅ Sprint CP：`make r08-cross-space-gate`（云 RLS live 随 H-01） |

本地不依赖云的替代门禁：

```bash
make smoke-static
make postgres-app-gate
make postgres-local-rds-e2e   # sqlite→Postgres + M3-04（需 Docker + .ash/ash.db）
make worker-local-gate
make regression-short && make web-gate
```

---

## P2 — 硬化（可代码 / 文档，范围解冻后）

| 域 | 项 | 备注 |
|----|-----|------|
| 附录 F | ~~Artifacts 默认路径与权限~~ | ✅ Sprint CG：`paths.go` + storage/profile |
| 附录 F | ~~Canonical JSON~~ | ✅ Sprint CG：`canonical.go` |
| ARCH | ~~插件打包/签名策略~~ | ✅ CH + CR：`plugin-sign` / SOP / `plugin-sign-smoke` |
| 场景 | ~~Hotfix / Security 模板深化~~ | ✅ Sprint CH：v1.1.0 + human approve |
| 前端 | ~~SSE 浏览器级 E2E~~ | ✅ Sprint CM：`make sse-browser-e2e` |
| 度量 | ~~Agent 稳定性度量（R-02）~~ | ✅ Sprint CN：KPI-11 + `scenarioStability` |
| 产品 | ~~付费/决策/组织样板~~ | ✅ Sprint CO：`org-templates` + ORG 文档 |

---

## P3 — Backlog（明确不做进 v0.1）

- 向量库 / 符号索引 Hybrid RAG  
- 完整沙箱隔离、外部 IdP 联邦、计费  
- 多端网关 / Skill Marketplace / 知识图谱  

---

## 代码维护约定（持续）

- 新表 DDL 必须同步 RLS（`000013` 租户 / run / memory / org）+ Doctor **M3-11**  
- `PostgresRLSDeferredTables()` 应保持为空  
- 公开 API 变更：handler 注释 → `make swagger` → OpenAPI → `make openapi-check`

---

## 已完成归档（摘要）

- Sprint DK：Quest 工作台（看板 / Diff 行批注 / 步骤评分 / contextRefs；SQL 25 / RLS 45）
- Sprint DL：Repo Profile + Wiki（方案 B 即时投影、无新表；`/repos/profile` `/wiki/pages*`；Run `knowledge.injected`；`/ui/knowledge`）
- Sprint DM：Space Rules（方案 C：DB + `.ash/rules.yaml` 双向同步；SQL 26 / RLS 46；Goal 路由；Space 页面板）
- Sprint DN：Skills 目录（SKILL.md 扫描 / 场景绑定 / skill: contextRefs；无新表）
- Sprint DO：Sub-run（谱系 SQL 27；spawn/tree；深度与工具白名单；Quest 树）
- Sprint DP：verify 步骤（DSL kind:verify + 重试 + improve 草稿；无新表）
- Sprint DQ：Provider + ExecGo（harness kind→agent；fallback 事件；`/providers/agent`；H-06 脚本就绪）
- Sprint DX2：isolated 强制（policy/scenario 地板 + KPI-19；无新表）
- Sprint DR：compaction + Doctor M4/M5（spill / harness.compaction；ALL 53）
- Sprint DS：Webhook CI→Run（HMAC `/webhooks/github` + diagnose + autoRun hotfix；无新表）
- Sprint DT：Session RPC/JSON（`/agents/sessions` + turns/events + `ash session rpc`；无新表）
- Sprint DU：演进 KPI + 移动审阅（KPI-17/18；Metrics 演进区；`/ui/m/reviews`；无新表）
- Sprint DV：v2 scope 冻结（`v2-release-scope.md`；`make v2-signoff`；tag 人工）
- Sprint DW：ACP provider 骨架（`ACPExecutor` / ProbeACP / `/providers/agent.acp`；无新表）
- Sprint DJ：Goal→Plan→Run（from-goal / approve / `ash quest` / Runs Quest；SQL 24 / RLS 44）
- Sprint DZ：编排评审 UI + promote 闸门 + scenario_patch + rollback（SQL 23 / RLS 43）
- Sprint DY：演进平面（feedback 全类型 + reviews queue/decide；SQL 22）
- Sprint DX：Sandbox POC（Authorize + process/Docker + `sandbox-smoke`）
- Sprint DI：Loop Adapter（`harness.*` 事件 + sandbox stub + HAR-02）
- Sprint DH：Harness Profile 骨架（SQL 21 / RLS 42 / `/harness/profiles*` / `harness-smoke`）
- Sprint CY：收紧 prior mvp-signoff；失败不覆盖 latest；默认本地 H-01 dry-run；修复 rollback-drill 环境泄漏与清单死链
- Sprint CX：doctor 脚本统一 `--agent static`；smoke-index/H-01 文档对齐 `postgres-local-rds-e2e`；Runs 门禁 / Space 组织样板 vitest
- Sprint CW：`make postgres-local-rds-e2e`（Docker 模拟云 migrate+M3-04）；修复 ALL 套件重复 M3-04 行数漂移；归档 release-window 证据
- Sprint CV：启动 Docker 复验 `postgres-app-gate` + `local-readiness-gate`；doctor 阶段取消误开的 `ASH_MIGRATE_E2E`（避免无 SQLite 的 M3-04 假红）
- Sprint CU：TR0 固定探测语料 `ash.doctor.probe/v1`；SpacePage `listOrgTemplates` mock；memory scope AbsRepoRoot 对齐
- Sprint CT：断开 `config→memory` 导入环（恢复 `smoke-static`）；PLAN RACI/Artifacts 文档闭环
- Sprint CS：危险操作目录 `GET /tools/risk-catalog` + Runs 门禁文案（citation/tool_risk/human）
- Sprint CR：插件签名轮换 SOP + `ash plugin-sign` + `make plugin-sign-smoke`；proto `RegisterRequest.signature`
- Sprint CQ：KPI-08 SSE 稳定率口径修正（closed/(closed+failed)；中途 poll 失败记 session_failed）
- Sprint CP：R-08 跨 space 发布窗口门禁（扩 `TestCrossSpaceAPIRegression` + retention spaceId 校验 + `r08-cross-space-gate`）
- Sprint CO：商业组织样板（PRD §3：`orgtemplates` + `/org-templates` + Space 页）
- Sprint CN：KPI-11 场景稳定率 + scenarioStability 分解（R-02 / P2-4 后半）
- Sprint CM：SSE 浏览器级 E2E（P2-4 前半：Playwright + `sse-browser-e2e`）
- Sprint CH：插件签名 + Hotfix/Security 场景 v1.1.0（P2-2/P2-3）
- Sprint CG：Artifacts 路径策略 + canonical JSON（附录 F / P2-1）
- Sprint CF：附录 J + data-policy + events/artifacts retention apply（PRD-8 / P1-4）
- 自动化与门禁：T-01..T-15、H-04~H-09 **静态/fixture/本地 live**、Sprint AY–CE（SSE、跨 space、状态机、CI 韧性、rollback drill、evidence-sha、全页 vitest 等）。  

详见 `CHANGELOG.md` `[Unreleased]` 与 `[v0.1.0-mvp]`。
