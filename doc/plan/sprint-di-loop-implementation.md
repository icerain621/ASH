# Sprint DI：Loop Adapter Implementation Plan

> **For agentic workers:** 按任务勾选推进；设计已批准（薄 Adapter）。  
> **Goal:** Run 工具路径发出 `harness.*` 事件；沙盒 stub；事件不变量可测（M4-HAR-02 等价）。  
> **Architecture:** `internal/harness/loop` + `internal/sandbox` stub；接线 `runs.callTool*`，不改写主循环。  
> **Tech Stack:** Go / events / toolbus / existing harness.LoadActive  
> **状态：** DI-1~DI-5 ✅；DX 待开

## 文件地图

| 路径 | 职责 |
|------|------|
| `internal/sandbox/router.go` | `ResolveSandboxMode` + `NoopRouter` |
| `internal/harness/loop/adapter.go` | Loop Adapter 钩子 + HAR-02 |
| `internal/runs/execute.go` / `service.go` | 接线 turn/step/tool 钩子 |
| `internal/runs/loop_events_test.go` | 集成冒烟 |

## 任务板

| ID | Sprint | 任务 | 状态 |
|----|--------|------|------|
| DI-1 | DI | sandbox ResolveSandboxMode + NoopRouter | ✅ |
| DI-2 | DI | harness/loop Adapter + 单测 | ✅ |
| DI-3 | DI | 接线 callToolWithRetry + LoadActive | ✅ |
| DI-4 | DI | HAR-02 不变量测试 | ✅ |
| DI-5 | DI | PLAN/CHANGELOG/TODO | ✅ |
| DX-1 | DX | process/Docker Executor | 后续 |

### 勾选

- [x] DI-1~DI-5
- [ ] DX（替换 NoopRouter；danger+off 拒绝；`make sandbox-smoke`）

## 退出标准（已满足）

- `TestHarnessLoopEmitsRoutedAndCompleted`：`harness.tool.routed` + `harness.tool.completed`
- `go test ./internal/harness/... ./internal/sandbox/... ./internal/harness/loop/...` 绿
