# Sprint DX：Sandbox POC Implementation Plan

> **Goal:** danger+`off` 拒绝；`workspace-write` 下 process/Docker 可跑命令；路径不出 repoRoot；`make sandbox-smoke`。  
> **状态：** DX-1~DX-5 ✅

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX-1 | ResolveSandboxMode 风险地板 + Authorize | ✅ |
| DX-2 | process Executor + PathWithinRoot | ✅ |
| DX-3 | Docker Executor + Dockerfile | ✅ |
| DX-4 | DefaultRouter + runs 拒绝 / runtime.command Dispatch | ✅ |
| DX-5 | sandbox-smoke + CHANGELOG/TODO | ✅ |

## 验收

```bash
go test ./internal/sandbox/... ./internal/sandbox/process/... -count=1
ASH_SKIP_SANDBOX=1 make sandbox-smoke
# 有 Docker 时：make sandbox-smoke
```

## 备注

- Windows：`isolated` 以 process jail 为主（HLD-03）
- Docker 镜像：`deploy/sandbox/Dockerfile` → `ash-sandbox-runner:dev`
- `ASH_SKIP_SANDBOX=1` 跳过 Docker 段
