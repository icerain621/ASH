# v4.0 DX65 — Device mint UI 薄切片 + smoke

> Status: **done** (2026-09-09)  
> Program: A Auth · **no new tables** · reuse DX57–58 device mint API  
> Depends: Space Auth Sessions panel (DX59) · device mint API (DX57/58)

## Intent

Thin console UX to mint a device-scoped session from a primary JWT, show the issued tokens **once** without switching the console session, plus a Makefile smoke for evidence.

## Decisions (locked)

| # | Choice |
|---|--------|
| Scope | **A+C**: mint form on existing Auth Sessions panel + `make device-session-smoke` |
| After mint | **Show once**; keep primary console session (`setAuthSession` not called) |
| Approach | Inline form (not modal, not smoke-only) |

## Non-goals

- New Devices page  
- Auto-switch console to device token  
- Backend contract changes  
- Full device lifecycle product UI

## Frontend

**API** (`platform.api.ts`):

```ts
createDeviceAuthSession(body?: {
  deviceId?: string;
  deviceLabel?: string;
  scope?: string[];
  ttlSeconds?: number;
}): Promise<AuthSessionResponse>
// POST /auth/sessions/device
```

**SpacePage Auth Sessions panel:**

- Form fields: `deviceId` (optional), `scope` (comma-separated → `string[]`), `ttlSeconds` (optional number)
- Submit disabled when no access token or mint in flight
- On success: render one-shot panel with `token`, `refreshToken`, `session.sid` / `typ` / `did` (`data-testid="device-mint-result"`)
- On error: surface API error message/code
- Invalidate `listAuthSessions` after success
- Do **not** call `setAuthSession` with device tokens

**Tests:** SpacePage (or unit) — mint control present; mock success shows result; mock `AUTH_SCOPE_INVALID` shows error.

## Smoke

- Script: `scripts/device-session-smoke.sh`
- Target: `make device-session-smoke`
- Flow (self-contained, jwt mode, temp DB):
  1. Seed/login primary (or password login against temp worker)
  2. `POST /auth/sessions/device`
  3. `GET /auth/sessions` contains device `typ`
  4. `DELETE /auth/sessions/{sid}`
  5. Device access token → `/auth/me` → 401
- Style: follow `scripts/oidc-console-smoke.sh` patterns (bash, LF, exit non-zero on fail)

## Docs

- CHANGELOG, TODO, `sprint-dx65-device-mint-ui.md`, v4.0 scope tick
- OpenAPI unchanged unless annotations missing (device route already exists)

## Verification

```bash
# frontend
cd frontend && npx vitest run src/pages/SpacePage.test.tsx
# smoke
make device-session-smoke
```
