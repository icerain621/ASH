# Sprint DX50 — Diff 审查闭环（v3.1 · T2）

> **方案：** `POST .../diff/reject`（file|all）；全量→Cancel；文件→记录；Quest Approve 继续门禁；**无新表**  
> **状态：** ✅ 完成 · **无新表**  
> **设计：** [`../../docs/superpowers/specs/2026-09-08-v31-dx50-diff-reject-design.md`](../../docs/superpowers/specs/2026-09-08-v31-dx50-diff-reject-design.md)

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX50-1 | `diffreview.Reject` + 单测 | ✅ |
| DX50-2 | API + OpenAPI / swagger / apicodes | ✅ |
| DX50-3 | Quest UI reject + approve continue | ✅ |
| DX50-4 | sprint / TODO / CHANGELOG | ✅ |

## 验收

```bash
go test ./internal/diffreview/ -count=1
cd frontend && npm test -- --run src/pages/QuestPage.test.tsx
make openapi-check
```
