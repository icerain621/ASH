# Sprint VX32 — RPC Doctor 探针 / RPC Doctor probe

> **前置 / Prerequisite：** VX31  
> **状态 / Status：** ⬜  
> **做法 / Approach：** Doctor 增加 RPC 可达探针（进程内 ServeRPC 或 CLI）；不可用环境 skip-pass。无新表。  
> Add a Doctor probe for RPC reachability (in-process ServeRPC or CLI); skip-pass when unavailable. No new table.

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| VX32-1 | Doctor case + 计数更新 | ⬜ |
| VX32-2 | TR/ALL suite 对齐 | ⬜ |
| VX32-3 | CHANGELOG | ⬜ |

## 验收 / Verify

```bash
go test ./internal/doctor/... -run TestTR3Suite -count=1
```
