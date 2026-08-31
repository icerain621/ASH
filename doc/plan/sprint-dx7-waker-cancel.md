# Sprint DX7：Waker cancel 安全闸门（v2.2 · 方案 C）

> **方案：** 已批准 **C** = 合入 PR 后强化 Waker：`action=cancel` + 三重闸门  
> **Goal:** 默认不可 cancel；需 `ASH_WAKER_ALLOW_CANCEL=1` + `confirm=CANCEL_STALE_RUNS` + `dryRun=false`  
> **状态：** 本批完成 · **无新表**

## 闸门

| # | 条件 |
|---|------|
| 1 | `ASH_WAKER_ALLOW_CANCEL=1` |
| 2 | body `confirm` = `CANCEL_STALE_RUNS` |
| 3 | `dryRun=false` 才真正改状态（`dryRun=true` 仅预览） |
| — | 后台 ticker **永不** cancel |

## 验收

```bash
go test ./internal/waker/ -run TestCancel -count=1
make waker-smoke
```

失败码：`WAKER_CANCEL_DENIED`
