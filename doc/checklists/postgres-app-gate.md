# Postgres ash_app 门禁（H-02 / H-03 本地）

> 在 Docker Postgres 已迁移 schema 的前提下，验证 RLS、`ash_app` 运行时连接与 `/readyz` 契约。云 RDS 全链路见 [`postgres-rds-e2e.md`](postgres-rds-e2e.md)。

## 前置

1. `make postgres-up`
2. 已应用 schema（任选其一）：
   - `make postgres-sql-schema-e2e`
   - `go run ./cmd/cli migrate schema up --postgres "$ASH_DATABASE_URL"`

## 步骤

| # | 动作 | 期望 |
|---|------|------|
| 1 | `make postgres-app-gate` | RLS 集成 + Doctor **M3-06/07** pass + readyz 契约 |
| 1b | 等价 | `bash scripts/postgres-app-gate.sh` |
| 2 | 全量迁移对照 | `make postgres-e2e`（含 M3-04 migrate verify） |

## 环境变量（脚本自动设置）

| 变量 | 值 |
|------|-----|
| `ASH_DATABASE_URL` | `postgres://ash:ash@127.0.0.1:<port>/ash` |
| `ASH_DATABASE_APP_URL` | `postgres://ash_app:ash_app@127.0.0.1:<port>/ash` |
| `ASH_POSTGRES_RLS` | `1` |
| `ASH_POSTGRES_RLS_FORCE` | `1` |

## 相关

- [`smoke-index.md`](smoke-index.md) — H-04–H-09 烟测索引
- [`postgres-production-config.md`](postgres-production-config.md) — 生产 `ash_app` 配置
- [`release-window-audit.md`](release-window-audit.md) — 发布窗口 M3-06/07 live 断言
