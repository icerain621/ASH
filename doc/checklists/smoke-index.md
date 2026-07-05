# 烟测索引（H-04–H-09）

> 本地静态烟测无需 Worker；live 烟测需 `ASH_WORKER_URL`。发布窗口全量审计见 [`release-window-audit.md`](release-window-audit.md)（H-08）。

## 一键入口

| 场景 | 命令 | 说明 |
|------|------|------|
| 快捷回归 | `make regression-short` 或 `make smoke-static` | Doctor M3/TR3 + API + CI fixture + H-07/H-09 静态 |
| 发布审计（静态） | `make release-window-audit` | H-08：Doctor ALL/M3/TR3 + regression-short + openapi |
| Postgres ash_app（本地） | `make postgres-app-gate` | H-02/H-03：Docker + schema 后 M3-06/07 + readyz |
| 业务抽样（静态） | `make release-sampling-static` | H-09 §7 API 单测 |
| 业务抽样（静态+live） | `ASH_WORKER_URL=... make release-sampling-smoke` | H-09 |
| Live Worker 联调 | `ASH_WORKER_URL=... make live-smoke` | H-04/05/06/07/09 live 编排 |
| 本地全量 | `bash scripts/verify-local.sh` | regression-short + Doctor CLI + openapi + 可选 Postgres RLS |

## H 清单对照

| # | 项 | 静态 | Live |
|---|-----|------|------|
| H-04 | CI runs sync | `TestCISyncRunsWithFixture` / `TestReleaseSamplingCIFixtureH04H05` | [`ci-fixture-smoke.sh`](../scripts/ci-fixture-smoke.sh) |
| H-05 | CI jobs / 诊断 | 同上 | 同上（含 `jobId` 诊断） |
| H-06 | ExecGo | `TestM3ExecGoLiveSmoke` | [`execgo-live-smoke.md`](execgo-live-smoke.md) |
| H-07 | 密钥轮换 | `TestSecretRotateRepoConnectionH07` | [`secret-rotate-smoke.md`](secret-rotate-smoke.md) |
| H-08 | 发布审计 | [`release-window-audit.md`](release-window-audit.md) | 设 `ASH_WORKER_URL` 触发 §7 live |
| H-09 | 业务抽样 §7 | [`release-sampling-smoke.md`](release-sampling-smoke.md) | [`release-sampling.sh`](../scripts/release-sampling.sh) |

## Live Worker 环境变量

| 变量 | 用途 |
|------|------|
| `ASH_WORKER_URL` | Worker 基址（live-smoke / release-sampling 必需） |
| `ASH_CI_FIXTURE=1` | Worker 进程启用 CI fixture（H-04/05/07 live） |
| `ASH_EXECGO_E2E=1` | ExecGo live doctor M3-05（H-06） |
| `ASH_SPACE_ID` | 默认 `local` |
| `ASH_AUTH_HEADER` | 可选 Bearer |

## 相关

- [`postgres-rds-e2e.md`](postgres-rds-e2e.md) — H-01～H-03、云 RDS §7
- [`postgres-production-config.md`](postgres-production-config.md) — 生产 env 与轮换 SOP
