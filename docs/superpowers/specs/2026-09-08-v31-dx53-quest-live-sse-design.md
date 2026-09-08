# v3.1 DX53 — Quest 看板 live / plan.* SSE

> Status: **implemented** (2026-09-08)  
> Program: T2 Quest · Approach **A** · **no new tables**

## Goals

- Space-scoped SSE `GET /api/v1/quest/stream` for Quest kanban live refresh
- Push `plan.*` + key `run.*` / `gate.waiting_approval` from existing `run_events`
- QuestPage invalidates `quest-board` on matching events; show stream status

## API

`GET /api/v1/quest/stream`

- Auth / space: same as `/quest/board`
- `text/event-stream`; resume via `Last-Event-ID` / `lastEventId` (event id → resume after that `ts`)
- Fresh connect (no resume): start from **now** (live-only, no backlog dump)
- Poll interval ~500ms (mirror `streamRun`)

## Backend

- `events.Service.ListAfterSpace(spaceID, afterTS, limit)`  
  `run_id IN (goal_plans.id ∪ runs.id for space)` AND type ∈ board-live set, `ts > afterTS`, order by `ts, id`
- Live types: `plan.created|approved|started|rejected`, `run.started|finished|failed|canceled`, `gate.waiting_approval`

## Frontend

- `useQuestBoardStream(spaceId, { onEvent })` → EventSource `/api/v1/quest/stream`
- On board-live types → `invalidateQueries(["quest-board", spaceId])`
- Fallback: after reconnect exhaustion, `refetchInterval`-style board poll (not timeline)
- Status chip on Quest header (`quest-stream-status`)

## Non-goals

- Extending `/runs/{planId}/stream` for draft plans  
- New tables / Doctor bumps  
- DX54 freeze / signoff

## Tests

- `events` ListAfterSpace unit test  
- API stream smoke (optional httptest)  
- FE hook vitest + QuestPage wiring
