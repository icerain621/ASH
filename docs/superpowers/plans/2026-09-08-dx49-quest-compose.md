# DX49 Quest-native Goal/Plan compose Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let users compose and approve Goal Plans on `/ui/quest` using existing APIs, without new tables.

**Architecture:** Frontend-only QuestPage compose pane + board Plan selection; reuse `runs.api` from-goal/approve/reject; optional thin `getGoalPlan` client.

**Tech Stack:** React 18, TanStack Query, Vitest, existing Gin Goal APIs.

## Global Constraints

- No new SQL tables / RLS bump
- Doctor ALL 57 / M4 10 unchanged
- Runs `quest-pane` retained
- Chinese commit messages when committing
- Prefer `data-testid` prefix `quest-wb-*` for new Quest workbench controls

---

### Task 1: Goal plan GET client (if missing)

**Files:**
- Modify: `frontend/src/modules/runs/api/runs.api.ts`

- [ ] Add `getGoalPlan(planId: string)` → `GET /runs/plans/{planId}` returning `GoalPlan`
- [ ] Confirm types match existing `GoalPlan` (id, status, scenarioName, steps, runId if present)

### Task 2: Failing QuestPage tests for compose

**Files:**
- Modify: `frontend/src/pages/QuestPage.test.tsx`

- [ ] Mock `createRunFromGoal` / `approveGoalPlan` / `rejectGoalPlan` / `getGoalPlan` from runs.api
- [ ] Test: compose pane visible (`quest-wb-compose`)
- [ ] Test: route goal → plan preview (`quest-wb-plan-preview`)
- [ ] Test: approve draft plan calls approve API
- [ ] Test: clicking board plan card loads preview (not only toast)
- [ ] Run `cd frontend && npm test -- --run src/pages/QuestPage.test.tsx` (expect fail before impl)

### Task 3: QuestPage compose + board Plan actions

**Files:**
- Modify: `frontend/src/pages/QuestPage.tsx`

- [ ] State: goalText, goalRepo, activePlan (GoalPlan | null), message
- [ ] Mutations: fromGoal / approve / reject; invalidate `quest-board` on success
- [ ] Compose UI pane with testids `quest-wb-goal-input`, `quest-wb-repo-input`, `quest-wb-route`, `quest-wb-approve`, `quest-wb-reject`
- [ ] `selectItem` for `kind===plan`: fetch/show plan; enable approve when `status==="draft"`
- [ ] On approve success: setSelectedRunId from plan.runId if available; clear draft UI as appropriate
- [ ] Run vitest until green

### Task 4: Sprint docs + watermarks

**Files:**
- Create/update: `doc/plan/sprint-dx49-quest-compose.md`, `doc/plan/v3.1-release-scope.md`, `doc/plan/TODO.md`, `CHANGELOG.md`, `doc/plan/v3.x-program.md` (v3.1 board pointer)

- [ ] Mark DX49 tasks ✅ when done
- [ ] TODO: DX49–DX54 rows; 详排 point at v3.1 scope

### Task 5: Verify

- [ ] `cd frontend && npm test -- --run src/pages/QuestPage.test.tsx`
- [ ] Manual smoke note: open `/ui/quest`, generate plan, approve (optional if no worker)

---

## Out of scope

SSE `plan.*`, Diff reject, webhook Plan path, removing Runs quest-pane.
