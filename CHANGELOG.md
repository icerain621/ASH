# Changelog

All notable changes to ASH will be documented in this file.

This project follows a Keep a Changelog style. Version numbers can be attached when a release tag is cut.

## [Unreleased]

### Added

- Sprint DX12（v2.4 草案）：Waker duties 账本 — 表 `waker_duties` / `waker_duty_runs`（SQL **29** / RLS **50**）；默认 ensure `stale_run`；ticker 读 ledger 只 report/flag；`GET /waker/status` · duties 列表/手动 run；`make waker-smoke` 扩展；范围草案 `v2.4-release-scope.md`（**未冻结**）；**不自动**打 `v2.4.0` tag。
- Sprint DX9（v2.3 草案）：RAG Hybrid 符号索引（方案 C）— 表 `rag_path_entries` / `rag_symbols`（SQL **28** / RLS **48**）；`RebuildSymbols` + Hybrid Query（path/symbol/text RRF）；`POST /rag/symbols/rebuild`；`make rag-hybrid-smoke`；范围草案 `v2.3-release-scope.md`；**不自动**打 `v2.3.0` tag。DX10 Hybrid 接线与 DX11 v2.3 冻结/签字仍为工作区未提交 WIP，不在本分支交付。
- Sprint DX8（v2.2）：范围冻结 + 签字门禁（方案 C）— `v2.2-release-scope` 已冻结；`make v2.2-signoff`（含 `waker-smoke`）；清单/签字模板；**不自动**打 `v2.2.0` tag。
- Sprint DX7（v2.2）：Waker cancel 安全闸门（方案 C）— `action=cancel` 需 `ASH_WAKER_ALLOW_CANCEL=1` + `confirm=CANCEL_STALE_RUNS`；审计 `waker.cancel_completed`；后台永不 cancel；`WAKER_CANCEL_DENIED`（无新表）。
- Sprint DX6（v2.2）：Waker 雏形（方案 C）— `GET /waker/queue` + `POST /waker/sweep`（dryRun 默认）；`ASH_WAKER_RUN_TTL` / `ASH_WAKER_INTERVAL`；审计 `waker.sweep_completed`；`make waker-smoke`；范围草案 `v2.2-release-scope.md`（无新表）。
- Sprint DX5（v2.1）：范围冻结 + ACP Doctor（方案 C）— M4-ACP-01/02；Doctor ALL **55**/M4 **8**；`v2.1-release-scope` 已冻结；`make v2.1-signoff`；清单/签字模板；**不自动**打 `v2.1.0` tag。
- Sprint DX4（v2.1）：ACP 任务契约硬化（方案 C）— `ash.acp.task.v1` 出入站校验；`cmd/acp-mock` + `make acp-smoke`；Session turn best-effort 转发 ACP；schema JSON + 清单 `acp-smoke.md`（无新表）。
- Sprint DX3（v2.1）：ACP ↔ Session 互通（方案 C）— Session `providerKind` 探测字段；Run 在 `acp_sdk` 选型时 `EnsureForRun` + 事件 `session.linked`；AgentTask/ACP 使用 `sess_*`；清单 `acp-session.md`（无新表）。
- Sprint DW（v2.1）：ACP provider 骨架（方案 C）— `ACPExecutor` + `ProbeACP`；Harness `acp_sdk` 按端点探测，失败回退 static；`GET /providers/agent` 增加 `acp`/`acpE2EEnabled`；清单 `acp-provider.md`；范围草案 `v2.1-release-scope.md`（无新表）。
- P1 可信度（方案 C）：真 GitHub CI live（`make ci-live-smoke`，`ASH_CI_LIVE=1`，禁 fixture）+ ExecGo live 证据/providers 探测；编排 `make p1-live-credibility`；清单 [`p1-live-credibility.md`](doc/checklists/p1-live-credibility.md)。
- Sprint DV（v2）：scope 冻结 + 签字（方案 C）— `doc/plan/v2-release-scope.md`；`scope-freeze-gate` 校验 MVP+v2；`make v2-signoff`；清单/签字模板；**不自动**打 `v2.0.0` tag。
- Sprint DU（v2）：演进 KPI + 移动审阅（方案 C）— Metrics **KPI-17**（编排评审积压）/ **KPI-18**（Improve 回滚率）与 17–19 演进区；`/ui/m/reviews` 精简批准/拒绝；清单 `evolve-kpi-mobile.md`（无新表）。
- Sprint DT（v2）：Session RPC/JSON（方案 C）— `POST/GET /api/v1/agents/sessions`、turns、events/`streamUrl`；CLI `ash session rpc`（LF-JSON `session.start`/`turn.prompt`）；会话落 `audit_log`（无新表）。
- Sprint DS（v2）：Webhook 触发 CI→Run（方案 C）— `POST /api/v1/webhooks/github` HMAC；失败 `workflow_run` → upsert CI + Diagnose；`autoRun=1` 创建 hotfix Run；`X-GitHub-Delivery` 幂等；清单 `ci-webhook.md`（无新表）。
- Sprint DR（v2）：compaction + Doctor M4/M5（方案 C）— 工具大输出 spill + `harness.compaction`；Doctor **M4**（HAR/SBX）与 **M5**（EVO）；canary 上限 10%；ALL **53/53**（无新表）。
- Sprint DX2（v2）：isolated 强制（方案 C）— hotfix/security 与 `sandbox.minMode` 地板；danger &lt; isolated 拒绝；KPI-19 danger 沙盒覆盖率；清单 `sandbox-isolated.md`（无新表）。
- Sprint DQ（v2）：Provider + ExecGo live（方案 C）— Harness `provider.kind` → agent 选型；ExecGo 探测失败回退 static 并发 `provider.fallback`；`GET /api/v1/providers/agent`；平台默认 provider=`execgo`；H-06 代码就绪（真实 live 仍需 `ASH_EXECGO_E2E=1`）。
- Sprint DP（v2）：verify 步骤（方案 C）— DSL `kind: verify` + `verify.checks`/`onFail`；执行重试；失败可自动 improve draft（`improve.draft_created`）；`feature_delivery` qa.verify 升级（无新表）。
- Sprint DO（v2）：Sub-run（方案 C）— runs 谱系字段（SQL **27**）；`POST /runs/{id}/sub-runs` + 深度/工具白名单；`GET /runs/{id}/tree`；`run.spawned`；Quest Sub-run 树。
- Sprint DN（v2）：Skills 目录（方案 C）— `.ash/skills/*/SKILL.md` 扫描；`GET /skills`；场景 `skills:` + Harness `skills[]` 绑定；Run `skills.injected` / `skill:` contextRefs；Automation 页 Skills 面板（无新表）。
- Sprint DM（v2）：Space Rules（方案 C）— 表 `space_rules`（SQL **26** / RLS **46**）；`GET/PUT /spaces/{id}/rules` + import/export/preview；`from-goal` 注入；Space 页 Rules 面板；与 `.ash/rules.yaml` 双向同步。
- Sprint DL（v2）：Repo Profile + Wiki（方案 B）— `GET /repos/profile`、`/wiki/pages*` 即时投影（无新表）；Run `prepareExecutionContext` 注入 `profile:`/`wiki:`；控制台 `/ui/knowledge`。
- Sprint DK（v2）：Quest 工作台 — `/ui/quest` 看板、unified Diff 行级批注、步骤评分；manifest `contextRefs`；表 `diff_review_comments`（SQL **25** / RLS **45**）。
- Sprint DJ（v2）：Goal→Plan→Run — `POST /runs/from-goal`、Plan approve/reject、`ash quest`、Runs 页 Quest 表单；表 `goal_plans`（SQL **24** / RLS **44**）。
- Sprint DZ（v2）：编排评审 UI + promote 硬化 — promote 仅 `in_review`；`POST .../rollback`；`scenario_patch_drafts`（SQL **23** / RLS **43**）+ `/scenario-patches*`；Reviews 页双队列 + Scenario patch 草稿；Automation Harness Profile 面板；Feedback targetType 对齐附录 K。
- Sprint DY（v2）：演进平面基础 — feedback `targetType` 白名单（含 `harness_profile` 等）+ `runId` 去重；`GET /api/v1/reviews/queue`；`POST /api/v1/reviews/{id}/decide`（memory / harness）；SQL rev **22**（feedback.run_id）。
- Sprint DX（v2）：Sandbox POC — `Authorize`（danger+`off` 拒绝）、`DefaultRouter`、process/Docker Executor、`deploy/sandbox/Dockerfile`、`make sandbox-smoke`（`ASH_SKIP_SANDBOX=1` 可跳过 Docker）；risk-catalog 增加 `minSandboxMode`。
- Sprint DI（v2）：Harness Loop Adapter — `internal/harness/loop` 钩子 + `harness.*` 事件；`internal/sandbox` NoopRouter/`ResolveSandboxMode` stub；runs `callTool*` 接线；HAR-02 不变量包测。
- Sprint DH（v2）：Harness Profile 骨架 — Schema `ash.harness.profile.v1`、表 `harness_profile_versions`（SQL rev **21** / RLS **42**）、`internal/harness` CRUD/LoadActive/promote、API `/api/v1/harness/profiles*`、`make harness-smoke`。
- Sprint CY：`release-window` prior mvp-signoff 须 web-gate 绿且无 ❌；失败不覆盖 `mvp-signoff-latest`；mvp-signoff 默认本地 H-01 dry-run（`ASH_MVP_SKIP_LOCAL_RDS=1` 可跳过）；修复 `ASH_REQUIRE_ROLLBACK_DRILL` 泄漏导致 regression-short 假红；修复清单死链与 RunsPage tsc。
- Sprint CX：本地 doctor 脚本统一 `--agent static`；smoke-index / H-01 清单对齐 `postgres-local-rds-e2e`；Runs 危险工具门禁与 Space 组织样板 vitest。
- Sprint CW：`make postgres-local-rds-e2e`（Docker 本地模拟云 RDS migrate）；`postgres-rds-e2e` 在 M3-04 通过后取消 `ASH_MIGRATE_E2E`，避免 ALL 套件行数漂移假红；归档 2026-08-08 发布窗口证据。
- Sprint CV：Docker 复验 `postgres-app-gate` / `local-readiness-gate`；`postgres-app-gate` doctor 阶段不再误开 `ASH_MIGRATE_E2E`（M3-04 假红）。
- Sprint CU：Doctor TR0 固定探测语料（`ash.doctor.probe/v1` + embed）；关闭附录 E §3.3 TODO；SpacePage vitest 补齐 org-templates mock；运行时 memory 检索对齐 AbsRepoRoot。
- Sprint CT：`config.LoadDataPolicy` 不再依赖 `memory`，消除测试导入环并恢复 `make smoke-static`；hotfix/security_patch@1.1.0 场景测试走人工门禁；memory 注入测试对齐 AbsRepoRoot；PLAN 补齐 Epic owner RACI。
- Sprint CS：内置工具风险目录（`GET /api/v1/tools/risk-catalog` + 自动化页）；Runs 门禁文案区分引用 / 危险工具 / 人工步骤；关闭 ARCH 危险操作产品化 TODO。
- Sprint CR：插件签名生产轮换 SOP；`ash plugin-sign` CLI；`make plugin-sign-smoke`；proto `RegisterRequest.signature` 与 gRPC 注册接线；关闭 ARCH §3.2 签名 TODO。
- Sprint CQ（R-07 / KPI-08）：SSE 稳定率改为 `closed/(closed+failed)`；stream 中途 poll 失败记 `session_failed` 且不再记成功关闭；补 KPI 口径与单测。
- Sprint CP（P1-5 / R-08）：扩 `TestCrossSpaceAPIRegression`（RAG/feedback/retention/space 矩阵等）；events/artifacts retention 校验可选 `spaceId`；`make r08-cross-space-gate` 接入 `release-window-gate`；清单 `doc/checklists/r08-cross-space.md`。
- Sprint CO（P2-5 / PRD §3）：三套组织样板（小团队/中大型/强合规）与付费·决策·审批约定；`GET/POST /api/v1/org-templates…` 一键开通；Space 控制台样板卡片；设计文档 `ORG-组织样板与商业落地.md`。
- Sprint CN（P2-4 / R-02）：metrics overview 增加 KPI-11 场景稳定率与 `scenarioStability` 分解；Metrics 看板高亮低于 85% 门槛的场景；风险册 R-02 → Mitigating（度量侧）。
- Sprint CM（P2-4 SSE）：Playwright 浏览器级 SSE 烟测（`frontend/e2e/sse-run-stream.spec.ts` + `make sse-browser-e2e`）；Runs 页 `data-testid` 便于断言「已连接」与事件行。
- Sprint CH（P2-2/P2-3）：插件注册 HMAC 签名（`ASH_PLUGIN_SIGNING_KEY` / `signature` / `ash.sign.hmac=`）；`plugins/abi` 暴露签名策略；hotfix/security_patch 场景升至 v1.1.0（blast radius / risk matrix / human approve）；附录 H §6。
- Sprint CG（P2-1）：Artifacts 跨平台路径策略（`ASH_RUNS_DIR` / `EnsureRunLayout` / `storage/profile.artifactPaths`）与 canonical JSON digest（`MarshalCanonicalJSON`）；附录 F TODO 关闭。
- Sprint CF（P1-4）：附录 J 数据分级与保留期；`internal/config` data policy；`GET /api/v1/data-policy`；`POST /api/v1/events/retention/apply` 与 `/artifacts/retention/apply`；Scale/readyz 暴露保留默认值；关闭 PRD §8 TODO。

### Changed

- 文档归属重组：`doc/design/`（设计）、`doc/plan/`（计划/TODO/范围/风险/KPI）、`doc/progress/`（发布清单）；`checklists/`/`evidence/`/`api/`/`appendices/` 路径保持稳定；早期草案与 MySQL DDL 在 `doc/archive/`。索引见 `doc/README.md`。

### Added

- Sprint CE：Observability 治理面板展示 `run_inflight_count`；`TestCrossSpaceAPIRegression` 扩 secret/memory/CI/audit/plugin/artifact·checkpoint access；CI diagnosis adopt/dismiss 先按 ID 鉴权再决策（跨 space 403）；`evidence-sha-gate`（默认 WARN，`ASH_REQUIRE_EVIDENCE_SHA=1` 硬失败）接入 release-window / mvp-signoff / `regression-short` smoke（R-06/R-08）。
- fix: memory query 空间隔离单测补齐 `confidence>=0.2`（BR 检索门槛回归）。
- CI：`release-window` / `mvp-signoff` 设置 `ASH_ROLLBACK_DRILL_SKIP_DOCTOR=1`，避免与 Backend Doctor 重复跑 `TestALLSuite`。
- Sprint CD：本地 Docker 复验 release-window / postgres-app-gate / cloud-acceptance；证据刷新至当前 HEAD（R-12/BI）。
- Sprint CC：mvp-signoff 默认 `ASH_RELEASE_AUDIT_SKIP_DOCTOR/REGRESSION=1` + drill 跳过 Doctor ALL；`TestCrossSpaceAPIRegression` 覆盖 runs 控制面 cancel/resume/replay/approve（R-08）。
- fix: `data-backup` 校验和改为 basename，verify 用绝对路径比对 sha256（修复相对路径下 `pre-migrate-gate` 失败）；`ash-data-backup-smoke` 接入 `regression-short`。
- Sprint CB：CIPage 同步失败展示 `CI_PROVIDER_UNAVAILABLE` / `CI_*_LIST_FAILED`；控制面负例覆盖 Cancel 后 Resume/Approve（R-09/R-05）。
- Sprint CA：发布窗口 / mvp-signoff 默认 `ASH_REQUIRE_ROLLBACK_DRILL=1`；告警指标 `run_inflight_count`（阈值 20）与 Prometheus `ash_run_inflight_live`（R-12/R-06）。
- Sprint BZ：发布窗口强制 `rollback-drill`（含 `ASH_ROLLBACK_DRILL_MAX_MS` SLA）；`data-backup` 后 sha256 + SQLite `integrity_check`；`EvaluateGate` 纳入 rollback drill（缺省 warn，`ASH_REQUIRE_ROLLBACK_DRILL=1` 硬阻断）；触发准则写入证据报告（R-12）。
- Sprint BY：`run.canceled` 进入 derive/`ash_run_inflight` 对账；Scale 暴露 `runRunningCount` / `runWaitingApprovalCount` / `runInflightCount`；OpenAPI 控制面 409；RunsPage 展示 `RUN_NOT_*` 等错误码；控制面负例 API 单测；Scale/Runs 前端 smoke；vitest `afterEach(cleanup)` 避免多挂载串扰（R-06/R-10）。`make swagger` 代理不可达时自动镜像重试。
- Sprint BX：GitHub provider 对 429/5xx/瞬时错误退避重试；连续失败熔断；API `CI_PROVIDER_UNAVAILABLE`（R-09）。
- Sprint BW：`canApprove` / `canResume`；`ErrRunNotApprovable`；Approve 刷新后再转换以防 Cancel 竞态；API `RUN_NOT_APPROVABLE` / `ILLEGAL_STATUS_TRANSITION` / `RUN_CANCELED`（R-05）。
- Sprint BV：Replay 仅允许终态源（finished/failed/canceled）；`ErrRunNotReplayable` / API `RUN_NOT_REPLAYABLE`（R-05）。
- Sprint BU：`observeCanceled` 在步边界与 agent 完成后轮询；Create/Resume 将中途 Cancel 视为终态；`TestMidLoopCancelStopsWithoutFinish`（R-05）。
- Sprint BT：run 生命周期状态机（`canTransition` / `trySetRunStatus`）；`failRun`/finish/waiting_approval 尊重 Cancel；Cancel/Resume/Approve 走统一转换；表驱动与幂等单测（R-05）。
- Sprint BS：`regression-short` 纳入 BQ/BR 单测；CI fixture 增加 cancel/OOM/frontend job（9103–9105）；RunsPage SSE「重连中 / 轮询回退」smoke（R-03/R-07）。
- Sprint BR：`ApplyFeedbackDecay`（rating≤2 → confidence −0.15）经 `POST /feedback`（`memory`/`memory_hit`）接线；audit/event `memory.confidence_adjusted`；Query 与 run 检索按 confidence 排序并排除 `<0.2`（R-04）。
- Sprint BQ：`postgresMigrationDSN` 回归单测（open-time owner DSN 优先于 `ASH_DATABASE_APP_URL`）；CI 诊断规则扩展 `actions_cancel_or_runner_abort` / `runner_resource_exhaustion` / `frontend_lint_or_typecheck_failure`（R-03）。
- Sprint BP：跨平台 `testutil.WriteFakeExecGoCLI`；`OpenTest` Windows SQLite 清理；CI/Postgres E2E 失败日志 artifact + docker 诊断；补齐 apicodes 缺码；RLS tester 角色改为幂等 ensure；`migrate()` 使用打开时的 owner DSN；postgres-e2e RLS 段跳过重复 M3-04（避免种子漂移）。
- Sprint BO：`regression-short` / H-09 static 接入 `TestCrossSpaceAPIRegression` 与 `TestStreamRunResumesFromQueryLastEventID`。
- Sprint BN：SSE 重连耗尽后回退 timeline 轮询（R-07）；`TestStreamRunResumesFromQueryLastEventID` 校验 query 续传。
- Sprint BM：前端 SSE 自动重连（指数退避 + `Last-Event-ID` 续传）；`useRunStream` 暴露连接状态；`TestCrossSpaceAPIRegression` 覆盖 stream/provenance/artifacts/approvals/releases/repo（R-07/R-08）。
- Sprint BL（续）：**12/12** 控制台页 vitest smoke（含 Space/Automation/Compliance）；Automation/Compliance 嵌套字段可选链防护；`signoff-apply` 幂等重跑无 WARN。
- Sprint BL：`make worker-local-gate` 本地演练（BI-5 §4 第 5 步）；`signoff-apply` 同步 `release-window-runbook.md` §1/§2/§7；Observability 页 vitest smoke。
- Sprint BI（续）：证据刷新（2026-07-07）；`cloud-acceptance-gate` 区分本地 Docker / 云 RDS；`release-window-latest` 更新。
- `scripts/postgres-truncate-dev-data.sql` 本地 e2e 漂移清理；`postgres-rds-e2e` / `postgres-smoke` / Doctor CLI 加固。

### Changed

- SSE `/api/v1/runs/{runId}/stream` 同时接受 header / query `Last-Event-ID`（及 `lastEventId`），便于前端手动重连续传。

### Fixed

- `schema_meta` migrate verify 允许 Postgres 额外 `sql_migrations` 键。
- Doctor M3-01 / CLI 在 Postgres+RLS 下验收（owner URL、`row_security=off`、测试行清理）。
- `OpenTest` 与 `release-sampling-static` 隔离云环境变量，H-09 API 烟测稳定。

## [v0.1.0-mvp] - 2026-07-05

### Added

- Sprint BH：前端 Runs/CI/Memory/Scale 页 vitest smoke；`ci.yml` 并行 `release-window-gate`；`make kpi-reconcile-gate`（KPI §9）。
- Sprint BG：`make signoff-apply` / `make signoff-gate`；`config/signoff.env.example` + `mvp-signoff-roster.md`；§11/范围冻结回填（占位 dry-run）。
- Sprint BF：`make local-readiness-gate`；`web-gate.sh` 禁用 audit + `npm … | cat`（嵌套 evidence 规避）；`release-window-gate` 默认快速路径。
- Sprint BE：`make release-window-gate` / `make bootstrap-local-ash-db`；`release-gates.yml` 静态 release-window + `worker-production-gate`；MVP §6 备份 / §8 自动化勾选。
- Sprint BD：`make config-env-gate` / `make worker-production-gate` / `make release-window-prefill`；`scripts/sse-live-smoke.sh`（H-09 §7.2 SSE）；`mvp-signoff` + CI 接入 config-env-gate；MVP §8 配置核对 / §10 T+1 勾选。
- Sprint BC：`make scope-freeze-gate`；`release-gates.yml`（`worker-local-gate` + `mvp-signoff`）；`cloud-acceptance` 接入 `pre-migrate-gate`。
- Sprint BB：`make worker-local-gate`（临时 Worker + live-smoke）；`make pre-migrate-gate` / `make t1-metrics-gate`；`release-window-runbook.md`。
- Sprint BA：`make queue-gate` / `make t0-alert-gate` / `make data-backup`；`TestTTLQueueConsumeBaseline`；`TestEvaluateCleanSpaceNoCriticalAlerts`；`doc/mvp-release-scope.md`。
- Sprint AZ：`make production-config-gate` / `make rollback-drill`；`TestHealthEndpointsLatencyBaseline` / `TestConcurrentRunsListBaseline`；`internal/config/production_guard.go`。
- Sprint AY：`scripts/web-gate.sh` + `make web-gate`（前端 eslint/vitest/build）；CI `frontend` job 升级；`scripts/source-cloud-rds-env.sh`。
- Sprint AX：`make cloud-acceptance` / `make mvp-signoff` 证据门禁；`doc/checklists/h01-h03-cloud-signoff.md`；`doc/evidence/` 签字摘要；MVP 清单自动化勾选映射。
- Sprint AW：`scripts/postgres-app-gate.sh` + `make postgres-app-gate`（H-02/H-03 本地 ash_app/RLS）；`scripts/smoke-static.sh`；`postgres-app-gate.md`。
- Sprint AV：`postgres-rds-e2e` 接入 `live-smoke`；`smoke-index` 文档串联；MVP 清单与 CI `release-sampling-static` 显式步骤。
- Sprint AU：`scripts/live-smoke.sh` + `make live-smoke`（H-04/05/06/07/09 live 编排）；`doc/checklists/smoke-index.md`；`release-window-audit` live 段收敛。
- Sprint AT：`scripts/release-sampling-static.sh` + `make release-sampling-static`（H-09 §7 静态烟测）；`release-sampling-smoke.sh`；`doc/checklists/release-sampling-smoke.md`；`regression-short` / `release-window-audit` 接入。
- Sprint AS：`scripts/secret-rotate-smoke.sh` + `make secret-rotate-smoke`（H-07：`TestSecretRotateRepoConnectionH07` + live fixture 轮换后 CI sync）；CI sync upsert 保留 run/job ID；`scripts/regression-short.sh`；`make release-sampling`；CI `regression-short` 扩展。
- Sprint AR：`scripts/execgo-live-smoke.sh` + `make execgo-live-smoke`（H-06：`execgo-health` + `ASH_EXECGO_E2E=1` Doctor M3-05）；`TestM3ExecGoLiveSmoke`；`verify-local` 静态门禁。
- Sprint AQ：`make release-window-audit` 聚合 H-08 静态门禁（Doctor ALL/M3/TR3、regression-short、openapi-check、API 抽样）。
- Sprint AP：`scripts/ci-fixture-smoke.sh`（live Worker + `ASH_CI_FIXTURE=1`）；`release-sampling.sh` ttl-queue §7.3b；CI 页 fixture 模式提示。
- Sprint AO：CI fixture H-04/H-05 全链路（双 job 日志、仅 `jobId` 诊断、`logDigest` 落库、adopt）；`TestReleaseSamplingCIFixtureH04H05`。
- Sprint AN：OpenAPI 契约补全 TTL 端点（`GET /memory/ttl-queue`、`POST /memory/ttl-sweep`）与 `memoryTTLSweepInterval`；API/derive 单测；H-09 抽样含 ttl-queue；`make swagger` 再生。
- Sprint AM：Worker 后台记忆 TTL sweep（`ASH_MEMORY_TTL_SWEEP_INTERVAL`，最短 1m）；`/readyz` 与 Scale 暴露 `memoryTTLSweepInterval`。
- Sprint AL（P4+）：记忆 TTL 复核队列（`GET /memory/ttl-queue`）+ 到期 sweep（`POST /memory/ttl-sweep`）；`memory.ttl_expired` derive；Doctor **TR1-06**；Scale TTL 指标与 sweep 按钮；ALL **43/43**。
- Sprint AJ（P3）：记忆 catalog **v1→v2**（L1=90d / L2=365d 默认 TTL）；`RunMigrations` 多步升至 `CurrentSchemaVersion=2`；H-08 `release-window-audit.md`；H-09 `TestReleaseSamplingH09` + `scripts/release-sampling.sh`；`postgres-rds-e2e` 可选 Worker 抽样。
- Sprint AH：`ASH_CI_FIXTURE=1` CI sync fixture（runs/jobs/logs 全链路单测）；`/readyz` `liveGateHints`（M3-04..08）；Scale 面板；`doc/checklists/execgo-live-smoke.md`；`verify-local` 可选 `execgo-health`；生产配置密钥轮换 SOP 摘要。

### Fixed

- `release-sampling.sh` 创建 Run 使用 `issueOrSpec`（修复 live H-09 烟测）。
- `ci-fixture-smoke.sh` 按 `fixture-job-9101` 选取 jobId（修复 H-05 诊断 rootCause）。

- Sprint AD：Scale 页 **Worker /readyz** 面板（与 Scale 一致性对照）；`TestPostgresReadyzWithRLS` 并入 `postgres-sql-schema-e2e`；`ci.yml` PR 跑 `postgres-rls-e2e`；RDS 脚本补 `doctor TR3`。
- Sprint AC: Doctor **TR3-10** `/readyz` HealthResponse 契约（`openapicheck.ValidateReadyzContract`）；M3-09 校验 `sqlMigrationExpected` 与嵌入修订一致；`postgres-rds-e2e.sh` 同步 rev **20**；ALL **42/42**。
- Sprint AB: `/readyz` 暴露 RLS/SQL 漂移字段（`postgresRLSPolicyExpected`、`rlsCatalogSummary`、`readinessWarnings`）；Doctor **M3-09** 补 `rlsCatalog`/`rlsPolicies`/`sqlExpected` 证据；`postgres-e2e.yml` 新增 `postgres-rls-e2e` job。
- Sprint AA: Scale readiness `postgresRLSPolicyExpected` / `rlsCatalogSummary` + RLS/SQL 漂移 `readinessWarnings`；`regression-short` RLS catalog 冒烟；`doc/checklists/postgres-rls-new-table.md`；RDS 清单同步 rev 20 / 41 policies。
- Sprint Z: SQL rev **20** org identity RLS (`users`/`orgs`/`roles`/`members` via `app.ash_org_id`); Worker RLS middleware resolves `spaces.org_id`; policy count **41**; `PostgresRLSDeferredTables()` empty.
- Sprint Y: Postgres integration `TestPostgresRLSSpaceIsolationOnMemoryChildren`（rev 19）；`postgres-rls-e2e` / `postgres-sql-schema-e2e` 接入 RLS 子表校验；附录 C memory 子表 RLS 说明。
- Sprint X: SQL rev **19** memory-scoped RLS (`memory_evidence`/`memory_reviews` via `ash_rls_memory_visible`); `PostgresRLSMemoryScopedTables()`; policy count **37**; deferred catalog reduced to org identity tables.
- Sprint W: SQL rev **18** run-scoped RLS for `model_usage`; `PostgresRLSDeferredTables()` + `VerifyRLSMigrationSQL()`; Doctor **M3-11** migration/RLS catalog parity; appendix G §9 OpenAPI/legacy `/v1/*`.
- Sprint V: Doctor **TR3-09** OpenAPI contract alignment (`openapicheck.ValidateContract`); `make openapi-check` runs full `internal/openapicheck` suite (path + schema + envelope); `regression-short` adds contract smoke tests.
- OpenAPI schemas for tool calls, agent tasks, quality metrics, M2 permission matrix, and KPI metrics overview.
- Sprint U: Doctor **M3-10** RLS global table exclusions (`memory_migrations`, `schema_meta`); `PostgresRLSGlobalTables()` catalog; SQL rev 15 table comment; `verify-local` runs `regression-short`.
- Sprint T: Doctor **TR3-08** Prometheus event-replay segment (`ASH_METRICS_EVENT_REPLAY=1`); `make regression-short` quick regression target; alerts `TestPrometheusEventReplaySegment`.
- Sprint S: memory tests use `store.OpenTest` for reliable DB cleanup; Doctor report UI labels for M3-03..M3-09.
- Sprint R: `internal/opsenv` shared worker ops snapshot; Doctor **M3-09** readyz/Scale contract; CI runs full `postgres-e2e` on push to main.
- Sprint Q: `/readyz` exposes schema mode, SQL migration version, OTel, alerts eval interval, and metrics replay flags; production config observability section; CI PR job `postgres-sql-schema-e2e`.
- Doctor **TR3-07** plugin export health (registry + `plugin.export_failed` audit); Scale `metricsEventReplayEnabled`; Observability OTel exporter health panel.
- OTel waterfall export health: built-in `ash-otel-exporter` plugin row; run completion reports export via `pluginhealth.ReportExport`; Scale readiness `otelEnabled` / `alertsEvalInterval`.
- Worker background governance alert evaluation via `ASH_ALERTS_EVAL_INTERVAL` (minimum 1m); `postgres-sql-schema-e2e` runs Doctor TR3 on live Postgres; swagger regen for Sprint E–M API endpoints.
- Sprint M: `postgres-sql-schema-e2e` runs Postgres RAG FTS integration test; `postgres-e2e-migrate` sources `postgres-up.sh` for port export; Doctor **TR3-06** validates `postgres-tsvector` retrieval (skipped on SQLite).
- Postgres RAG full-text search: `rag_chunks.search_vector` tsvector + GIN index (SQL rev 17), `postgres-tsvector` retrieval path alongside SQLite FTS5.
- Observability governance alert rules panel; Scale page plugin export health summary.
- Governance alert rules and live Prometheus gauges: memory backlog, RAG FTS fallback rate, plugin export failures; `GET /api/v1/rag/profile`; Observability RAG retrieval card.
- RAG degradation observability: query `retrievalMode` / `ftsAvailable`, `rag.retrieved` run events, derive `ash_rag_queries_total` and `ash_rag_fts_fallback_total`, Scale readiness RAG retrieval snapshot.
- Plugin export failure events: `plugin.export_failed` audit + optional run event (`runId` on export-report), derive `ash_plugin_export_failures_total`.
- Plugin export health: `plugin_registry` columns `last_export_at` / `export_errors` / `drop_count` (SQL rev 16), `POST /api/v1/plugins/{pluginId}/export-report`, `GET /api/v1/plugins/health`, Automation UI health columns.
- OTel traces skeleton: `internal/observability/otel` (OTLP provider, live run/step/gate/rag/model/tool spans, waterfall batch export), Worker init, `GET /api/v1/observability/otel/status`.
- Memory schema migration runner: `memory_migrations` table (SQL rev 15), `POST /api/v1/memory/migrate`, `memory.migrated` event, `ash_memory_migration_runs_total` derive; Scale exposes `memoryCatalogVersion` / `memoryPendingMigrationRecords`.
- Memory P1 derive metrics: `ash_memory_hit_used_total`, `ash_memory_deprecated_total`, `ash_memory_queries_total`, `ash_memory_query_latency_ms`; `memory.query` run event when query includes `runId`.
- Memory governance derive metrics: `ash_memory_unreviewed_backlog`, `ash_memory_missing_evidence_total`; `memory.reviewed` events include `layer` for replay labels.
- Scale readiness `readinessWarnings` when `ASH_SCHEMA_MODE=sql` conflicts with dual-write shadow Postgres.
- Event-derived metrics parity: `derive.ValidateReplayParity`, Doctor **TR3-05**, nightly `make postgres-sql-schema-e2e` CI job.
- Production Postgres template `doc/checklists/postgres-production-config.md`; RDS checklist aligned to SQL revision 14 and M3 8 / ALL 34.
- TR0 event payload JSON Schema validation (`ASH_VALIDATE_EVENT_PAYLOADS=1`) and Postgres SQL revision `000003` for `runs` / `run_events`.
- Postgres `golang-migrate` skeleton (`internal/store/sqlmigrations`), `ash migrate schema up|down|version`, `ASH_SCHEMA_MODE` / `ASH_DISABLE_AUTOMIGRATE`, and `make migrate-schema`.
- Observability config JSON Schema (`ash.obs/v0.1`): `config/ash-observability.yaml`, `obsconfig.Load()` at Worker startup, outbound-export requires redaction.
- Rules DSL JSON Schema (`internal/rules/schemas/ash.rules.v0.1.schema.json`) with runtime validation in `ParseAndValidate` and 20 invalid-sample regression tests.
- Added frontend scenario listing API and Runs page scenario picker backed by `/api/v1/scenarios`.
- Added Doctor console suite selector for TR0, TR1, TR2, and ALL.
- Added TR0-08 doctor case to verify `feature_delivery`, `hotfix`, and `security_patch` scenarios are loaded.
- Enabled dev-mode default plugin registry gRPC on `127.0.0.1:19091` when `ASH_PLUGIN_GRPC_ADDR` is unset.
- Added M1 memory governance hints on candidate creation (`governance.duplicates` / `governance.conflicts`).
- Added M1 self-iteration API (`/api/v1/improve/proposals`) with experiment replay compare, canary, promote, and rollback.
- Added Doctor TR1-05 Rules DSL schema validation case.
- Expanded Memory and Automation console panels for governance edges, memory query, and improve proposals.
- Added TR2 compliance console at `/ui/compliance` with live readiness cards and structured doctor reports.
- Added `GET /api/v1/spaces/:spaceId/resource-scopes` for resource scope visibility.
- Reworked Doctor page to render pass/fail case tables via `DoctorReportView`.
- Added secret leak scanning (`internal/security/leakscan`) with compliance API and TR2-05 doctor case.
- Added compliance audit export bundling doctor reports with audit logs.
- Applied audit log payload redaction in list API when `redactPayload` policy is enabled.
- Fixed doctor `ALL` suite to run the full TR0/TR1/TR2 case set.
- Added Doctor TR3 suite (TR3-01 memory migration, TR3-02 RAG FTS fallback, TR3-03 cost/latency SLO metrics, TR3-04 audit provenance).
- Added scale readiness API and `/ui/scale` console for GA TR3 visibility.
- Added run provenance API (`GET /api/v1/runs/:runId/provenance`).
- Embedded secret-scan summary in compliance export bundles; compliance page one-click audit redact toggle.
- Extended doctor `ALL` to include TR3 cases.
- Added Runs console provenance panel (TR3-04) wired to `GET /api/v1/runs/:runId/provenance`.
- Added API tests for scale readiness and run provenance; Swagger annotations for compliance/scale/provenance.
- Added cross-platform `scripts/verify-local.sh` for Git Bash and Linux.
- Added Swagger godoc for improve proposals API and `scripts/regenerate-swagger.sh`.
- Added `TestALLSuite` doctor regression (22 cases) and compliance export integration test.
- Compliance console export supports TR2/TR3/ALL doctor suite selection.
- Added M2 permission matrix (`internal/authz`): RBAC catalog, scenario×role tool policies, run-time enforcement, API/UI, and Doctor `M2-01`.
- Added `PUT /api/v1/spaces/:spaceId/resource-scopes/:scopeId` for scenario tool policy updates; Doctor `M2-02`; M2 cases in `ALL` suite (24 cases).
- Space console scenario policy editor; compliance console M2 readiness cards.
- Audit log on scenario policy updates (`scope.policy_updated`); Doctor M2-03 runtime `POLICY_DENIED` enforcement.
- M3 tenant isolation helpers (`store.EnforceSpaceAccess`), Postgres profile/readiness, Doctor M3 suite, and migration guide `doc/05-M3-多租户与Postgres演进.md`.
- Scale readiness exposes `databaseDialect`, `postgresConfigured`, `migrationReady`; `scripts/postgres-smoke.sh`.
- Postgres migration CLI: `ash migrate plan|copy|verify|sync` and `ash migrate dual-write enable|disable|status|sync`; runtime mirror via `ASH_DUAL_WRITE_POSTGRES_URL`; `scripts/migrate-postgres.sh`.
- Doctor M3-03 migration catalog check; scale readiness exposes migration table count, dual-write status, and last sync time; Scale/Compliance consoles updated.
- API tenant enforcement helpers (`requireRequestSpace` / `requireTargetSpace`) across space-param routes and cross-space writes; Worker auto-loads dual-write from `.ash/migration/dual-write.json`; `doc/TODO.md` tracks Postgres e2e migrate validation.
- Docker Postgres dev stack (`docker-compose.postgres.yml`); `scripts/postgres-e2e-migrate.sh` and `make postgres-e2e`; Doctor M3-04 live migrate verify (`ASH_MIGRATE_E2E=1`); integration test tag `integration`.
- Runs API returns `201` with `executionError` when run is created but policy/execution fails; `actorRole` on run summary.
- Cross-space run/memory access returns `403`; Runs console actor-role picker; Scale page database/M3 section.
- Switched default SQLite driver to pure-Go `github.com/glebarez/sqlite` (no CGO on Windows).
- Fixed JSON secret detection/redaction in `leakscan`; dev auth honors `X-ASH-Space-ID`; test DB auto-close via `store.OpenTest`.
- Added an ExecGo/Codex agent execution boundary under `internal/agentexec`, including executor contracts, a Codex-backed ExecGo adapter, cancellation/status hooks, and a deterministic static executor for tests and smoke runs.
- Added `agent` step support to the Rules DSL, with agent adapter metadata, capabilities, prompts, timeout configuration, retry metadata, and approval metadata on gates.
- Added a QA verification step to `scenarios/feature_delivery.yaml`, separating Codex implementation from test execution evidence.
- Added lightweight repository RAG indexing and query support under `internal/rag`, with line-range citations, content digests, space scoping, and simple term scoring.
- Added persistent run observability models for run steps, tool calls, agent tasks, artifact indexes, checkpoints, audit logs, RAG documents/chunks, usage metrics, quality metrics, feedback, organization/space membership, roles, scopes, audit exports, and plugin registry records.
- Added run timeline, tool call, agent task, cancel, approval, and RAG service surfaces in the run service layer.
- Added richer run artifact bundle metadata, including repo root, issue/spec, event range, agent task id, evidence references, generated release notes, rollback plan, diff capture, and default test report generation.
- Added SPA fallback coverage for `/ui/` routes so direct navigation such as `/ui/runs` serves the frontend shell while still serving static assets.
- Added frontend dependencies for TanStack Router, TanStack Table, and lucide-react icons.
- Added Vite type declarations and lockfiles for reproducible frontend and Go dependency installs.

### Changed

- Switched the frontend router from `react-router-dom` to TanStack Router with `/ui` as the basepath.
- Reworked the ASH Console layout with icon navigation, page headings, denser panes, status pills, empty states, and table-based Runs rendering.
- Expanded the Runs page to use TanStack Table and clearer controls for refresh, new run, resume, and replay actions.
- Refined Memory and Doctor pages with structured headers, action icons, report panes, candidate counts, and explicit empty states.
- Expanded run execution to persist step state, checkpoints, tool call results, agent execution records, audit entries, RAG/memory evidence, and artifact indexes during scenario execution.
- Changed artifact generation from M0 stubs toward evidence-backed release notes, rollback plans, git diff capture, test report preservation, and manifest producer metadata.
- Updated `feature_delivery` from a direct tool-chain implementation step to an agent implementation step followed by QA test verification.
- Updated frontend documentation to reflect the new Vite + React + TypeScript stack, TanStack Router/Query/Table usage, lucide icons, and `/ui/` dev route.

### Fixed

- Fixed `runs.Service.eventsFor` infinite recursion when bound to a request context (approval/cancel paths stack overflow).
- Fixed malformed return statements in the SQLite store initialization path.
- Fixed backend static UI routing so nested SPA routes fall back to `index.html` without allowing path traversal outside the web directory.
- Fixed event spelling for memory review request emission.
- Added regression coverage for static UI SPA fallback behavior.
- Updated run control and doctor tests to account for the expanded execution model and static agent executor.

### Notes

- The working tree also contains IDE project metadata under `.idea/`. Keep those files staged only if the repository intentionally tracks editor configuration.
- **v0.1.0-mvp** is the first tagged MVP release anchor (Sprints AY–BH, release gates, signoff workflow). Cloud Postgres cutover and production sign-off names remain post-tag checklist items (see `doc/TODO.md` Sprint BI).
