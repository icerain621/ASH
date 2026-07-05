# 业务抽样 Smoke（H-09）

> 对应 [`postgres-rds-e2e.md`](postgres-rds-e2e.md) §7。本地/CI 静态测试无需 Worker；live 抽样需运行中 Worker。

## 前置

1. 静态：`go test` 即可（`ASH_AUTH_MODE=dev` 由测试自设）
2. Live：Worker 可达（默认 `http://127.0.0.1:8080`）
3. CI fixture 联调（§7.5）：`ASH_CI_FIXTURE=1` 启动 Worker

## 步骤

| # | 动作 | 期望 |
|---|------|------|
| 1 | `make release-sampling-static` | `TestReleaseSamplingH09` / `SSE` / `CrossSpace` / `CIFixture` pass |
| 1b | 等价 | `go test ./internal/api/ -run 'TestReleaseSamplingH09|TestReleaseSamplingSSE|TestReleaseSamplingH09CrossSpaceMemoryDenied|TestReleaseSamplingCIFixtureH04H05' -count=1` |
| 2 | `make release-sampling` | live Worker §7.0–7.7 curl 抽样成功 |
| 2b | CI fixture 扩展 | `ASH_CI_FIXTURE=1` + `make ci-fixture-smoke`（H-04/H-05） |

## §7 覆盖对照

| 节 | 静态测试 | Live 脚本 |
|----|----------|-----------|
| 7.1 Run | `TestReleaseSamplingH09` | `release-sampling.sh` create run |
| 7.2 SSE | `TestReleaseSamplingSSE` | 手动 `GET /runs/{id}/stream` |
| 7.3 Memory | `TestReleaseSamplingH09` + `CrossSpace` | memory candidate/review/query |
| 7.3b TTL queue | `TestReleaseSamplingH09` | `GET /memory/ttl-queue` |
| 7.4 KPI | `TestReleaseSamplingH09` | `GET /metrics/overview` |
| 7.5 CI diagnose | `TestReleaseSamplingH09` + `CIFixtureH04H05` | diagnose + fixture smoke |
| 7.6 合规导出 | `TestReleaseSamplingH09` | `POST /compliance/export` |
| 7.7 Scale | `TestReleaseSamplingH09` | `GET /scale/readiness` |

## 与 release-window-audit 集成

`make release-window-audit` 静态段经 `regression-short` 覆盖 H-09；设置 `ASH_WORKER_URL` 时调用 `make live-smoke`。索引见 [`smoke-index.md`](smoke-index.md)。

## 相关

- [`release-window-audit.md`](release-window-audit.md) — H-08 发布窗口
- [`smoke-index.md`](smoke-index.md) — H-04–H-09 烟测索引
- [`ci-fixture-smoke`](../scripts/ci-fixture-smoke.sh) — H-04/H-05 live
- [`secret-rotate-smoke.md`](secret-rotate-smoke.md) — H-07
