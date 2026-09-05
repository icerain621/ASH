# Sandbox 烟测（Sprint DX / DX18）

```bash
make sandbox-smoke
# 无 Docker / Windows：ASH_SKIP_SANDBOX=1 make sandbox-smoke
# 可选 Landlock 段：ASH_SANDBOX_LANDLOCK=1 make sandbox-smoke
```

| 覆盖 | 说明 |
|------|------|
| `./internal/sandbox/` | Authorize、Router、ResolveSandboxMode |
| `./internal/sandbox/process/` | process executor + path jail |
| `./internal/sandbox/docker/` | Docker executor（包测） |
| `./internal/sandbox/landlock/` | `Available()` + 非 Linux 拒绝（**DX18**） |
| `./internal/runs/` | danger+off 拒绝 + harness 路由事件 |

## Docker 段（可选）

| 变量 | 说明 |
|------|------|
| `ASH_SKIP_SANDBOX=1` | 跳过镜像 build/run；单元测仍跑 |
| `ASH_SANDBOX_IMAGE` | 默认 `ash-sandbox-runner:dev` |

无 `docker` CLI 或 daemon 不可用时脚本 **exit 0**（process-only OK）。

## Landlock 段（可选 · DX18）

设 `ASH_SANDBOX_LANDLOCK=1` 时，在单元测之后、Docker 段之前：

| 平台 | 行为 |
|------|------|
| 非 Linux | 打印 `skipped (non-Linux)`；**不**跑 Landlock e2e |
| Linux | 打印 `landlock available=true|false`；内核无支持时 **non-fatal** 提示 |
| Linux + available | 额外 `go test ./internal/sandbox/landlock/` |

Doctor **M4-SBX-04** 与 smoke 一致：非 Linux / 无内核 Landlock → pass（skipped）。

## 相关

- [`../plan/sprint-dx-sandbox-implementation.md`](../plan/sprint-dx-sandbox-implementation.md)
- [`../plan/sprint-dx18-landlock.md`](../plan/sprint-dx18-landlock.md)
- [`sandbox-isolated.md`](sandbox-isolated.md)
