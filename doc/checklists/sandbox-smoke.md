# Sandbox 烟测（Sprint DX / DX18 / DX21）

```bash
make sandbox-smoke
# 无 Docker / Windows：ASH_SKIP_SANDBOX=1 make sandbox-smoke
```

| 覆盖 | 说明 |
|------|------|
| `./internal/sandbox/` | Authorize、Router、ResolveSandboxMode；Landlock **默认优先**（DX21） |
| `./internal/sandbox/process/` | process executor + path jail |
| `./internal/sandbox/docker/` | Docker executor（包测） |
| `./internal/sandbox/landlock/` | `Available()` + 最小 seccomp；非 Linux stub（**DX18/DX21**） |
| `./internal/runs/` | danger+off 拒绝 + harness 路由事件 |

## Docker 段（可选）

| 变量 | 说明 |
|------|------|
| `ASH_SKIP_SANDBOX=1` | 跳过镜像 build/run；单元测仍跑 |
| `ASH_SANDBOX_IMAGE` | 默认 `ash-sandbox-runner:dev` |

无 `docker` CLI 或 daemon 不可用时脚本 **exit 0**（process-only OK）。

## Landlock 段（DX21 默认探测）

默认开启探测（除非 `ASH_SANDBOX_LANDLOCK=0`）：

| 平台 | 行为 |
|------|------|
| 非 Linux | 打印 `skipped (non-Linux)`；**不**跑 Landlock e2e |
| Linux | 打印 `available` + `seccomp`；额外 `go test ./internal/sandbox/landlock/` |

| 变量 | 说明 |
|------|------|
| `ASH_SANDBOX_LANDLOCK=0` | 关闭 landlock 优先与 smoke 探测段 |
| `ASH_SANDBOX_SECCOMP=0` | 子进程跳过 seccomp deny-list |

Doctor **M4-SBX-04/05**：非 Linux → pass（skipped）；默认 `LandlockPreferred=true`。

## 相关

- [`../plan/sprint-dx-sandbox-implementation.md`](../plan/sprint-dx-sandbox-implementation.md)
- [`../plan/sprint-dx18-landlock.md`](../plan/sprint-dx18-landlock.md)
- [`../plan/sprint-dx21-sandbox-harden.md`](../plan/sprint-dx21-sandbox-harden.md)
