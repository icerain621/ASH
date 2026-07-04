# 发布窗口审计门禁（H-08）

> 维护窗口切换 Postgres / 发版前由发布负责人勾选。证据归档至变更单附件。

## 切换前（T-24h）

| # | 项 | 命令 / 入口 | 证据 |
|---|-----|-------------|------|
| 1 | 云 RDS 全链路 | `make postgres-rds-e2e`（需 `ASH_DATABASE_URL`） | 脚本 stdout；Doctor M3/TR3/ALL md |
| 2 | 本地四门 Postgres（对照） | `make postgres-e2e` + `postgres-sql-schema-e2e` + `postgres-rls-e2e` | CI workflow 绿或本地日志 |
| 3 | 静态 Doctor | `go run ./cmd/cli doctor --suite ALL --agent static --format md` | **43/43** |
| 4 | 契约 | `make openapi-check` + `make regression-short` | openapicheck pass |
| 5 | Worker 配置核对 | `doc/checklists/postgres-production-config.md` | env 清单签字 |
| 6 | 密钥轮换（如适用） | §密钥轮换 SOP | 轮换前后 `/readyz` JSON |

## 切换窗口（T0）

1. 最终 `migrate copy` + `verify`（见 [`postgres-rds-e2e.md`](postgres-rds-e2e.md) §8）
2. 停旧 Worker → 设置 `ASH_DATABASE_APP_URL` / RLS env → 启新 Worker
3. `curl -s $ASH_WORKER_URL/readyz | jq .` — `dialect=postgres`，`liveGateHints` 含预期门禁
4. `go run ./cmd/cli doctor --suite M3 --require M3-04,M3-06,M3-07`（live）
5. 业务抽样 §7：`bash scripts/release-sampling.sh` 或 `go test ./internal/api/ -run TestReleaseSamplingH09`
6. CI fixture（可选）：Worker `ASH_CI_FIXTURE=1` + `bash scripts/ci-fixture-smoke.sh`

## 切换后（T+0～T+1）

| # | 观察项 | 阈值 / 动作 |
|---|--------|-------------|
| 1 | `/readyz` / Scale readiness | 持续 200；无 `readinessWarnings` 漂移 |
| 2 | `/metrics` + 告警 | 30–60 分钟无 P0 告警 |
| 3 | ExecGo（若启用） | `ASH_EXECGO_E2E=1` + [`execgo-live-smoke.md`](execgo-live-smoke.md) |
| 4 | CI sync（若启用） | 真实 token `sync=true` 或 `ASH_CI_FIXTURE=1` 联调记录 |

## 回滚触发（任一即执行）

与 [`postgres-rds-e2e.md`](postgres-rds-e2e.md) §8 一致：

- `migrate verify` 失败或 §7 抽样失败
- M3-04 / M3-06 / M3-07 live 失败
- 跨 `space_id` 数据泄漏
- `readyz` 非 postgres 或持续 5xx

回滚：Worker 回 SQLite；保留 RDS 快照与上述证据供复盘。

## 相关

- [`postgres-rds-e2e.md`](postgres-rds-e2e.md) — H-01～H-03、§7 抽样
- [`postgres-production-config.md`](postgres-production-config.md) — 生产 env 模板
- [`../11-mvp-release-checklist.md`](../11-mvp-release-checklist.md) — MVP 勾选清单
