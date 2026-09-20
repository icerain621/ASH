# Sprint VX34 — Waker → Notifier / Waker notify hook

> **前置 / Prerequisite：** VX33  
> **状态 / Status：** ✅  
> **做法 / Approach：** `persistDutyRun` 成功后投递 `waker.duty.<status>`；默认 null；`ASH_NOTIFIER=log` 可观察。不接 IM。无新表。  
> After a successful `persistDutyRun`, emit `waker.duty.<status>`; null by default; `ASH_NOTIFIER=log` is observable. No IM. No new table.

## 如何开启 log 投递 / Enable log notifier

```bash
export ASH_NOTIFIER=log
# then run worker / waker duties as usual
```

未知值（如 IM 通道名）会 **fail-closed 到 null**，不会悄悄接外部 SDK。  
Unknown values (e.g. IM channel names) fail closed to null — no silent external SDKs.

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| VX34-1 | duty 完成钩子 + 单测 | ✅ |
| VX34-2 | 文档：如何开 log notifier | ✅ |
| VX34-3 | CHANGELOG | ✅ |

## 验收 / Verify

```bash
go test ./internal/waker/... ./internal/notify/... -count=1
```
