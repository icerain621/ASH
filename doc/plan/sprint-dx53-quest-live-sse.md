# Sprint DX53 — Quest 看板 live / plan.* SSE（v3.1 · T2）

> **方案 A：** `GET /quest/stream` Space 级 SSE；命中 `plan.*`/`run.*` 刷新看板；**无新表**  
> **状态：** ✅ 完成  
> **设计：** [`../../docs/superpowers/specs/2026-09-08-v31-dx53-quest-live-sse-design.md`](../../docs/superpowers/specs/2026-09-08-v31-dx53-quest-live-sse-design.md)

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX53-1 | `ListAfterSpace` + `/quest/stream` | ✅ |
| DX53-2 | `useQuestBoardStream` + QuestPage | ✅ |
| DX53-3 | 测试 + OpenAPI + CHANGELOG / TODO | ✅ |

## 验收

```bash
go test ./internal/events/ -run ListAfterSpace -count=1
cd frontend && npm test -- --run src/services/sse/questBoardStream.test.tsx src/pages/QuestPage.test.tsx
make openapi-check
```

## 交付摘要

- `events.ListAfterSpace` + `GET /api/v1/quest/stream`（live-only，Last-Event-ID 续传）
- Quest 看板状态条 + 事件驱动 invalidate；断连后 board 轮询回退
