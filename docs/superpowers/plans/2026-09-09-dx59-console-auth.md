# DX59 Console Auth Implementation Plan

> **For agentic workers:** Use executing-plans inline. Checkbox tracking.

**Goal:** Console login + OIDC UI redirect + Space Sessions panel + smoke evidence.

**Architecture:** Backend stores `ui` flag on OIDC state; callback 302 to `/ui/login#token&spaceId`. Frontend LoginPage + Space Sessions using DX57–58 APIs. No new tables.

**Tech Stack:** Go/Gin, React/Vite/TanStack Router/Query, vitest, bash smoke.

## Global Constraints

- No new SQL tables
- `?ui=1` optional; default callback remains JSON
- Keep Space `dev-login` and password API login
- Chinese CHANGELOG/TODO; commit only if asked

---

### Task 1: OIDC ui flag + redirect

**Files:** `internal/idp/oidc.go`, `internal/api/oidc.go`, `internal/api/oidc_test.go`, OpenAPI

### Task 2: Frontend login + auth client + Sessions

**Files:** `platform.api.ts`, `LoginPage.tsx`, `router.tsx`, `AppLayout.tsx`, `SpacePage.tsx`, vitest

### Task 3: Smoke + docs

**Files:** `scripts/oidc-console-smoke.sh`, Makefile, evidence, sprint/TODO/CHANGELOG
