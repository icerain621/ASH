# Sprint VX33 — Notifier 适配器缝 / Notifier seam

> **前置 / Prerequisite：** VX32  
> **状态 / Status：** ✅  
> **做法 / Approach：** `internal/notify`：`Notifier` + `null` / `log`；`ASH_NOTIFIER`（默认 null；未知值 fail-closed 到 null）。无 IM SDK、无新表。  
> `internal/notify`: `Notifier` + `null`/`log`; `ASH_NOTIFIER` (default null; unknown → null). No IM SDK, no new table.

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| VX33-1 | 包与接口 + 单测 | ✅ |
| VX33-2 | 配置选择 `ASH_NOTIFIER=null|log`（默认 null） | ✅ |
| VX33-3 | CHANGELOG | ✅ |

## 验收 / Verify

```bash
go test ./internal/notify/... -count=1
```
