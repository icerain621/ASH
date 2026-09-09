# DX64 Console Auth Gate Implementation Plan

> **For agentic workers:** Inline execution (user: 写实现计划并开工).

**Goal:** Optional SPA login gate via `ASH_CONSOLE_AUTH_REQUIRED` → `/readyz.consoleAuthRequired`; hide Dev Token when on.

**Architecture:** Config helper + HealthResponse field; frontend caches readyz flag; root `beforeLoad` redirects; `api()` clears session on auth 401 and redirects when gate on.

**Tech Stack:** Go/Gin, TanStack Router, Vite React.

## Global Constraints

- Default gate **off**
- No new tables; do not change `ASH_AUTH_MODE` default
- readyz-only field (not required on scale readiness)

---

### Task 1: Backend readyz flag + test

**Files:** `internal/config/config.go`, `internal/api/swagger_models.go`, `internal/api/handlers.go`, `internal/api/ops_env_test.go` (or new), OpenAPI

### Task 2: Frontend gate helper + router + client

**Files:** `frontend/src/modules/platform/auth/consoleGate.ts`, router, client, LoginPage, SpacePage, tests, docs
