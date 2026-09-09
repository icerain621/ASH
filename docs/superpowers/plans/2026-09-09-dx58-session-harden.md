# DX58 Session Harden Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (inline in this session). Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete Session gateway POC with same-`sid` access JWT refresh and JWT `scope` enforcement — no new tables.

**Architecture:** Extend DX57 `audit_log` (`auth.session`) with rotation fields; add `POST /auth/sessions/refresh`; put claims.`scope` in Gin context and gate `requirePermission` / `requireOrgPermission`; validate device-mint scope ⊆ caller grants.

**Tech Stack:** Go, Gin, existing HS256 JWT helpers in `internal/api/auth.go`, GORM `audit_log`.

## Global Constraints

- No new SQL tables / migrations (E5)
- Refresh only while JWT unexpired (expired → `AUTH_SESSION_EXPIRED` via refresh-path allow-parse)
- Empty scope = no narrowing; non-empty = intersection
- Refresh must not enlarge scope; reuse same `sid`
- Console UI deferred to DX59
- Chinese CHANGELOG / TODO; commit only if user asks

---

### Task 1: Refresh + rotation fields

**Files:**
- Modify: `internal/api/auth_sessions.go`
- Modify: `internal/api/handlers.go` (route)
- Modify: `internal/api/auth.go` (middleware expired-on-refresh; context keys)
- Modify: `internal/apicodes/catalog.go`
- Test: `internal/api/auth_sessions_test.go`

**Interfaces:**
- Produces: `refreshAuthSession`, `rotateAuthSession(c, sid, ttl)`, payload fields `rotatedAt`/`rotateCount`

- [ ] **Step 1:** Add failing tests `TestAuthSessionRefreshAndRevoke`, `TestAuthSessionScopeEnforcement`
- [ ] **Step 2:** Implement refresh + rotate-in-place + apicodes + route
- [ ] **Step 3:** `go test ./internal/api/ -run AuthSession -count=1` PASS

### Task 2: Scope enforcement + device mint subset

**Files:**
- Modify: `internal/api/auth.go` (`setAuthSessionContext`, `requirePermission`, `requireOrgPermission`, `tokenScopeAllows`)
- Modify: `internal/api/auth_sessions.go` (`createDeviceAuthSession` scope ⊆ check)

- [ ] **Step 1:** Implement scope context + deny codes + mint validation
- [ ] **Step 2:** Re-run AuthSession tests PASS

### Task 3: OpenAPI + docs

**Files:**
- Modify: `doc/api/openapi-ash-v1.yaml`
- Modify: `doc/plan/sprint-dx58-session-harden.md`, `TODO.md`, `CHANGELOG.md`
- Modify: design status → done
- Run: `make swagger` + `make openapi-check`

- [ ] **Step 1:** OpenAPI paths/schemas + regen
- [ ] **Step 2:** Mark sprint/TODO/CHANGELOG ✅
