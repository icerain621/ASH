# Sprint DX18：Landlock 沙箱 POC（v2.5 · 方案 C）

> **方案：** 已批准 **C** = Linux Landlock executor + `ASH_SANDBOX_LANDLOCK` 选型 + Doctor M4-SBX-04  
> **Goal:** `DefaultRouter` 可选 `landlock` executor；非 Linux 跳过；`make sandbox-smoke` 可选段  
> **状态：** 本批完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX18-1 | `landlock` 包：`Available()` + Linux executor | ✅ |
| DX18-2 | `DefaultRouter` 接线 + `ASH_SANDBOX_LANDLOCK` | ✅ |
| DX18-3 | Doctor M4-SBX-04（非 Linux / 无内核支持 → skip pass） | ✅ |
| DX18-4 | `sandbox-smoke` landlock 单元测 + 可选 `ASH_SANDBOX_LANDLOCK=1` 段 | ✅ |

## 行为

- `ASH_SANDBOX_LANDLOCK=1` 且 Linux 且 `landlock.Available()` 时，isolated 路径可选用 Landlock executor
- 非 Linux 或内核无 Landlock：Doctor **pass（skipped）**；smoke 仅打印 availability，不要求 e2e 成功
- Windows / Git Bash：`make sandbox-smoke` 与 `ASH_SKIP_SANDBOX=1` 保持绿

## 退出标准

- [x] `go test ./internal/sandbox/... ./internal/sandbox/landlock/... -count=1`
- [x] `ASH_SKIP_SANDBOX=1 make sandbox-smoke`（Windows 友好）
- [x] `doc/checklists/sandbox-smoke.md` + `v2.5-release-scope.md` 草案水位（**未冻结**）
