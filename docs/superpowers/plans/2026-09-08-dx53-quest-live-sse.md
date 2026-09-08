# DX53 Quest Live SSE Implementation Plan

> **For agentic workers:** space stream + board invalidate; no new tables.

**Goal:** Live Quest kanban via `GET /api/v1/quest/stream`.

**Architecture:** Poll `run_events` for space plan/run IDs; FE invalidates board query.

---

### Task 1: Backend ListAfterSpace + quest stream

**Files:** `internal/events/service.go`, `service_test.go`, `internal/api/quest.go`, `handlers.go`, openapi, apicodes

### Task 2: Frontend hook + QuestPage

**Files:** `frontend/src/services/sse/questBoardStream.ts`, test, `QuestPage.tsx` / test

### Task 3: Docs

sprint / TODO / CHANGELOG / design status
