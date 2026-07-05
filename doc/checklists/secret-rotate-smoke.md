# 密钥轮换 Smoke（H-07）

> 验证 Console `secretId` 引用在轮换后仍可用；本地/CI 可用 **`ASH_CI_FIXTURE=1`** 避免真实 GitHub API。

## 前置

1. Worker 运行中（默认 `http://127.0.0.1:8080`）
2. `ASH_SECRET_KEY` 已配置（加密存储）
3. 本地 fixture 联调：`ASH_CI_FIXTURE=1` 启动 Worker

## 步骤

| # | 动作 | 期望 |
|---|------|------|
| 1 | `go test ./internal/api/ -run TestSecretRotateRepoConnectionH07 -count=1` | 静态 H-07 pass（无需 Worker） |
| 2 | `ASH_CI_FIXTURE=1 make secret-rotate-smoke` | secret 创建 → connection → 轮换 → CI sync 仍成功 |
| 2b | 等价脚本 | `ASH_CI_FIXTURE=1 bash scripts/secret-rotate-smoke.sh` |
| 3 | 生产轮换后 | Doctor + `/readyz` 归档（见 `postgres-production-config.md` §密钥轮换） |

## 与 release-window-audit 集成

`make release-window-audit` 静态段包含 `TestSecretRotateRepoConnectionH07`；设置 `ASH_WORKER_URL` + `ASH_CI_FIXTURE=1` 时可选 live smoke。

## 相关

- API：`POST /api/v1/secrets/{secretId}/rotate`
- Console：Automation 页 Secrets 轮换按钮
- [`postgres-production-config.md`](postgres-production-config.md) — GitHub token 轮换 SOP
