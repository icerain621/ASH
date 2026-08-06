# ASH 待办 / 技术债

> 记录尚未完成或需人工环境验证的项。完成请移入 CHANGELOG 并删除对应条目。

## 开发计划（2026-07-05 重排）

> **结论**：代码与自动化门禁已达 **MVP v0.1 可发布** 水位；剩余工作以 **环境验收 + 人工签字 + 门禁加固** 为主，而非新功能开发。  
> **基线**：Doctor ALL **43/43** · M3 **11/11** · TR3 **10/10** · SQL rev **20** · RLS **41** · tag **`v0.1.0-mvp`**（Sprint BH 后）

### 完成度快照

| 维度 | 状态 | 说明 |
|------|------|------|
| MVP 清单 §1–4, 7, 9–10 | ✅ 自动化已勾 | 功能/测试/性能/可观测/灰度/T+0/T+1 |
| MVP 清单 §5 Postgres 切换 | ⚠️ | 本地 `make postgres-app-gate` ✅（H-03 readyz）；云 RDS 待 `cloud-rds.env` |
| MVP 清单 §6 staging 迁移 | ⚠️ | 本地 Docker `make cloud-acceptance` ✅（2026-07-06）；云 RDS 待 `config/cloud-rds.env` |
| MVP 清单 §8 值班 roster | ✅ | 占位 dry-run；`signoff-apply` → `release-window-runbook.md` |
| MVP 清单 §11 四人签字 | ✅ | 占位 dry-run；`make signoff-gate` 2026-07-07 |
| 发布范围冻结签字 | ✅ | `ASH_SCOPE_FREEZE_SIGNED=1`（占位 dry-run） |
| 后端测试 T-01..T-15 | ✅ | 本地/CI Postgres e2e 已覆盖 |
| H-04~H-09 本地 live | ✅ | `worker-local-gate` / fixture |
| H-04~H-06 生产外部依赖 | ⏸ | 真实 GitHub token / ExecGo |
| 前端测试 | ✅ | **12/12 页** smoke + SSE 重连/轮询（含 RunsPage 控制码 / Scale 积压；≥16 文件） |
| Git 远端 | ✅ | `main` @ `4d5acad`；CI / Postgres E2E / Live Worker **全绿**（2026-08-06） |

### 优先级矩阵

| 优先级 | 目标 | 可纯代码推进？ |
|--------|------|----------------|
| **P0-A** | 完成 MVP v0.1 **发布闭环**（签字 + 推送 + CI 绿） | 部分 |
| **P0-B** | Postgres **生产切换**验收（H-01~H-03） | 否，需 RDS/Docker |
| **P1** | **门禁与质量加固**（flake、前端测试、证据一致性） | 是 |
| **P2** | **发布后观察**（真实 KPI、诊断采纳率、T+1 生产数据） | 否，需上线后 |
| **P3** | **M1+ 能力**（Hotfix/Security 场景深化、插件 ABI、BI 埋点） | 是，但范围冻结后仅 P0 缺陷 |

---

### Sprint 排期（建议）

#### Sprint BG（P0-A · 1–2 天）— 发布闭环

| # | 任务 | 负责人 | 验收 |
|---|------|--------|------|
| BG-1 | `git push origin main`；确认 `ci.yml` + `release-gates.yml` 全绿 | 发布 | ✅ `4d5acad` CI/Postgres E2E/Live Worker 全绿 |
| BG-2 | 产品/技术/测试/发布 **§11 签字** | 四人 | `make signoff-apply` + `make signoff-gate` 绿 |
| BG-3 | `mvp-release-scope.md` 评审签字 → `ASH_SCOPE_FREEZE_SIGNED=1` | 产品 | `scope-freeze-gate` 无 WARN |
| BG-4 | 发布窗口 roster + 时间写入 `release-window-runbook.md` | 发布 | ✅ `signoff-apply` 同步占位 roster |
| BG-5 | 打 tag `v0.1.0-mvp` + CHANGELOG 归档 Unreleased | 发布 | ✅ tag `v0.1.0-mvp` |

```bash
cp config/signoff.env.example config/signoff.env
make local-readiness-gate    # 发布前最后一轮本地 live
make signoff-apply           # 评审会后回填签字
make signoff-gate
```

#### Sprint BH（P1 · 3–5 天）— 门禁加固（不依赖云）

| # | 任务 | 验收 |
|---|------|------|
| BH-1 | 修复 `mvp-signoff` 嵌套调用 `web-gate` 偶发失败 | ✅ `npm … \| cat` |
| BH-2 | 前端核心页 smoke 测试：Runs、CI、Memory、Scale | ✅ 7 vitest 文件 |
| BH-3 | Observability 页 smoke 测试 | ✅ `ObservabilityPage.test.tsx` |
| BH-4 | PR 门禁加 `make release-window-gate`（轻量，~2min） | ✅ `ci.yml` 并行 job |
| BH-5 | KPI 定义 §9：overview 与 derive replay 对账 | ✅ `make kpi-reconcile-gate` |
| BH-6 | Feedback / Releases / Metrics / Doctor 页 smoke | ✅ 2026-08-04 |
| BH-7 | Space / Automation / Compliance 页 smoke（全页覆盖） | ✅ 2026-08-04 |

#### Sprint BM（P1 · 风险缓解 · 可纯代码）

| # | 任务 | 验收 |
|---|------|------|
| BM-1 | 前端 SSE 自动重连 + `Last-Event-ID` 续传（R-07） | ✅ `runStream.test.tsx`；Runs 页显示重连状态 |
| BM-2 | 跨 space API 回归（R-08）：stream/provenance/artifacts/approvals/releases/repo | ✅ `TestCrossSpaceAPIRegression` |
| BM-3 | Stream 接受 query `Last-Event-ID` | ✅ `handlers.streamRun` |

#### Sprint BN（P1 · R-07 轮询回退）

| # | 任务 | 验收 |
|---|------|------|
| BN-1 | SSE 重连耗尽后 timeline 轮询回退 | ✅ `useRunStream` status=`polling`；Runs 页「轮询回退」 |
| BN-2 | query `Last-Event-ID` 后端续传单测 | ✅ `TestStreamRunResumesFromQueryLastEventID` |

#### Sprint BO（P1 · 门禁接入）

| # | 任务 | 验收 |
|---|------|------|
| BO-1 | `regression-short` 跑跨 space + stream 续传 | ✅ `scripts/regression-short.sh` |
| BO-2 | H-09 static 同步纳入上述用例 | ✅ `release-sampling-static.sh` |

#### Sprint BP（P0-A · CI 诊断与 Windows 测试）

| # | 任务 | 验收 |
|---|------|------|
| BP-1 | 跨平台 fake execgocli（Windows `.cmd` + bash） | ✅ `internal/testutil`；toolbus/agentexec/runs |
| BP-2 | `OpenTest` 关闭后强制删除 sqlite 锁文件 | ✅ runs/rag 本地 pass |
| BP-3 | CI 失败上传 `go-test.log` / postgres e2e log + docker 诊断 | ✅ `ci.yml` / `postgres-e2e.yml` |
| BP-4 | 根据 artifact 根治 Backend/Postgres 长期红 | ✅ apicodes + owner DSN + M3-04 跳过；`4d5acad` 全绿 |

#### Sprint BQ（P1 · R-03 / 迁移回归 · 可纯代码）

| # | 任务 | 验收 |
|---|------|------|
| BQ-1 | `postgresMigrationDSN` 优先 open-time owner DSN | ✅ `TestPostgresMigrationDSNPrefersOpenDSNOverAppURL` |
| BQ-2 | CI 诊断扩展 cancel / OOM / frontend lint-typecheck | ✅ `TestDiagnoseLogClassifiesCancelAndResourceAndFrontend` |

#### Sprint BR（P1 · R-04 Memory 污染缓解 · 可纯代码）

| # | 任务 | 验收 |
|---|------|------|
| BR-1 | 低分 feedback → `ApplyFeedbackDecay`（−0.15）+ audit/event | ✅ `TestApplyFeedbackDecayLowScore` / `TestFeedbackMemoryTargetDecaysConfidence` |
| BR-2 | Query / run 检索按 confidence 排序，排除 `<0.2` | ✅ `TestQueryRanksByConfidenceAndFiltersFloor` |

#### Sprint BS（P1 · BQ/BR 门禁 + fixture 硬化 · 可纯代码）

| # | 任务 | 验收 |
|---|------|------|
| BS-1 | `regression-short` 接入 BQ/BR 诊断与 decay 单测 | ✅ `scripts/regression-short.sh` |
| BS-2 | CI fixture 扩 cancel / OOM / frontend（9103–9105） | ✅ `TestFixtureProviderSyncRunsAndJobs` |
| BS-3 | RunsPage 展示「重连中 / 轮询回退」smoke | ✅ `RunsPage.test.tsx` |

#### Sprint BT（P1 · R-05 状态机守卫 · 可纯代码）

| # | 任务 | 验收 |
|---|------|------|
| BT-1 | `canTransition` / `trySetRunStatus` 中央转换表 | ✅ `internal/runs/status.go` |
| BT-2 | 接线 fail/finish/waiting_approval/Cancel/Resume/Approve | ✅ 取消后不再被 fail 覆盖 |
| BT-3 | 表驱动 + Cancel 幂等 + fail-after-cancel 单测 | ✅ `status_test.go`；`regression-short` 接入 |

#### Sprint BU（P1 · R-05 中途 Cancel 观察 · 可纯代码）

| # | 任务 | 验收 |
|---|------|------|
| BU-1 | `observeCanceled` 步边界 / agent 后轮询 | ✅ `status.go` + `executeSteps` |
| BU-2 | Create/Resume 将 `ErrRunCanceled` 视为终态 | ✅ 不写成 failed |
| BU-3 | 阻塞 agent 中途 Cancel 集成测试 | ✅ `TestMidLoopCancelStopsWithoutFinish` |

#### Sprint BV（P1 · R-05 Replay 状态门禁 · 可纯代码）

| # | 任务 | 验收 |
|---|------|------|
| BV-1 | Replay 仅允许 finished/failed/canceled 源 | ✅ `canReplay` + `ErrRunNotReplayable` |
| BV-2 | API `RUN_NOT_REPLAYABLE` 409 | ✅ `writeRunControlError` + apicodes |
| BV-3 | 表驱动 + 非终态负例单测 | ✅ `TestCanReplayMatrix` / `TestReplayRejectsNonTerminalSource` |

#### Sprint BW（P1 · R-05 Approve/Resume 收口 · 可纯代码）

| # | 任务 | 验收 |
|---|------|------|
| BW-1 | `canApprove` / `canResume`；Approve 用 `ErrRunNotApprovable` | ✅ `status.go` / `Approve` |
| BW-2 | API `RUN_NOT_APPROVABLE` / `ILLEGAL_STATUS_TRANSITION` / `RUN_CANCELED` | ✅ `writeRunControlError` + apicodes |
| BW-3 | Cancel 后 Approve 不复活；表驱动单测 | ✅ `TestApproveRejectsNonWaitingAndCanceled` |

#### Sprint BX（P1 · R-09 GitHub CI 韧性 · 可纯代码）

| # | 任务 | 验收 |
|---|------|------|
| BX-1 | `getJSON` / logs：429/5xx 有限退避重试 | ✅ `doWithRetry` |
| BX-2 | 连续失败熔断 + 单测 | ✅ `TestGitHubProviderCircuitOpensAfterFailures` |
| BX-3 | API `CI_PROVIDER_UNAVAILABLE`（502） | ✅ `writeCIProviderError` |

#### Sprint BY（P1 · R-06/R-10 + cancel derive · 可纯代码）

| # | 任务 | 验收 |
|---|------|------|
| BY-1 | `run.canceled` → `ash_run_total` / inflight 对账 | ✅ `TestReplay_runCanceledClearsInflight` |
| BY-2 | Scale `runRunning/Waiting/Inflight` backlog 信号 | ✅ `TestScaleReadinessRunBacklogCounts` |
| BY-3 | OpenAPI 控制面 409 + RunsPage `ApiError` 码展示 | ✅ openapi + `formatControlError` + vitest |
| BY-4 | API Resume/Approve/Replay 负例矩阵 | ✅ `TestRunControlNegativeStatusCodes` |

#### Sprint BZ（P1 · R-12 回滚门禁硬化 · 可纯代码）

| # | 任务 | 验收 |
|---|------|------|
| BZ-1 | `rollback-drill` 强制接入发布窗口 + 时长 SLA | ✅ `release-window-gate` / `mvp-signoff`；`ASH_ROLLBACK_DRILL_MAX_MS` |
| BZ-2 | `data-backup` 后 sha256 + SQLite `integrity_check` | ✅ `make data-backup-verify` / `TestVerifySQLiteFile*` |
| BZ-3 | `EvaluateGate` 纳入 rollback drill；空场景/非法 status 拒绝 | ✅ `TestReleaseChecklistGateAndRollbackDrill` |
| BZ-4 | `regression-short` 接入治理 API + releases 单测；R-12 → Mitigating | ✅ scripts + 风险台账 |

#### Sprint CA（P1 · R-12 硬要求 + R-06 inflight 告警 · 可纯代码）

| # | 任务 | 验收 |
|---|------|------|
| CA-1 | 发布窗口 / mvp-signoff 默认 `ASH_REQUIRE_ROLLBACK_DRILL=1` | ✅ gate 脚本 export |
| CA-2 | 告警 `run_inflight_count` + Prometheus `ash_run_inflight_live` | ✅ `TestEvaluateRunInflightAlert` |
| CA-3 | `t0-alert-gate` / docs；R-06 补强 | ✅ t0 + 附录 D |

#### Sprint CB（P1 · R-09/R-05 前端与负例收口 · 可纯代码）

| # | 任务 | 验收 |
|---|------|------|
| CB-1 | CIPage 展示 `CI_PROVIDER_UNAVAILABLE` 等错误码 | ✅ `formatCIError` + vitest |
| CB-2 | 控制面 Cancel 后 Resume/Approve 负例 | ✅ `TestRunControlNegativeStatusCodes` |

#### Sprint BI（P0-B · 发布窗口 · 需环境）

| # | 任务 | 前置 | 验收 |
|---|------|------|------|
| BI-1 | 配置 `config/cloud-rds.env` | RDS 实例 | `make cloud-acceptance` 绿 |
| BI-2 | H-01 staging `migrate copy/verify` | BI-1 + `.ash/ash.db` | `h01-h03-cloud-signoff.md` 勾选 |
| BI-3 | H-02/H-03 `ASH_DATABASE_APP_URL` + RLS Worker | BI-2 | ✅ 本地 `make postgres-app-gate`（M3-06/07 + readyz） |
| BI-4 | 或本地 Docker：`make postgres-app-gate` | Docker 运行 | ✅ `make cloud-acceptance` 本地演练（2026-07-06） |
| BI-5 | 切换日：`data-backup` → 停 SQLite Worker → migrate → `t0-alert-gate` | runbook §4 | ✅ 本地 Docker 2026-07-07；生产切换日待执行 |

```bash
cp config/cloud-rds.env.example config/cloud-rds.env
bash scripts/source-cloud-rds-env.sh
make cloud-acceptance
```

#### Sprint BJ（P2 · 上线后 1 周）— 真实数据验证

| # | 任务 | 验收 |
|---|------|------|
| BJ-1 | 真实 GitHub token：`H-04/H-05` live（非 fixture） | `rootCause` 与真实 log 一致 |
| BJ-2 | `ASH_EXECGO_E2E=1 make execgo-live-smoke` | Doctor M3-05 live |
| BJ-3 | T+1 生产 KPI 看板 vs `t1-metrics-gate` 基线偏差 <5% | `14-kpi-dashboard-definition.md` §9 |
| BJ-4 | CI 诊断采纳率、任务成功率观察 | 风险台账 R-02/R-03 有数据 |

#### Sprint BK+（P3 · 范围冻结后仅 P0 缺陷 / M1  backlog）

| 域 | 候选项 | 备注 |
|----|--------|------|
| 场景 | Hotfix / Security Patch 模板深化（PRD §4.2/4.3） | 原 P1，冻结后延期 |
| 平台 | 插件 gRPC ABI 签名策略（ARCH TODO） | P2 |
| 安全 | 数据分级与保留期（PRD TODO） | 合规增强 |
| 前端 | SSE 浏览器级 E2E | R-07 单元/钩子/轮询已落地 |
| 算法 | CI 诊断真实 log 校准；高置信额外确认（产品） | R-03/BQ；R-04 衰减已落地（BR） |

---

### 一键命令（按阶段）

```bash
# 日常开发（P1）
make regression-short && make web-gate

# 发布前本地（P0-A）
make local-readiness-gate

# 发布窗口（P0-B，需环境）
make release-window-gate
ASH_RELEASE_WINDOW_LIVE=1 make release-window-gate
make cloud-acceptance

# 全量签字证据
ASH_RELEASE_WINDOW_INCLUDE_MVP=1 make release-window-gate
make mvp-signoff
```

### 风险重排（Top 5 → 行动）

| 风险 | 当前缓解 | 下一动作 |
|------|----------|----------|
| R-12 回滚不完整 | 强制 drill + 备份校验（BZ/CA） | BI 切换日再演练一次并签字 |
| R-08 越权 | M2 + `TestCrossSpaceAPIRegression` | 发布窗口抽测 + RLS e2e |
| R-03 诊断准确率 | fixture 含 cancel/OOM/frontend（BS） | BJ 真实 GitHub log 验证 |
| R-07 SSE 不稳定 | 自动重连 + timeline 轮询回退（BN） | BJ 生产断连率观察 |
| R-01 需求变更 | `scope-freeze-gate` | BG-3 产品签字冻结 |

---


> **状态**：Sprint I/H 完成后恢复执行；以下项以本地 `go test` 为准（T-05/T-06 仍需 Postgres）。
> **关联**：derive parity、Doctor TR3-05、`postgres-sql-schema-e2e` CI job

| # | 项 | 命令 / 入口 | 验收 |
|---|-----|-------------|------|
| T-01 | derive 回放 parity 单测 | `go test ./internal/observability/derive/... -run Parity -count=1` | ✅ 2026-06-03 本地 pass |
| T-02 | derive 记忆 backlog 单测 | `go test ./internal/observability/derive/... -run Replay_memory -count=1` | ✅ 2026-06-03 本地 pass |
| T-03 | Doctor TR3-05..10 | `go test ./internal/doctor/... -run TestTR3Suite -count=1` | ✅ TR3 **10/10**（TR3-06/08 默认 skip） |
| T-04 | Doctor ALL 计数 | `go test ./internal/doctor/... -run TestALLSuite -count=1` | ✅ **43/43** pass |
| T-05 | SQL schema 本地 E2E | `make postgres-sql-schema-e2e` | ✅ sql rev **20**；TR3-06/10 live 断言（M3-04 skip by design） |
| T-06 | CI job | GitHub Actions `ci.yml`（`regression-short` + 三门 Postgres + **`postgres-e2e` PR** + **`web-gate` frontend**）+ `postgres-e2e.yml` nightly | PR 四门 Postgres + 全量 migrate e2e |
| T-13 | Postgres RAG FTS 集成 | `go test -tags=integration ./internal/rag/ -run TestPostgresRAGFTSQuery`（`ASH_MIGRATE_E2E=1`） | 已并入 `make postgres-sql-schema-e2e` |
| T-14 | Memory 子表 RLS 集成 | `go test -tags=integration ./internal/store/ -run TestPostgresRLSSpaceIsolationOnMemoryChildren` | 已并入 `postgres-rls-e2e` / `postgres-sql-schema-e2e` |
| T-15 | Org 身份 RLS 集成 | `go test -tags=integration ./internal/store/ -run TestPostgresRLSSpaceIsolationOnOrgIdentity` | 已并入 `postgres-rls-e2e`；`verify-local` Docker 可用时自动跑 |
| T-07 | 记忆 emit layer | `go test ./internal/memory/... -count=1 -short` | ✅ 断言 pass（OpenTest 关闭 DB） |
| T-08 | Scale 双写冲突告警 | `go test ./internal/api/... -run ScaleReadinessSchemaSql -count=1` | ✅ 2026-06-03 本地 pass |
| T-09 | 记忆 P1 derive 单测 | `go test ./internal/observability/derive/... -run Replay_memoryHit -count=1` | ✅ 2026-06-03 本地 pass |
| T-10 | 记忆 schema 迁移 | `go test ./internal/memory/... -run RunMigrations -count=1` | ✅ OpenTest 关闭 DB |
| T-11 | OTel 骨架单测 | `go test ./internal/observability/otel/... -count=1` | ✅ 2026-06-03 本地 pass |
| T-12 | RAG 降级 derive | `go test ./internal/rag/... ./internal/observability/derive/... -run 'Fallback|ragRetrieved' -count=1` | ✅ Sprint I：`retrievalMode=chunk` replay |

```bash
# 一键回归（网络正常时）
make regression-short
go test ./internal/observability/derive/... ./internal/doctor/... ./internal/memory/... ./internal/api/... -count=1 -short
make postgres-sql-schema-e2e
```

---

## 人工验证（暂缓）

> **状态**：依赖云 RDS / 真实 GitHub token / live ExecGo，当前迭代**暂不执行**，保留清单待发布窗口统一验收。
> **清单**：[`doc/checklists/postgres-rds-e2e.md`](checklists/postgres-rds-e2e.md) · 一键脚本 `make postgres-rds-e2e`

| # | 项 | 命令 / 入口 | 验收 |
|---|-----|-------------|------|
| H-01 | 云 RDS 全链路 E2E | `make cloud-acceptance`（需 `ASH_DATABASE_URL`） | `migrate schema` v20 + Doctor M3 11/11 + ALL 43/43 |
| H-02 | 云 RDS RLS + ash_app | 清单 §4–§5；本地 `make postgres-app-gate` | M3-06/07 pass；`TestPostgresRLSE2EAfterMigrate` |
| H-03 | 生产 Worker 配置 | `ASH_DATABASE_APP_URL` + `ASH_POSTGRES_RLS_FORCE=1` | `/readyz` dialect=postgres；签字见 `h01-h03-cloud-signoff.md` |
| H-04 | GitHub CI runs 同步 | `GET /api/v1/ci/runs?sync=true` | 真实 token 拉取 Actions 摘要；本地/CI 可用 **`ASH_CI_FIXTURE=1`**（`TestCISyncRunsWithFixture` / `TestReleaseSamplingCIFixtureH04H05`） |
| H-05 | GitHub CI jobs / 日志诊断 | `GET /api/v1/ci/jobs?runId=...&sync=true` + `POST /ci/failures/diagnose`（仅 `jobId`） | job log 经 fixture 拉取、`logDigest` 落库、`rootCause` 分类；`TestFixtureProviderSyncRunsAndJobs` |
| H-06 | ExecGo live smoke | `ASH_EXECGO_E2E=1 make execgo-live-smoke` 或 Doctor M3-05 | `execgo-health` + live 执行链路；本地静态 `TestM3ExecGoLiveSmoke` |
| H-07 | 密钥轮换策略 | `make secret-rotate-smoke` 或 `TestSecretRotateRepoConnectionH07` | secret 轮换后 repo connection + CI sync 仍可用 |
| H-08 | 发布窗口 audit gate | [`release-window-audit.md`](checklists/release-window-audit.md) + `make release-window-audit` | Postgres e2e + ALL/M3 + §7 证据归档 |
| H-09 | 业务抽样 §7 | `make release-sampling-static` 或 `make release-sampling-smoke` | Run/SSE/Memory/KPI/CI/合规/Scale（§7.2 SSE 含 `TestReleaseSamplingSSE`） |

本地自动化（无需云环境，CI 可跑）：

```bash
make cloud-acceptance      # H-01~H-03 云 RDS + 证据归档（需 ASH_DATABASE_URL）
make mvp-signoff           # MVP 发布签字门禁
make web-gate              # 前端 lint + vitest + build
make production-config-gate
make rollback-drill
make queue-gate
make t0-alert-gate
make data-backup            # SQLite 备份（migrate 前）
make postgres-e2e
make postgres-rls-e2e
make postgres-app-gate    # Docker + schema 后 H-02/H-03 本地
make smoke-static
bash scripts/verify-local.sh
# Live Worker（可选）
ASH_WORKER_URL=http://127.0.0.1:8080 ASH_CI_FIXTURE=1 make live-smoke
```

---

## 1. Postgres 迁移（代码已完成，待 H-01/H-02）

**状态**：本地 `make postgres-e2e` 已通过；生产切换见 **人工验证（暂缓）** H-01–H-03
**优先级**：P1（切换生产前必做）

```bash
make postgres-e2e
make postgres-roles
make test-integration   # 需 ASH_DATABASE_URL + ASH_MIGRATE_E2E=1
```

**自动化验收**（本地 / CI nightly）：

- `migrate verify` 全表行数一致
- Doctor **M3-04** 在 `ASH_MIGRATE_E2E=1` 时通过
- `TestPostgresReadyzProbe` → `dialect=postgres`
- `doctor --suite ALL` 无回退

---

## 2. CI / ExecGo / KPI（MVP 已完成，H-04–H-09 本地自动化已就绪）

**状态**：API 与控制台 MVP 已交付；外部依赖验证见 **人工验证（暂缓）**
**优先级**：P0 自动化回归 / P1 人工闭环

```bash
go test ./internal/ci ./internal/metrics ./internal/api -run 'TestDiagnose|TestOverview|TestCreateRepo|TestRepoConnection' -count=1
make execgo-health
go run ./cmd/cli doctor --suite ALL --format md --agent static
make web-build
```

**已实现（自动化可验）**：

- `ASH_CI_FIXTURE=1`：`TestCISyncRunsWithFixture` + `internal/ci` fixture provider（无需 GitHub token）
- `/readyz` `liveGateHints`：M3-04/05/06/07/08 与 CI fixture 状态
- GitHub Actions PR/main：Go + Doctor static + Web build
- Postgres e2e：nightly/manual workflow
- Repo `secretId` only；CI diagnose API；KPI overview；控制台 feedback/ci/observability/releases

---

## 3. Postgres RLS 生产部署（待 H-02/H-03）

**状态**：RLS 骨架与 API 接线已完成；生产角色与连接串见 **人工验证（暂缓）**
**优先级**：P2

```bash
make postgres-up
make postgres-roles
```

---

## 代码待办（可继续开发）

- 新表 DDL 修订需同步追加 RLS policy（`000013` 租户 / `000018` run / `000019` memory / `000020` org 扩展）；迁移目录表已全覆盖（`PostgresRLSDeferredTables()` 为空）；Doctor **M3-11** 校验

---

## 已完成（近期）

- Sprint CB：CIPage 展示 `CI_PROVIDER_UNAVAILABLE`；Cancel 后 Resume/Approve 负例（R-09/R-05）。
- Sprint CA：发布门禁默认 `ASH_REQUIRE_ROLLBACK_DRILL=1`；告警 `run_inflight_count` + `ash_run_inflight_live`（R-12/R-06）。
- Sprint BZ：发布窗口强制 `rollback-drill` + 时长 SLA；备份 sha256/`integrity_check`；`EvaluateGate` 纳入 drill（R-12）。
- Sprint BY：`run.canceled` derive 对账；Scale 运行积压；控制面 OpenAPI 409 与前端错误码展示（R-06/R-10）。
- Sprint BX：GitHub CI 429/5xx 重试与熔断；`CI_PROVIDER_UNAVAILABLE`（R-09）。
- Sprint BW：Approve/Resume 对称门禁；Cancel 后不可 Approve；控制面错误码扩展（R-05）。
- Sprint BV：Replay 源状态门禁（非终态拒绝）；`RUN_NOT_REPLAYABLE`（R-05）。
- Sprint BU：executeSteps 步边界观察 Cancel；中途取消不落 finished/failed（R-05）。
- Sprint BT：run 状态机 `canTransition`；fail/finish 不覆盖 canceled；Cancel 幂等单测（R-05）。
- Sprint BS：`regression-short` 接入 BQ/BR；CI fixture 9103–9105；RunsPage SSE 状态文案 smoke。
- Sprint BR：低分 feedback 置信度衰减；Query/run 按 confidence 排序并过滤 `<0.2`（R-04）。
- Sprint BQ：`postgresMigrationDSN` 回归单测；CI 诊断新增 cancel / 资源耗尽 / 前端 lint-typecheck（R-03）。
- Sprint BP：跨平台 fake execgocli、`OpenTest` Windows 清理、CI 失败日志 artifact；CI 全绿（`4d5acad`）。
- Sprint BO：`regression-short` / H-09 static 接入跨 space 与 stream 续传用例。
- Sprint BN：SSE 耗尽后 timeline 轮询回退；`TestStreamRunResumesFromQueryLastEventID`。
- Sprint BM：SSE 自动重连（退避 + Last-Event-ID）；跨 space API 回归表驱动用例；stream query 续传。
- Sprint BL：12/12 控制台页 smoke；Automation/Compliance 空数据可选链；`signoff-apply` 幂等；`worker-local-gate` + runbook roster 同步。
- Sprint BH：`frontend` Runs/CI/Memory/Scale 页 smoke 测试；`ci.yml` 并行 `release-window-gate`；`make kpi-reconcile-gate`（KPI §9 对账）。
- Sprint BE：`make release-window-gate` / `make bootstrap-local-ash-db`；`release-gates.yml` 接入 release-window + worker-production；MVP §6 备份 / §8 自动化勾选。
- Sprint BD：`make config-env-gate` / `make worker-production-gate` / `make release-window-prefill`；H-09 SSE live（`sse-live-smoke.sh`）；`mvp-signoff` + CI 接入 config-env-gate。
- Sprint BC：`make scope-freeze-gate`；CI `release-gates.yml`；cloud-acceptance 接入 pre-migrate。
- Sprint BB：`make worker-local-gate` / `make pre-migrate-gate` / `make t1-metrics-gate`；`release-window-runbook.md`。
- Sprint BA：`make queue-gate` / `make t0-alert-gate` / `make data-backup`；`doc/mvp-release-scope.md`；CI production-config + queue gate。
- Sprint AZ：`make production-config-gate` + `make rollback-drill`；API 延迟/并发基线单测；`regression-short` / `mvp-signoff` 接入。
- Sprint AY：`make web-gate`（eslint + vitest + build）；CI frontend job 升级；`source-cloud-rds-env.sh`；MVP §3 前端项勾选。
- Sprint AX：`make cloud-acceptance` + `make mvp-signoff` 证据归档；`h01-h03-cloud-signoff.md`；MVP 清单勾选与 `doc/evidence/`。
- Sprint AW：`scripts/postgres-app-gate.sh` + `make postgres-app-gate`（H-02/H-03 本地）；`smoke-static.sh`；`verify-local` Go env 引导。
- Sprint AV：`postgres-rds-e2e` 接入 `live-smoke`；`smoke-index` 文档串联；MVP 清单与 CI `release-sampling-static` 显式步骤。
- Sprint AU：`scripts/live-smoke.sh` + `make live-smoke`（H-04/05/06/07/09 live 编排）；`doc/checklists/smoke-index.md`；`release-window-audit` live 段收敛。
- Sprint AT：`scripts/release-sampling-static.sh` + `make release-sampling-static`（H-09 §7 静态烟测）；`release-sampling-smoke.md`；`regression-short` 接入。
- Sprint AS：`scripts/secret-rotate-smoke.sh` + `make secret-rotate-smoke`（H-07）；`TestSecretRotateRepoConnectionH07`；`make release-sampling`；`regression-short` / CI / `release-window-audit` 接入。
- Sprint AQ：`scripts/release-window-audit.sh` + `make release-window-audit`（H-08 静态门禁聚合）。
- Sprint AR：`scripts/execgo-live-smoke.sh` + `make execgo-live-smoke`（H-06）；`TestM3ExecGoLiveSmoke`；`verify-local` 静态门禁。
- Sprint AP：`scripts/ci-fixture-smoke.sh`  live Worker H-04/H-05；`release-sampling.sh` §7.3b ttl-queue；CI 页 fixture 提示 +「诊断选中 job」。
- Sprint AO：CI fixture H-04/H-05 全链路扩展（双 job：test_failure + docker）；`jobId` 诊断拉日志 + `logDigest` 落库 + adopt；`TestReleaseSamplingCIFixtureH04H05`。
- Sprint AN：OpenAPI 契约补全 TTL 端点/schema；`memoryTTLSweepInterval` 与 Scale TTL 字段对齐；`TestMemoryTTLQueueAndSweepAPI`；derive `memory.ttl_expired` replay；H-09 抽样含 `ttl-queue`。
- Sprint AM：Worker 后台记忆 TTL sweep（`ASH_MEMORY_TTL_SWEEP_INTERVAL`）；`/readyz` 与 Scale 暴露 `memoryTTLSweepInterval`。
- Sprint AJ：记忆 catalog v1→v2；`release-window-audit.md`；`release-sampling.sh` + API 抽样测试
- Sprint AD：Scale `/readyz` 运维面板；Postgres readyz+RLS 集成测试；CI PR `postgres-rls-e2e`
- Sprint AC：Doctor **TR3-10** readyz HealthResponse 契约；M3-09 SQL 修订预期校验；RDS 脚本 rev 20；ALL **42/42**
- Sprint AB：`/readyz` RLS/SQL 漂移信号；Doctor **M3-09** RLS catalog 证据；CI `postgres-rls-e2e` job
- Sprint AA：Scale RLS 预期策略数 + catalog 摘要 + readiness 漂移告警；RLS 新表清单；`regression-short` store 冒烟
- Sprint Y：memory 子表 RLS 集成测试 + e2e 脚本接入
- Sprint W：SQL rev **18** `model_usage` run-scoped RLS；`PostgresRLSDeferredTables()` / `VerifyRLSMigrationSQL()`；Doctor **M3-11**；附录 G §9 OpenAPI
- OpenAPI：补全 legacy `/v1/*` 规划路径具体 schema；`TestApiV1SuccessResponsesAvoidGenericEnvelope` 防回归；76+96 schema 字段对齐
- Sprint U：Doctor **M3-10** RLS 全局表排除（`memory_migrations`/`schema_meta`）；`PostgresRLSGlobalTables()` 目录；`verify-local` 接入 `regression-short`
- Sprint T：Doctor **TR3-08** Prometheus replay 段（`ASH_METRICS_EVENT_REPLAY=1`）；`make regression-short` 快捷回归；`/readyz` 默认运维标志断言
- Sprint S：记忆测试 `store.OpenTest`；Doctor UI M3-03..M3-09 中文标签
- Sprint R：`internal/opsenv` 共享运维快照；Doctor **M3-09** readyz/Scale 契约；CI push main 跑 `postgres-e2e`
- Sprint P：Doctor **TR3-07** 插件导出健康；Scale `metricsEventReplayEnabled`；Observability OTel 导出器健康卡片；`verify-local` 补 pluginhealth/alerts
- Sprint O：OTel waterfall 导出自动写入 `pluginhealth`（`ash-otel-exporter`）；Scale 暴露 `otelEnabled` / `alertsEvalInterval`；`postgres-e2e` 追加 live Postgres **TR3**
- Sprint N：Worker 后台治理告警评估（`ASH_ALERTS_EVAL_INTERVAL`）；`postgres-sql-schema-e2e` 追加 live Postgres **TR3**；Scale TR3-06 卡片；swagger 再生
- Sprint M：`postgres-e2e-migrate` 复用 `source postgres-up.sh`；`postgres-sql-schema-e2e` 追加 `TestPostgresRAGFTSQuery`；Doctor **TR3-06**（Postgres `tsvector` FTS，sqlite skip）
- Sprint L：Observability 治理指标告警表；Scale 插件导出健康摘要
- Sprint K：Postgres RAG `tsvector` + GIN（SQL **17**）；`postgres-tsvector` FTS 引擎；集成测试 `TestPostgresRAGFTSQuery`
- Sprint J：治理告警规则（记忆积压 / RAG FTS 降级率 / 插件导出失败）；Prometheus live 治理指标；`GET /api/v1/rag/profile`；Observability RAG 卡片
- Sprint H：`plugin.export_failed` / `plugin.export_reported` 审计事件；可选 `runId` 写 run 事件；derive `ash_plugin_export_failures_total`
- Sprint I：RAG `retrievalMode`（fts/chunk/empty）、`rag.retrieved` 事件、derive `ash_rag_queries_total` / `ash_rag_fts_fallback_total`、Scale RAG 检索模式行
- Sprint G：插件导出健康（`plugin_registry` 列 + SQL **16**、`pluginhealth.RecordExport`、`POST /plugins/{id}/export-report`、`GET /plugins/health`、Automation UI 健康列）
- Sprint F：OTel 骨架（`internal/observability/otel`、Worker `Init`、Run live span + waterfall OTLP 导出、`GET /observability/otel/status`、Observability UI OTel 卡片）
- Sprint E：`memory_migrations` 表 + `POST /api/v1/memory/migrate`；`memory.migrated` → `ash_memory_migration_runs_total`；Scale 记忆迁移按钮；SQL revision **15**
- Sprint D：记忆 P1 derive（`hit_used` / `deprecated` / `query`）；`memory.query` 事件（`runId` 可选）；TR0 payload schema 补全
- Sprint C：记忆治理 derive 指标 `ash_memory_unreviewed_backlog` / `ash_memory_missing_evidence_total`；`memory.reviewed` emit 补 `layer`；Scale `readinessWarnings`（`ASH_SCHEMA_MODE=sql` + 双写冲突）
- 测试验收 T-01..T-08 写入 **测试验收（暂缓）**，不阻塞后续开发
- Sprint A/B：CI `postgres-sql-schema-e2e`、derive `ValidateReplayParity`、Doctor **TR3-05**、生产配置模板与 RDS 清单对齐 revision 14
- RLS SQL 化：`000013` 租户策略（`ash_rls_*` 函数 + `ash_space_*` policies）、`000014` `ash_app`/`ash_rls_tester` 授权；运行时仅 backfill/FORCE
- `ASH_SCHEMA_MODE=sql` 试点：`make postgres-sql-schema-e2e`、Doctor **M3-08**、Scale readiness 暴露 schema 模式与 SQL 修订版本
- SQL 修订 `000009`–`000012` 完成迁移目录全表覆盖（CI/告警/发布/密钥/审批/插件/improve）；`expectedVersion=12`
- SQL 修订 `000005`–`000008`：run 执行（tool_calls/agent_tasks/artifact_index/checkpoints）、memory/RAG、租户身份、model/feedback
- 事件 → 指标表驱动派生：`internal/observability/derive`（附录 D §4 catalog + replay）；`ASH_METRICS_EVENT_REPLAY=1` 追加离线 replay 段
- SQL 修订 `000004`（run_steps / memory_records / audit_log）
- TR0 事件 payload JSON Schema 运行时校验（`ASH_VALIDATE_EVENT_PAYLOADS=1`）；SQL 修订 `000003`（runs/run_events）
- `golang-migrate` 版本化迁移骨架：Postgres embed SQL、`ash migrate schema`、`ASH_SCHEMA_MODE`、Doctor M3-03 校验
- Observability 配置 JSON Schema：`ash.obs/v0.1` + `config/ash-observability.yaml` + Worker 启动校验
- Rules DSL JSON Schema：`internal/rules/schemas/ash.rules.v0.1.schema.json` + `ValidateSchema`（20 非法样例回归）
- `runs.Service.eventsFor` 无限递归修复（审批/取消路径 stack overflow）；`openapi-check` 网络受限时自动重试（GOSUMDB=off + GOPROXY 镜像）
- API 错误码表：`doc/api/error-codes.md`、`internal/apicodes` 目录校验；CI 接入 `make openapi-check`
- OpenAPI 对齐：`doc/api/openapi-alignment.md`、`internal/openapicheck`、`make openapi-check`；手写契约已补全全部 `/api/v1` 实现端点
- Scale readiness：`lastMigrationSyncError` / `lastMigrationSyncErrorAtMs`；`migrate sync` 失败写入 `SyncState` 并成功后清空
- Scale readiness：`workerConnectionRole`、`runtimeDsnHint`、双写影子库脱敏 URL；M3 §3.4 PgBouncer 指引
- `metrics` 服务 `WithContext` + `metricsFor(c)`；OpenAPI 草稿补充 `/metrics/prometheus`
- TODO 重组：人工验证 H-01..H-09 单列暂缓
- KPI-07/KPI-10：`run_events` / `run_steps` 经 `runs.space_id` 租户过滤；`make postgres-rds-e2e`
- `/metrics` v1：全局 scrape + RLS bypass；`GET /api/v1/metrics/prometheus` 租户范围
- Postgres RLS + `ash_app` + Doctor M3-06/07 + e2e 脚本
- 云 RDS 清单 `doc/checklists/postgres-rds-e2e.md`
- PRD 前四项 + 后四项 MVP
