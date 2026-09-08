# v3.1 DX50 — Diff 文件/全量拒绝 + Approve 继续

> Status: **implemented target** (2026-09-08)  
> Program: T2 Quest · Option: reuse `diff_review_comments` (**no new tables**)

## Goals

- Reject **current file** or **entire diff** from Quest Diff pane
- Persist rejection as review record (`side=reject`, `lineIndex=-1`)
- **Full reject** also **cancels** the run (stops delivery)
- **File reject** records only (run may continue / still Approving gates)
- When run is `waiting_approval`, Quest can **Approve** to resume (existing `/runs/{id}/approve`)

## API

`POST /api/v1/runs/{runId}/diff/reject`

```json
{ "scope": "file"|"all", "filePath": "path/for/file", "reason": "...", "actorId": "optional" }
```

Response includes comment view + `canceled` + run `status`.

## Non-goals

- Auto-regenerate patches after reject  
- New tables / Doctor bumps  
- waiting_approval artifact pane (DX51)
