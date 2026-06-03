# ASH Agent Guide

本文件给所有编码代理使用，用于快速理解 ASH 仓库、执行常用命令，并在修改时保持一致的工程习惯。

## 项目概览

ASH 是一个面向 AI 助手编排的项目，当前主要由 Go 后端 Worker/CLI 与 React Web 控制台组成。

- 后端：Go，Gin，GORM，SQLite，Swagger。
- 前端：Vite，React 18，TypeScript，TanStack Router/Query/Table，lucide-react。
- 文档：`doc/` 下保存 PRD、架构、迭代计划、API/事件/状态设计等材料。
- 场景：`scenarios/` 下保存运行场景 YAML。

## 目录速览

- `cmd/worker/`：Worker 服务入口。
- `cmd/cli/`：CLI 入口，包含 doctor/TR0 等诊断能力。
- `internal/api/`：HTTP API、Swagger、静态资源服务、SSE 相关接口。
- `internal/runs/`：运行生命周期、控制、执行逻辑。
- `internal/memory/`：记忆候选、评审、查询、命中审计。
- `internal/rag/`：RAG 查询服务。
- `internal/rules/`：场景规则解析、加载与校验。
- `internal/store/`：SQLite/GORM 数据模型与数据库初始化。
- `internal/toolbus/`：内置工具、Git 工具等执行适配。
- `frontend/src/`：Web 控制台源码。
- `doc/`：产品、架构、API、数据库、里程碑等文档。

## 常用命令

在仓库根目录执行：

```bash
make tidy
make test
make run
make doctor
make swagger
```

前端开发：

```bash
make web-dev
make web-build
```

等价前端命令：

```bash
cd frontend
npm install
npm run dev
npm run build
```

服务地址：

- 后端 Worker：`http://localhost:8080`
- Swagger UI：`http://localhost:8080/docs`
- 后端托管控制台：`http://localhost:8080/ui/`
- Vite 开发控制台：`http://127.0.0.1:5173/ui/`

## 开发约定

- 优先使用根目录 `Makefile` 中已有入口，不要随意新增平行脚本。
- 后端改动优先保持现有包边界：API 层处理 HTTP，业务逻辑放在对应 `internal/*` 服务包中，数据库模型放在 `internal/store/`。
- 前端改动优先沿用 `frontend/src/app/`、`frontend/src/pages/`、`frontend/src/modules/` 的现有结构。
- 新增 API 时同步考虑 Swagger 注解、前端 API 客户端、测试和 README/文档。
- 新增运行场景时优先放入 `scenarios/`，并确认 `internal/rules/` 的解析/校验逻辑是否需要更新。
- 修改持久化模型时检查 SQLite 初始化、测试数据、迁移兼容性和相关 API 响应模型。
- 不要回退或覆盖用户已有改动；修改前先看 `git status --short`。

## 验证建议

根据改动范围选择验证：

- Go 后端通用验证：`make test`
- TR0 诊断验证：`make doctor`
- Swagger 变更验证：`make swagger`
- 前端类型与构建验证：`make web-build`
- 端到端本地体验：先 `make run`，再访问 `/docs` 或 `/ui/`

如果只改文档，可以不跑完整测试，但要说明未运行测试的原因。

## Git 协作

- 提交前先确认工作区中哪些改动属于本次任务。
- 如果有与本次任务无关的用户改动，不要暂存或回退它们。
- 按功能拆分 commit；提交信息使用中文且保持 UTF-8。
- 修改完成后询问是否需要提交。

## 文档优先级

当实现与文档冲突时，按以下顺序判断：

1. 当前代码中的实际行为。
2. 根目录 `README.md` 与对应模块 README。
3. `doc/` 中的架构、API、事件协议、里程碑文档。
4. 本文件中的通用约定。

