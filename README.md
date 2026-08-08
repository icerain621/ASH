# ASH

<img width="100%" height="100%" alt="image" src="https://github.com/user-attachments/assets/de87f7be-5510-4288-9066-8a4c03b56785" />

全称：Advanced Species Of Human（人类高阶物种），旧世界顶尖超人工智能，灯塔克洛托系统的原型本体，末日玛娜生态诞生的关键推手之一。旨在做一个像一个ASH一样强大的AI助手。


## 目录结构

```txt
ash/
  cmd/worker|cli    # Go Worker / CLI
  internal/         # 业务包（api/runs/memory/store/...）
  frontend/         # Web 控制台（Vite+React → /ui/）
  scenarios/        # Rules DSL YAML
  doc/              # 归属索引见 doc/README.md
    design/         # 设计：PRD / HLD / ARCH / M3
    plan/           # 计划：PLAN / TODO / 范围 / 风险 / KPI
    progress/       # 进度：发布清单（+ checklists/ evidence/）
    appendices/ api/
    archive/        # 历史草案（勿作实现依据）
  scripts/          # 工具与门禁脚本
```

**文档归属**：[`doc/README.md`](doc/README.md) · **计划**：[`doc/plan/PLAN-进度与里程碑.md`](doc/plan/PLAN-进度与里程碑.md) · **待办**：[`doc/plan/TODO.md`](doc/plan/TODO.md)

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

**Web 控制台**：`http://localhost:8080/ui/`（Runs + SSE、Memory 评审、Repo/CI 诊断、KPI 指标、Doctor）

## 验证（TR0）

```bash
make doctor
# 或
go run ./cmd/cli doctor --suite TR0 --format md --out doctor-tr0.md
```

## ExecGo / Codex Agent 执行面

ASH M0 默认使用 `execgo_codex` 执行器：ASH 负责编排、事件、证据、审计、门禁和产物，真实 coding step 通过 ExecGo 提交给 Codex CLI。首次使用前先显式安装并检查执行面：

```bash
# 克隆并构建 github.com/iammm0/execgo 与 execgo-runtime 到 .ash/execgo
make execgo-bootstrap

# 在运行 ASH 的 shell 中加入 bootstrap 输出的环境变量后，启动 ExecGo / execgo-runtime
# 然后检查桥接可用性
make execgo-health
```

常用配置：

| 变量 | 默认值 | 说明 |
|---|---|---|
| `ASH_AGENT_EXECUTOR` | `execgo_codex` | Run 的 agent 执行器，测试/本地演示可临时设为 `static` |
| `ASH_CODEX_BIN` | `codex` | Codex CLI 可执行文件 |
| `ASH_CODEX_BYPASS_SANDBOX` | `0` | 设为 `1` 才会向 Codex CLI 传递危险的 bypass 参数；默认不绕过 Codex 审批/沙箱 |
| `EXECGO_EXECGOCLI` | `execgocli` | `execgocli` 路径；bootstrap 会打印 `.ash/execgo/execgo/bin/execgocli` |
| `EXECGO_URL` | `http://127.0.0.1:8080` | ExecGo 控制面地址 |
| `EXECGO_RUNTIME_URL` | `http://127.0.0.1:18080` | execgo-runtime 数据面地址 |

如果 `execgocli health` 或 `execgocli tools` 失败，`make execgo-health` 会区分 CLI 缺失、控制面不可达、runtime/tools 不可用或 JSON 输出异常；`ash run --agent execgo_codex` 会进入失败态并返回 `AGENT_BRIDGE_UNAVAILABLE`，不会静默降级为 stub。显式 live 验证可运行：

```bash
ASH_EXECGO_E2E=1 go run ./cmd/cli doctor --suite M3 --format md --agent execgo_codex
```

## CI / Repo 诊断 / KPI / 闭环治理

仓库已提供 GitHub Actions 门禁：

- `.github/workflows/ci.yml`：PR 与 `main` push 执行 `go test ./...`、`make test`、Doctor M3/ALL static、`make web-build`。
- `.github/workflows/postgres-e2e.yml`：手动或 nightly 执行 `make postgres-e2e`。
- 云 RDS 生产切换：见 [`doc/checklists/postgres-rds-e2e.md`](doc/checklists/postgres-rds-e2e.md)；`make postgres-rds-e2e`。

Repo/CI 诊断使用 GitHub Actions v1 provider。先通过 Secrets 保存 token，再创建 repo connection；token 只允许通过 `secretId` 引用，API 会拒绝明文 token。

```bash
curl -X POST http://localhost:8080/api/v1/repo/connections \
  -H "Content-Type: application/json" \
  -d '{"provider":"github","owner":"iammm0","repo":"ASH","secretId":"sec_xxx"}'

curl -X POST http://localhost:8080/api/v1/ci/failures/diagnose \
  -H "Content-Type: application/json" \
  -d '{"connectionId":"repo_conn_xxx","logText":"--- FAIL: TestExample"}'
```

KPI 看板入口：

- API：`GET /api/v1/metrics/overview?period=day&from=...&to=...`
- 控制台：`/ui/metrics`

PRD 后四项闭环入口：

- CI 诊断控制台：`/ui/ci`；API 支持 `GET /api/v1/ci/jobs`、`GET /api/v1/ci/diagnoses`、`POST /api/v1/ci/diagnoses/{id}/adopt|dismiss`。
- 反馈闭环：`/ui/feedback`；API 支持 `GET /api/v1/feedback`、`PATCH /api/v1/feedback/{id}`，低分反馈会写入站内 `AlertEvent`。
- 可观测与告警：`/ui/observability`；API 支持 alert rules、manual evaluate、trace 查询；租户 Prometheus 快照见 `GET /api/v1/metrics/prometheus`；运维 scrape 仍用全局 `GET /metrics`（RLS 开启时 bypass）。
- 发布治理：`/ui/releases`；API 支持 release record、MVP checklist、gate 评估和 rollback drill 记录。v1 只记录灰度/回滚策略与证据，不执行真实生产部署；人工模板见 `scripts/release-notes-template.md`、`scripts/rollback-drill-template.md`。

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
- `POST /api/v1/repo/connections` / `GET /api/v1/repo/connections` — GitHub repo connection
- `GET /api/v1/ci/runs` — CI run 摘要
- `GET /api/v1/ci/jobs` — CI job 摘要，可 `sync=true`
- `POST /api/v1/ci/failures/diagnose` — CI 失败确定性诊断
- `GET /api/v1/ci/diagnoses` — CI 诊断历史
- `POST /api/v1/ci/diagnoses/:id/adopt|dismiss` — 记录诊断采纳/驳回
- `GET /api/v1/feedback` / `PATCH /api/v1/feedback/:id` — 反馈列表与处理状态
- `GET /api/v1/metrics/overview` — KPI 看板聚合
- `GET /api/v1/metrics/prometheus` — 租户范围 Prometheus 文本（`observability:read`）
- `GET /api/v1/observability/alerts` — 告警事件
- `GET/PUT /api/v1/observability/alert-rules` — 告警规则
- `POST /api/v1/observability/alerts/evaluate` — 手动评估告警
- `GET /api/v1/observability/trace/:traceId` — trace 关联查询
- `POST /api/v1/releases` / `GET /api/v1/releases` — 发布治理记录
- `GET/PATCH /api/v1/releases/:id/checklist` — MVP 发布清单
- `POST /api/v1/releases/:id/gate` — 发布门禁评估
- `POST /api/v1/releases/:id/rollback-drills` — 回滚演练记录

带 `runId` 的 memory 操作会追加到该 run 的 **SSE 事件流**（`GET /api/v1/runs/:runId/stream`）：

- `memory.candidate_created` / `memory.review_requested`（L1+）
- `memory.reviewed` / `memory.deprecated`（deprecate 时）
- `memory.hit_used`

## 文档索引

- 变更记录：`CHANGELOG.md`
- 文档总索引：[`doc/README.md`](doc/README.md)
- 计划与进度：[`doc/plan/PLAN-进度与里程碑.md`](doc/plan/PLAN-进度与里程碑.md)
- 待办短清单：[`doc/plan/TODO.md`](doc/plan/TODO.md)
- 早期草案归档：[`doc/archive/`](doc/archive/README.md)
