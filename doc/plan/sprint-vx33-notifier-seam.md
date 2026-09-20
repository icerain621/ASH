# Sprint VX33 — Notifier 适配器缝 / Notifier seam

> **前置 / Prerequisite：** VX32  
> **状态 / Status：** ⬜  
> **做法 / Approach：** 定义 `Notifier` 接口 + `null` / `log` 适配器；不引入任何 IM SDK。无新表。  
> Define `Notifier` + `null`/`log` adapters; no IM SDKs. No new table.

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| VX33-1 | 包与接口 + 单测 | ⬜ |
| VX33-2 | 配置选择 `ASH_NOTIFIER=null|log`（默认 null） | ⬜ |
| VX33-3 | CHANGELOG | ⬜ |

## 验收 / Verify

```bash
go test ./internal/notify/... -count=1
```
