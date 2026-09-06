# Sprint DX21：Landlock 默认收紧 + 最小 seccomp（v2.6）

> **方案：** 已批准 **B2** = Landlock 默认优先（`ASH_SANDBOX_LANDLOCK=0` 关闭）；E2B Out；最小 seccomp deny-list  
> **Goal：** isolated→landlock（Available 时）；M4-SBX-05；ALL **57** / M4 **10**；**无新表**  
> **状态：** ✅ 完成（代码）

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX21-1 | `landlockPreferred` 默认 ON；`=0` 退出 | ✅ |
| DX21-2 | Linux 子进程最小 seccomp deny-list；`ASH_SANDBOX_SECCOMP=0` 可关 | ✅ |
| DX21-3 | Doctor M4-SBX-05 + ALL/M4 水位 | ✅ |
| DX21-4 | router 测 + `sandbox-smoke` | ✅ |
| DX21-5 | sprint / CHANGELOG / TODO | ✅ |

## 约定

- `ASH_SANDBOX_LANDLOCK`：unset/`1`/`true` → 优先 landlock；`0`/`false`/`off` → 不选 landlock
- `ASH_SANDBOX_SECCOMP`：`0` 跳过 seccomp；默认尝试安装 deny-list（ENOSYS 等软跳过）
- Deny-list：`mount`/`umount2`/`pivot_root`/`swapon`/`swapoff`/`reboot`/module/`keyctl` 族
- 非 Linux：Doctor skip-pass；router 因 `Available()=false` 回退 docker/process

## 退出标准

- [x] Windows：`go test ./internal/sandbox/...` + `TestM4Suite` 绿
- [x] M4 **10**/10 · ALL **57**/57
- [x] 无 SQL/RLS bump
