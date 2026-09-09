# DX63 jti Revoke Harden Implementation Plan

> **For agentic workers:** Inline execution in this session (user: 继续推进).

**Goal:** Bind access/refresh JWTs to current (and grace-window previous) jtis on `auth.session` registry; keep sid revoke.

**Architecture:** Extend `authSessionPayload` with current/prev jti pair; store on issue/rotate; middleware + refresh gate reject `AUTH_TOKEN_REPLAY` outside grace (`ASH_AUTH_TOKEN_GRACE_SEC`, default 60).

**Tech Stack:** Go, Gin, existing JWT helpers, `audit_log`.

## Global Constraints

- No new tables (F4)
- Legacy: no `jti` in JWT → sid check only
- Pre-DX63 registry rows with empty jti fields → skip jti bind until next issue/rotate fills them
- Grace default 60s, clamp 0–300

---

### Task 1: Failing tests for jti bind + grace

**Files:**
- Modify: `internal/api/auth_sessions_test.go`

- [ ] Add `TestAuthSessionJtiReplayAndGrace` covering grace=0 replay reject, grace>0 accept, revoke, legacy no-jti
- [ ] Run fail before impl

### Task 2: Payload + issue/rotate + gate

**Files:**
- Modify: `internal/api/auth_sessions.go`, `internal/api/auth.go`, `internal/apicodes/catalog.go`

- [ ] Store jtis on issue/rotate; `authTokenGrace()`; `authSessionAllowsJti`; wire middleware + refresh
- [ ] Tests pass

### Task 3: Docs + OpenAPI

**Files:**
- `doc/api/openapi-ash-v1.yaml`, CHANGELOG, TODO, v4.0 scope, sprint-dx63, spec status
- `make swagger` + `make openapi-check`
