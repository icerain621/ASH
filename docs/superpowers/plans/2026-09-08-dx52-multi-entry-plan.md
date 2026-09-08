# DX52 Multi-entry Plan Implementation Plan

> **For agentic workers:** implement webhook → FromGoal; no new tables.

**Goal:** CI failures always create GoalPlan; autoRun = AutoApprove.

**Architecture:** Replace direct `runs.Create` in `githubWebhook` with `goal.FromGoal` using Approach A keyword prefix.

**Tech Stack:** Go, Gin, existing `goal.Service`, httptest webhook tests.

---

### Task 1: Webhook → FromGoal + tests

**Files:**
- Modify: `internal/api/webhook.go`
- Modify: `internal/api/webhook_test.go`
- Modify: swagger/openapi + sprint/TODO/CHANGELOG

**Steps:**
1. Add `PlanID` to `githubWebhookResponse`
2. On `Diagnosis != nil && !Duplicate`, call `h.goalFor(c).FromGoal(...)` with goal prefix `hotfix prod: ` + diagnosis issue text; `AutoApprove: autoRun`
3. Map plan run fields to response; audit `planId`
4. Update tests; run `go test ./internal/api/ -run TestGitHubWebhook -count=1`
5. `make swagger` + openapi sync if needed; mark sprint ✅
