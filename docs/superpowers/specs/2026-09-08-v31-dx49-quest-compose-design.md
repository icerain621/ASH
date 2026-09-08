# v3.1 DX49 — Quest-native Goal/Plan compose + approve

> Status: **implemented** (2026-09-08 · DX49)  
> Program: T2 Quest · [`doc/plan/v3.x-program.md`](../../plan/v3.x-program.md)  
> Approach: **Option 1** — Quest page inline Goal→Plan→Approve; Runs quest-pane retained

## Problem

Goal/Plan compose lives on `/ui/runs` (`quest-pane`); `/ui/quest` only shows Plan cards and tells users to approve on Runs. That splits the T2 workbench.

## Goals

- Users can create a Plan from NL Goal and Approve/Reject **on QuestPage** without leaving `/ui/quest`.
- Selecting a draft Plan on the board opens the same preview + actions.
- On approve, select the new Run for Diff review and refresh the board.
- **No new tables**; Doctor stays **ALL 57 / M4 10**; reuse existing `from-goal` / `plans/{id}/approve|reject` APIs.

## Non-goals (DX49)

- `plan.*` SSE / live board refresh (→ DX53)
- Diff file/full reject loop (→ DX50)
- Quest waiting_approval run-control / artifacts pane (→ DX51)
- Webhook/session Plan unification (→ DX52)
- Removing Runs `quest-pane` (keep both until later)

## Design

### UI

1. **Compose pane** on QuestPage (above or beside board): Goal + repoRoot +「生成 Plan」— same UX as Runs, `data-testid` prefixed `quest-wb-*` to avoid collision with Runs `quest-*`.
2. **Plan preview**: scenario/version/routeReason/steps JSON; draft → Approve / Reject.
3. **Board Plan click**: `GET /runs/plans/{planId}` (if not already loaded) → show preview; draft enables Approve/Reject (replace toast「在 Runs 页批准」).
4. **After approve**: if response includes `runId` (or follow existing GoalPlan shape), `setSelectedRunId(runId)`, invalidate `quest-board`, show success message.

### API client

Reuse `frontend/src/modules/runs/api/runs.api.ts`:
- `createRunFromGoal`
- `approveGoalPlan` / `rejectGoalPlan`
- Add `getGoalPlan(planId)` if missing (thin GET wrapper).

### Tests

Extend `QuestPage.test.tsx`:
- Renders compose controls
- Mock from-goal → preview
- Mock approve → board invalidate / selected run

### Docs / watermark

- `doc/plan/v3.1-release-scope.md` (draft until DX54 freeze)
- `doc/plan/sprint-dx49-quest-compose.md`
- TODO / CHANGELOG

## Constraints

| # | Rule |
|---|------|
| E5 | Prefer **no new tables** |
| E6 | Doctor **57 / 10** |
| API | No OpenAPI change if only FE + existing GET plan |
| Tag | No auto `v3.1.0` |

## Success

- From `/ui/quest` alone: Goal → draft Plan → Approve → Run appears on board / Diff selectable
- Vitest green; no schema / Doctor count bump
