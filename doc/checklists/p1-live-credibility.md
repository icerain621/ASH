# P1 可信度：真 GitHub CI（H-04/05）+ ExecGo live（H-06）

> 脚本就绪；缺环境则 **SKIP**（不阻断）。硬失败仅在「已开启」轨道上出现。

## 一键

```bash
# 可选证据目录：.ash/evidence/execgo-live-*
make p1-live-credibility
```

## Track A — H-04/H-05 真 GitHub

| 变量 | 说明 |
|------|------|
| `ASH_CI_LIVE=1` | **必须** opt-in |
| `ASH_WORKER_URL` | 运行中 Worker（**勿**带 `ASH_CI_FIXTURE`） |
| `ASH_REPO_CONNECTION_ID` | 已有 connection（优先） |
| 或 `ASH_GITHUB_TOKEN` + `ASH_GITHUB_OWNER` + `ASH_GITHUB_REPO` | 现场建 secret/connection |

```bash
ASH_CI_LIVE=1 ASH_WORKER_URL=http://127.0.0.1:8080 \
  ASH_GITHUB_TOKEN=... ASH_GITHUB_OWNER=... ASH_GITHUB_REPO=... \
  make ci-live-smoke
```

| 结果 | 含义 |
|------|------|
| `OK CI live` | sync + diagnose 成功，且无 fixture 数据 |
| `SKIP ... not_enabled` | 未设 `ASH_CI_LIVE=1` |
| `SKIP ... no_credentials` | 无 connection/token |
| `FAIL` | Worker 仍开 fixture / API 错误 |

## Track B — H-06 ExecGo live

| 变量 | 说明 |
|------|------|
| `ASH_EXECGO_E2E=1` | **必须** |
| ExecGo + Codex | 见 [`execgo-live-smoke.md`](execgo-live-smoke.md) |
| `ASH_WORKER_URL` | 可选；探测 `/providers/agent` + readyz |

```bash
ASH_EXECGO_E2E=1 make execgo-live-smoke
# 可选：ASH_WORKER_URL=http://127.0.0.1:8080
```

## 与 fixture / live-smoke 关系

| 目标 | 用途 |
|------|------|
| `make ci-fixture-smoke` | 无 GitHub 的 H-04/05（Worker `ASH_CI_FIXTURE=1`） |
| `make ci-live-smoke` | **真** GitHub（禁 fixture） |
| `make live-smoke` | 编排；`ASH_CI_LIVE=1` 时追加 ci-live |
| `make execgo-live-smoke` | H-06 + 证据落盘 |

## 相关

- `scripts/ci-live-smoke.sh` · `scripts/execgo-live-smoke.sh` · `scripts/p1-live-credibility.sh`
- smoke 索引：[`smoke-index.md`](smoke-index.md)
