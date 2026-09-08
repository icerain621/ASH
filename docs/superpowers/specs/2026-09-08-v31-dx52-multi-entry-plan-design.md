# v3.1 DX52 — 多入口统一 Plan（webhook → GoalPlan）

> Status: **implemented** (2026-09-08)  
> Program: T2 Quest · **no new tables** · Approach **A** (keyword-prefixed `FromGoal`)

## Goals

- CI failure via GitHub webhook **always** creates a `goal_plans` draft (Quest board visible)
- `autoRun=1` maps to `FromGoal` + `AutoApprove: true` (same gate bypass as `--yes`)
- Response includes `planId` (+ `ashRunId` / `ashTraceId` when auto-approved)
- Session already uses `FromGoal` — document only; no behavior change required

## Behavior matrix

| Event | Today | DX52 |
|-------|-------|------|
| Failure, `autoRun=0` | Diagnosis only | Diagnosis + **draft Plan** |
| Failure, `autoRun=1` | Direct `runs.Create` hotfix | Plan + **autoApprove** → Run |
| Success / duplicate | Unchanged | Unchanged |

## Routing (Approach A)

Goal text = `"hotfix prod: "` + `ci.IssueOrSpecFromDiagnosis(...)` so Space Rules keyword routing picks **hotfix**; `latestDoc` → `hotfix@1.1.0` (matches prior hardcode).

```
CreatedBy: "webhook:github"
SpaceID: connection.SpaceID
RepoRoot: query repoRoot (default ".")
AutoApprove: autoRun
PolicyProfile: "hotfix" (optional override if FromGoal supports it)
```

## API delta

`POST /api/v1/webhooks/github` response adds:

```json
{ "planId": "gplan_…", "ashRunId": "…", "ashTraceId": "…" }
```

Swagger / OpenAPI updated. **No new tables / Doctor bumps.**

## Non-goals

- `goal.FromRouted` helper (Approach B — deferred)
- Closing `POST /runs` / `ash run` direct Create escape hatches
- Stricter session `autoApprove` RBAC
- Live board / `plan.*` SSE (DX53)

## Tests

- `TestGitHubWebhookHMACAndDiagnose`: `autoRun` off → `planId` set, `ashRunId` empty  
- `TestGitHubWebhookAutoRun`: `planId` + `ashRunId` set; plan row exists
