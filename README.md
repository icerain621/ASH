# ASH

<img width="100%" height="100%" alt="ASH" src="https://github.com/user-attachments/assets/de87f7be-5510-4288-9066-8a4c03b56785" />

**Advanced Species Of Human** — 交付驱动的 AI 编排平台：用自然语言 / Issue / Quest 发起真实交付任务，在可审计场景里跑到四件套（diff · 测试 · 发布说明 · 回滚），Human 只在门禁点介入。

| | |
|---|---|
| **栈** | Go **1.26** Worker/CLI · Vite + React 控制台 · SQLite 本地 / Postgres+RLS 规模化 |
| **模块** | `github.com/ash-repwiki/ash` |
| **当前水位** | **v4.0 Auth 已冻结**（DX61–DX66）· Doctor ALL **57** / M4 **10** · SQL rev **33** · RLS **51** |
| **下一轨** | **v5 双核管控**（薄交互 GV01 起：visibility / 意图 / Agent使用⇄评审管控）并行于 **v4.1** 企业 Agentic |
| **Tag** | 门禁绿后 **人工**打标（不自动）；`v4.0.0` 待签字后切 |

> 排期真相源：[`doc/plan/PLAN-进度与里程碑.md`](doc/plan/PLAN-进度与里程碑.md) · 短待办：[`doc/plan/TODO.md`](doc/plan/TODO.md) · 程序：[`doc/plan/v4.x-program.md`](doc/plan/v4.x-program.md)

---

## 它做什么

- **场景编排**：Rules DSL（`scenarios/`）驱动 feature / hotfix / security_patch 等交付流  
- **Run 生命周期**：创建 → 执行 → 人审门禁 → resume/replay → Artifacts；SSE 事件流  
- **Quest**：Goal → Plan → Approve → Run；看板 + Diff 审查  
- **记忆与 RAG**：候选评审、引用门禁；FTS + 可选向量 / LSP 符号  
- **安全执行面**：ToolBus 策略、Sandbox（landlock / docker / 可选 remote）、Harness Profile  
- **多租户与 Auth**：Org/Space、权限矩阵；OIDC（含 RS256/JWKS）、独立 refresh、jti 吊销、可选控制台强制登录  
- **治理闭环**：CI 诊断、KPI、反馈告警、发布治理记录、Doctor 探针套件  

定位与 In/Out 详见 [`doc/design/PRD-需求文档.md`](doc/design/PRD-需求文档.md)。

---

## 仓库结构

```txt
ash/
  cmd/worker|cli     # HTTP Worker · CLI（doctor / run / quest …）
  internal/          # api · runs · memory · rag · authz · sandbox · store …
  frontend/          # Web 控制台（构建后由 Worker 挂载 /ui/）
  scenarios/         # Rules DSL YAML
  doc/               # 设计 / 计划 / 清单 / OpenAPI / 证据（见 doc/README.md）
  docs/superpowers/  # Sprint 设计规格（specs）
  scripts/           # 门禁与烟测脚本
  .github/workflows/ # CI · Postgres E2E
```

架构图（可交互 HTML）：[`doc/diagrams/archify/`](doc/diagrams/archify/README.md)

---

## 快速开始

Windows 推荐 **Git Bash**。Go 代理受限时可：`export GOPROXY=https://goproxy.cn,direct`

```bash
cd /c/Go_Work/src/ash   # 或你的仓库根
make tidy
make run                # Worker → http://localhost:8080
```

| 入口 | URL |
|------|-----|
| 控制台 | http://localhost:8080/ui/ |
| Swagger | http://localhost:8080/docs |
| 健康 | http://localhost:8080/readyz |
| 前端热更 | `make web-dev` → http://127.0.0.1:5173/ui/ |

本地库默认 SQLite（常见路径 `.ash/ash.db`）。Postgres / RLS：`source scripts/postgres-up.sh`，再 `make postgres-sql-schema-e2e` 等（见 `.cursor/rules` / `doc/design/M3-多租户与Postgres演进.md`）。

### 验证

```bash
make doctor             # TR0 等 Doctor 套件
make test
make openapi-check
make v4.0-signoff       # v4.0 冻结门禁（含 Auth smoke）
bash scripts/verify-local.sh   # 若存在：本地综合烟测
```

### ExecGo / Codex 执行面（可选）

默认 agent 执行器多为 `execgo_codex`：ASH 负责编排与证据，coding step 经 ExecGo 交给 Codex CLI。

```bash
make execgo-bootstrap   # 构建到 .ash/execgo
# 按输出 export PATH / EXECGO_* 后启动 ExecGo
make execgo-health
```

| 变量 | 默认 | 说明 |
|------|------|------|
| `ASH_AGENT_EXECUTOR` | `execgo_codex` | 演示可临时 `static` |
| `ASH_CODEX_BIN` | `codex` | Codex CLI |
| `ASH_CODEX_BYPASS_SANDBOX` | `0` | `1` 才传危险 bypass |
| `EXECGO_URL` / `EXECGO_RUNTIME_URL` | `8080` / `18080` | 控制面 / 数据面 |

桥接不可用时进入失败态（`AGENT_BRIDGE_UNAVAILABLE`），不静默降级。Live：`ASH_EXECGO_E2E=1 go run ./cmd/cli doctor --suite M3 --format md --agent execgo_codex`。

### Auth 速查（v4.0）

| 能力 | 要点 |
|------|------|
| OIDC | `ASH_OIDC_*`；id_token **RS256/JWKS**（兼 HS256） |
| Session | access + **独立 refresh**；`POST /auth/sessions/refresh` 双票轮换 |
| 吊销 | jti 绑定 + grace（`ASH_AUTH_TOKEN_GRACE_SEC`） |
| 控制台门闸 | `ASH_CONSOLE_AUTH_REQUIRED=1` → 无 token 跳登录 |
| Device | Space Auth Sessions mint；`make device-session-smoke` |

范围与签字：[`doc/plan/v4.0-release-scope.md`](doc/plan/v4.0-release-scope.md) · [`doc/checklists/v4.0-signoff.md`](doc/checklists/v4.0-signoff.md)

---

## 常用 Make 目标

| 目标 | 用途 |
|------|------|
| `make run` / `test` / `tidy` | 启动 / 单测 / 依赖 |
| `make doctor` | Doctor（默认 TR0） |
| `make swagger` / `openapi-check` | 生成 Swagger · 对齐手写 OpenAPI |
| `make web-build` / `web-dev` | 控制台构建 / 开发服 |
| `make postgres-e2e` / `postgres-rls-e2e` | 本地 Postgres / RLS |
| `make sandbox-smoke` / `rag-*-smoke` / `skill-pack-smoke` | 域烟测 |
| `make v3.2-signoff` / `v4.0-signoff` | 代际冻结门禁 |
| `make scope-freeze-gate` | 范围文档冻结检查 |

完整列表见根目录 `Makefile`。

---

## CI 与云切流

- [`.github/workflows/ci.yml`](.github/workflows/ci.yml)：`go test`、Doctor、`web-build` 等  
- [`.github/workflows/postgres-e2e.yml`](.github/workflows/postgres-e2e.yml)：手动 / nightly Postgres  
- 云 RDS：[`doc/checklists/postgres-rds-e2e.md`](doc/checklists/postgres-rds-e2e.md) · `make postgres-rds-e2e`  

Repo/CI 诊断走 GitHub Actions provider；token 仅经 Secrets `secretId` 引用，API 拒明文。

控制台入口示例：`/ui/quest` · `/ui/ci` · `/ui/metrics` · `/ui/observability` · `/ui/releases` · `/ui/feedback`

---

## API 与契约

完整契约以 **Swagger**（`/docs`）与手写 OpenAPI 为准，勿依赖本 README 枚举端点：

- [`doc/api/openapi-ash-v1.yaml`](doc/api/openapi-ash-v1.yaml)  
- [`doc/api/error-codes.md`](doc/api/error-codes.md)  
- 变更对齐：`make swagger && make openapi-check`

SSE：`GET /api/v1/runs/:runId/stream`（含 memory.* 等事件）。

---

## 文档索引

| 文档 | 说明 |
|------|------|
| [`CHANGELOG.md`](CHANGELOG.md) | 变更事实 |
| [`doc/README.md`](doc/README.md) | 文档归属总索引 |
| [`doc/design/`](doc/design/) | PRD / HLD / ARCH / M3 |
| [`doc/plan/v4.x-program.md`](doc/plan/v4.x-program.md) | v4.x 四代分冻 |
| [`doc/plan/agentic-roadmap-to-qoder.md`](doc/plan/agentic-roadmap-to-qoder.md) | Agentic 路线图 |
| [`doc/checklists/`](doc/checklists/) · [`doc/evidence/`](doc/evidence/) | 验收清单与门禁证据 |
| [`doc/archive/`](doc/archive/README.md) | 历史草案（勿作实现依据） |

---

## 初始化远程仓库（可选）

若尚未绑定 GitHub remote：

```bash
bash scripts/init-repo.sh
# 或：GITHUB_REMOTE=git@github.com:YOUR_USER/ash.git bash scripts/init-repo.sh
```

无 `gh` 时脚本会完成本地 commit 并提示网页建空库后手动 `git push`。PowerShell：`scripts/init-repo.ps1`。

| 变量 | 说明 |
|------|------|
| `GH_REPO_NAME` | 默认 `ash` |
| `GH_OWNER` | 用户或组织 |
| `GITHUB_REMOTE` | 完整 URL，跳过 `gh repo create` |
