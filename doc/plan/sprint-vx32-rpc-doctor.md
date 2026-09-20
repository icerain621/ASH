# Sprint VX32 — RPC Doctor 探针 / RPC Doctor probe

> **前置 / Prerequisite：** VX31  
> **状态 / Status：** ✅  
> **做法 / Approach：** Doctor `M4-RPC-01`：进程内 `ServeRPC` 空闲 `session.start`；无 db 时 skip-pass。无新表。  
> Doctor `M4-RPC-01`: in-process ServeRPC idle session.start; skip-pass without db. No new table.

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| VX32-1 | Doctor case + 计数更新（M4 13 / ALL 63） | ✅ |
| VX32-2 | TR/ALL suite 对齐 | ✅ |
| VX32-3 | CHANGELOG | ✅ |

## 验收 / Verify

```bash
go test ./internal/doctor/... -run 'TestM4Suite|TestALLSuite' -count=1
```
