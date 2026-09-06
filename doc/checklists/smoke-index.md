# 烟测索引（H-04–H-09）

> 本地静态烟测无需 Worker；live 烟测需 `ASH_WORKER_URL`。发布窗口全量审计见 [`release-window-audit.md`](release-window-audit.md)（H-08）。

## 一键入口

| 场景 | 命令 | 说明 |
|------|------|------|
| 快捷回归 | `make regression-short` 或 `make smoke-static` | Doctor M3/TR3 + API + CI fixture + H-07/H-09 静态 |
| 发布审计（静态） | `make release-window-audit` | H-08：Doctor ALL/M3/TR3 + regression-short + openapi |
| Postgres ash_app（本地） | `make postgres-app-gate` | H-02/H-03：Docker + schema 后 M3-06/07 + readyz |
| Postgres 本地 RDS（H-01 dry-run） | `make postgres-local-rds-e2e` | Docker + `.ash/ash.db`：sqlite→Postgres + M3-04（Sprint CW） |
| 业务抽样（静态） | `make release-sampling-static` | H-09 §7 API 单测 |
| 业务抽样（静态+live） | `ASH_WORKER_URL=... make release-sampling-smoke` | H-09 |
| Live Worker 联调 | `ASH_WORKER_URL=... make live-smoke` | H-04/05/06/07/09 live 编排（`ASH_CI_LIVE=1` 时追加真 GitHub） |
| P1 真 CI + ExecGo | `make p1-live-credibility` | H-04/05 真 GitHub + H-06；缺 env 则 SKIP |
| 本地全量 | `bash scripts/verify-local.sh` | regression-short + Doctor CLI（`--agent static`）+ openapi + 可选 Postgres；`ASH_VERIFY_LOCAL_RDS=1` 时含 local-rds |
| 前端门禁 | `make web-gate` | eslint + vitest（全页 smoke）+ build |
| SSE 浏览器 E2E（可选） | `make sse-browser-e2e` | Playwright + 临时 Worker：`/ui/runs` → 已连接 + 事件行（P2-4） |
| R-08 跨 space | `make r08-cross-space-gate` | API 越权回归 + RLS 静态；Docker/URL 时跑 Postgres live |
| 插件签名 | `make plugin-sign-smoke` | HMAC 单测 + `ash plugin-sign` 往返（Sprint CR） |
| 生产配置 | `make production-config-gate` | dev-secret / CHANGE_ME 拦截 |
| 配置核对 | `make config-env-gate` | 模板 + production/scope gate |
| 回滚演练 | `make rollback-drill` | 发布 API drill + 基线 + Doctor ALL |
| 队列治理 | `make queue-gate` | TTL sweep 消费 + 洁净告警 |
| T+0 告警 | `make t0-alert-gate` | 洁净空间 evaluate 无 alert |
| T+1 指标 | `make t1-metrics-gate` | KPI overview + feedback + §9 对账 |
| KPI 对账 | `make kpi-reconcile-gate` | overview ↔ derive replay（§9） |
| Worker 本地 live | `make worker-local-gate` | 临时 Worker + live-smoke |
| RAG Hybrid（DX9） | `make rag-hybrid-smoke` | RebuildSymbols + Hybrid Query + 回退 |
| RAG LSP（DX35） | `make rag-lsp-smoke` | session / hover / def / refs / expandRefs；假 gopls；写 evidence |
| Waker（DX6/DX12） | `make waker-smoke` | queue/sweep + duties/status；可选 live curl |
| 发布范围冻结 | `make scope-freeze-gate` | MVP + v2 + … + v2.8 scope 结构校验 |
| 发布窗口门禁 | `make release-window-gate` | §8 快速聚合（~2min；含 backup/T+0/T+1） |
| 本地就绪 | `make local-readiness-gate` | release-window + worker live（~4min） |
| §11 签字回填 | `make signoff-apply` | `config/signoff.env` → 证据 + 清单 |
| 签字验收 | `make signoff-gate` | 四人 + 范围冻结齐全 |
| 发布窗口预填 | `make release-window-prefill` | 空模板（无跑门禁） |
| H-03 JWT Worker | `make worker-production-gate` | 生产-like 密钥 + dev-login live |
| 迁移前 | `make pre-migrate-gate` | backup + migrate plan |

## H 清单对照

| # | 项 | 静态 | Live |
|---|-----|------|------|
| H-04 | CI runs sync | `TestCISyncRunsWithFixture` / `TestReleaseSamplingCIFixtureH04H05` | fixture: [`ci-fixture-smoke.sh`](../scripts/ci-fixture-smoke.sh)；真 GitHub: [`ci-live-smoke.sh`](../scripts/ci-live-smoke.sh) + [`p1-live-credibility.md`](p1-live-credibility.md) |
| H-05 | CI jobs / 诊断 | 同上 | 同上（含 `jobId` 诊断） |
| H-06 | ExecGo | `TestM3ExecGoLiveSmoke` | [`execgo-live-smoke.md`](execgo-live-smoke.md)（证据落盘） |
| — | Sandbox isolated (DX2) | KPI-19 / sandbox policy tests | [`sandbox-isolated.md`](sandbox-isolated.md) |
| — | CI webhook (DS) | HMAC → diagnose → optional autoRun | [`ci-webhook.md`](ci-webhook.md) |
| — | Session RPC (DT) | `/agents/sessions` + `ash session rpc` | [`session-rpc.md`](session-rpc.md) |
| — | Evolve KPI + mobile (DU) | KPI-17~19 + `/ui/m/reviews` | [`evolve-kpi-mobile.md`](evolve-kpi-mobile.md) |
| — | v2 signoff (DV) | scope freeze + `make v2-signoff` | [`v2-signoff.md`](v2-signoff.md) |
| — | ACP provider (DW) | ProbeACP + harness `acp_sdk` fallback | [`acp-provider.md`](acp-provider.md) |
| — | ACP ↔ Session (DX3) | `providerKind` + `session.linked` | [`acp-session.md`](acp-session.md) |
| — | ACP 契约烟测 (DX4) | `make acp-smoke` + turn 转发 | [`acp-smoke.md`](acp-smoke.md) |
| — | v2.1 signoff (DX5) | scope freeze + `make v2.1-signoff` | [`v2.1-signoff.md`](v2.1-signoff.md) |
| — | Waker (DX6/DX12) | stale Run queue/sweep + duties/status | [`waker-smoke.md`](waker-smoke.md) |
| — | v2.2 signoff (DX8) | scope freeze + `make v2.2-signoff` | [`v2.2-signoff.md`](v2.2-signoff.md) |
| — | RAG Hybrid (DX9) | RebuildSymbols + Hybrid Query RRF | [`rag-hybrid-smoke.md`](rag-hybrid-smoke.md) |
| — | v2.3 signoff (DX11) | scope freeze + `make v2.3-signoff` | [`v2.3-signoff.md`](v2.3-signoff.md) |
| — | v2.4 signoff (DX14) | scope freeze + `make v2.4-signoff` | [`v2.4-signoff.md`](v2.4-signoff.md) |
| — | v2.5 signoff (DX19) | scope freeze + `make v2.5-signoff` | [`v2.5-signoff.md`](v2.5-signoff.md) |
| — | skill-pack (DX22) | 私有签名 pack verify/install | [`skill-pack-smoke.md`](skill-pack-smoke.md) |
| — | v2.6 signoff (DX24) | scope freeze + `make v2.6-signoff` | [`v2.6-signoff.md`](v2.6-signoff.md) |
| — | v2.7 signoff (DX30) | scope freeze + `make v2.7-signoff` | [`v2.7-signoff.md`](v2.7-signoff.md) |
| — | RAG LSP (DX35) | session / hover / def / refs smoke + evidence | [`rag-lsp-smoke.md`](rag-lsp-smoke.md) |
| — | v2.8 signoff (DX36) | scope freeze + `make v2.8-signoff` | [`v2.8-signoff.md`](v2.8-signoff.md) |
| H-07 | 密钥轮换 | `TestSecretRotateRepoConnectionH07` | [`secret-rotate-smoke.md`](secret-rotate-smoke.md) |
| H-08 | 发布审计 | [`release-window-audit.md`](release-window-audit.md) | 设 `ASH_WORKER_URL` 触发 §7 live |
| H-09 | 业务抽样 §7 | [`release-sampling-smoke.md`](release-sampling-smoke.md) | [`release-sampling.sh`](../scripts/release-sampling.sh) |

## Live Worker 环境变量

| 变量 | 用途 |
|------|------|
| `ASH_WORKER_URL` | Worker 基址（live-smoke / release-sampling 必需） |
| `ASH_CI_FIXTURE=1` | Worker 进程启用 CI fixture（H-04/05/07 live） |
| `ASH_CI_LIVE=1` | 真 GitHub sync/diagnose（禁 fixture；需 token 或 connection） |
| `ASH_EXECGO_E2E=1` | ExecGo live doctor M3-05（H-06） |
| `ASH_ACP_ENDPOINT` | ACP 控制面基址（Harness `acp_sdk`） |
| `ASH_ACP_E2E=1` | ACP liveGateHints |
| `ASH_SPACE_ID` | 默认 `local` |
| `ASH_AUTH_HEADER` | 可选 Bearer |

## 相关

- [`postgres-rds-e2e.md`](postgres-rds-e2e.md) — H-01～H-03、云 RDS §7
- [`h01-h03-cloud-signoff.md`](h01-h03-cloud-signoff.md) — H-01～H-03 云验收签字
- [`postgres-production-config.md`](postgres-production-config.md) — 生产 env 与轮换 SOP
- [`../evidence/README.md`](../evidence/README.md) — `make mvp-signoff` / `make cloud-acceptance` 证据
