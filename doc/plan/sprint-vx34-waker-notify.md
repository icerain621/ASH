# Sprint VX34 — Waker → Notifier / Waker notify hook

> **前置 / Prerequisite：** VX33  
> **状态 / Status：** ⬜  
> **做法 / Approach：** 选定一个 Waker duty 完成路径调用 Notifier（默认 null 无副作用；log 可观察）。不接 IM。无新表。  
> Hook one Waker duty completion path to Notifier (null by default; log optional). No IM. No new table.

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| VX34-1 | duty 完成钩子 + 单测 | ⬜ |
| VX34-2 | 文档：如何开 log notifier | ⬜ |
| VX34-3 | CHANGELOG | ⬜ |

## 验收 / Verify

```bash
go test ./internal/waker/... ./internal/notify/... -count=1
```
