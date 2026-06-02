# ASH

<img width="100%" height="100%" alt="image" src="https://github.com/user-attachments/assets/de87f7be-5510-4288-9066-8a4c03b56785" />

全称：Advanced Species Of Human（人类高阶物种），旧世界顶尖超人工智能，灯塔克洛托系统的原型本体，末日玛娜生态诞生的关键推手之一。旨在做一个像一个ASH一样强大的AI助手。


## 目录结构

```txt
ash/
  backend/          # Go Worker（Gin + GORM + SQLite，M0）
  doc/
    product/        # MVP 产品/工程文档（01–17 编号）
    design/         # v0.1 实现设计（PRD/HLD/ARCH/PLAN + appendices）
    api/            # OpenAPI 草稿
    db/             # MySQL DDL
  frontend/         # Web 控制台（M0 静态 UI → /ui/）
  reports/          # 调研与分析报告
  scripts/          # 工具脚本
```

## 首次整理目录

若从 `ash_repwiki/ash` 复制后尚未整理，在 **Git Bash** 中执行：

```bash
cd /c/Go_Work/src/ash
bash scripts/reorganize.sh
```

## 创建 Git 仓库并推送

在 **Git Bash**（推荐）或 PowerShell 中，于仓库根目录执行：

```bash
cd /c/Go_Work/src/ash
bash scripts/init-repo.sh
```

**没有安装 `gh`（GitHub CLI）也可以**：脚本会先完成本地 commit，再提示你在 GitHub 网页创建空仓库后手动 push。

手动推送示例（把 `YOUR_USER` 换成你的 GitHub 用户名）：

```bash
# 1. 在 https://github.com/new 创建名为 ash 的空仓库（不要勾选 README）
# 2. 添加远程并推送
git remote add origin git@github.com:YOUR_USER/ash.git
git push -u origin main
```

或指定 owner 后重跑脚本（仍无 gh 时需手动在网页建库）：

```bash
GITHUB_REMOTE=git@github.com:YOUR_USER/ash.git bash scripts/init-repo.sh
```

Windows PowerShell 等价命令：

```powershell
cd C:\Go_Work\src\ash
powershell -ExecutionPolicy Bypass -File scripts/init-repo.ps1
```

可选环境变量：

| 变量 | 说明 |
|---|---|
| `GH_REPO_NAME` | 仓库名，默认 `ash` |
| `GH_OWNER` | GitHub 用户名或组织名 |
| `GITHUB_REMOTE` | 完整 remote URL，跳过 `gh repo create` |

安装 `gh` 后（可选）：`gh auth login`，脚本可自动创建远程仓库。

## 启动 Worker

```bash
cd /c/Go_Work/src/ash
make tidy
make run
```

Swagger UI：`http://localhost:8080/docs`

**Web 控制台**：`http://localhost:8080/ui/`（Runs + SSE、Memory 评审、Doctor TR0）

## 验证（TR0）

```bash
make doctor
# 或
go run ./cmd/cli doctor --suite TR0 --format md --out doctor-tr0.md
```

## M0 新增 API

- `POST /api/v1/runs/:runId/resume` — 恢复 failed run
- `POST /api/v1/runs/:runId/replay` — 回放（`mode: exact|latest_memory`）
- `GET /api/v1/runs/:runId/artifacts` — 交付物 manifest
- `POST /api/v1/doctor/run` — 运行 TR0 验证
- `GET /api/v1/doctor/reports/:reportId` — 获取报告
- `POST /api/v1/memory/candidates` — 创建记忆候选（L1+ 需 evidence）
- `GET /api/v1/memory/candidates` — 列表（layer/status/repo）
- `POST /api/v1/memory/candidates/:id/review` — 评审 approve/reject/deprecate
- `GET /api/v1/memory/records/:id` — 单条详情
- `POST /api/v1/memory/query` — 检索已 approved 记忆
- `POST /api/v1/memory/hit-used` — 记录 run 命中审计

带 `runId` 的 memory 操作会追加到该 run 的 **SSE 事件流**（`GET /api/v1/runs/:runId/stream`）：

- `memory.candidate_created` / `memory.review_requested`（L1+）
- `memory.reviewed` / `memory.deprecated`（deprecate 时）
- `memory.hit_used`

## 文档索引

- 产品/MVP：`doc/product/README.md`（见 `doc/README.md`）
- 实现设计 v0.1：`doc/design/`（含 `appendices/`）
