# Landlock + seccomp e2e 证据清单（DX27）

> 目标：在 **Linux** 上证明 Landlock FS deny 与最小 seccomp deny-list 行为。  
> **不接 E2B**；远程 Executor 仅保留接口扩展注释（见 `internal/sandbox.Executor`）。

## 本地 / CI

```bash
# 单元 + 探测（跨平台；非 Linux 跳过 e2e）
make sandbox-smoke

# Linux 证据段（写入 doc/evidence/sandbox-landlock-e2e-latest.md）
ASH_SANDBOX_E2E=1 make sandbox-smoke
```

| 变量 | 说明 |
|------|------|
| `ASH_SANDBOX_E2E=1` | 跑 `TestE2E*` 并刷新证据摘要 |
| `ASH_SANDBOX_LANDLOCK=0` | 关闭 landlock 优先（e2e 仍可直接测 Executor） |
| `ASH_SANDBOX_SECCOMP=0` | 跳过 seccomp；`TestE2ESeccomp*` skip |
| `ASH_SKIP_SANDBOX=1` | 跳过 Docker 段（不影响 landlock e2e） |

## 断言覆盖

| 用例 | 期望 |
|------|------|
| `TestE2ELandlockAllowsRepoRootRead` | 可读 `$HOME/.../repo` 内文件 |
| `TestE2ELandlockDeniesOutsideRepoRoot` | 不可读同级 `secret/`（且不在 `/tmp` 白名单下） |
| `TestE2ESeccompDeniesMountSyscall` | `SYS_MOUNT` 探针被 kill / 非零退出；软跳过可记 skip |

非 Linux 或 `Available()=false`：用例 **Skip**；smoke 仍绿，证据标注 `platform=skipped`。

## 手工核对（可选）

1. Doctor：`go run ./cmd/cli doctor --suite M4` → M4-SBX-04/05  
2. 打开证据文件确认三行 PASS/SKIP  
3. 确认仓库无 E2B SDK / 云计费客户端依赖  

## 相关

- [`sandbox-smoke.md`](sandbox-smoke.md)
- [`../evidence/sandbox-landlock-e2e-latest.md`](../evidence/sandbox-landlock-e2e-latest.md)
- [`../plan/sprint-dx27-sandbox-evidence.md`](../plan/sprint-dx27-sandbox-evidence.md)
