# v4.0 DX63 — jti 吊销硬化（当前对 + 短宽限）

> Status: **done** (2026-09-09)  
> Program: A Auth · **no new tables** · extend `audit_log` `auth.session`  
> Depends: DX62 (`jti`/`iat` already issued on access+refresh)

## Intent

Close the post-rotate replay window: after refresh, previous access/refresh JWTs are rejected outside a short grace window. Keep existing `sid` revoke as the hard kill switch for the whole session.

## Decisions (locked)

| # | Choice |
|---|--------|
| Threat | C with A primary: bind current jtis **and** keep sid revoke |
| Grace | Configurable short grace; default **60s** |
| Legacy | Tokens with `sid` but **no `jti`**: sid-revoke check only (compat) |
| Storage | Approach **1**: current + previous jti pair on registry payload |

## Non-goals

- Full historical jti denylist / new tables  
- Distributed lock across workers  
- Exposing jti fields on public `AuthGatewaySession` list API  
- Auto-refresh interceptor on the console (optional later)

## Data model (`auth.session` payload)

Extend existing JSON (no SQL migration):

| Field | Meaning |
|-------|---------|
| `accessJti` | Current access JWT `jti` |
| `refreshJti` | Current refresh JWT `jti` |
| `prevAccessJti` | Previous access `jti` (set on rotate) |
| `prevRefreshJti` | Previous refresh `jti` (set on rotate) |
| `rotatedAt` | Already present; grace anchor |
| `status` | `active` \| `revoked` (unchanged) |

**Issue:** set current pair; prev empty.  
**Rotate:** `prev* ← current*`, then write new current pair + `rotatedAt=now`.  
**Revoke:** `status=revoked` only (jti fields may remain for audit).

## Config

| Env | Default | Clamp | Notes |
|-----|---------|-------|-------|
| `ASH_AUTH_TOKEN_GRACE_SEC` | `60` | `0`–`300` | `0` = accept only current pair |

Helper: `authTokenGrace()` next to `authRefreshTTL()`.

## Validation flow

For any authenticated request whose JWT has `sid`:

1. Load registry by `sid`. Missing row → non-revoked compat (existing DX57 behavior).  
2. `status == revoked` → `401 AUTH_SESSION_REVOKED`.  
3. If `claims.jti` empty → pass (legacy A); skip jti bind.  
4. Else match by token typ:
   - `typ=refresh` → must equal `refreshJti`, or `prevRefreshJti` within grace  
   - else (access `primary`/`device`/empty) → `accessJti` or `prevAccessJti` within grace  
5. Mismatch → `401 AUTH_TOKEN_REPLAY`.

Grace: if grace <= 0, prev pair is never accepted; else `now.Unix() <= rotatedAt + graceSec` (requires `rotatedAt > 0`).

Refresh handler: after successful verify, run the same jti bind before rotate (so replayed refresh outside grace fails with `AUTH_TOKEN_REPLAY`, not a silent re-issue).

DX62 rules unchanged: refresh typ only on `/auth/sessions/refresh`; expired access without valid refresh → `AUTH_REFRESH_REQUIRED`.

## Error codes

| Code | When |
|------|------|
| `AUTH_TOKEN_REPLAY` | jti present but not current (and not prev-within-grace) |
| `AUTH_SESSION_REVOKED` | unchanged |
| `AUTH_REFRESH_REQUIRED` / `AUTH_REFRESH_TOKEN_MISUSE` | unchanged (DX62) |

## OpenAPI / docs

- Describe refresh + session gateway: jti binding + grace env.  
- Catalog + CHANGELOG + `sprint-dx63-*.md` + TODO/v4.0 scope tick.  
- Do not require clients to send jti explicitly (embedded in JWT).

## Tests

1. Login → refresh → old refresh **outside** grace → `AUTH_TOKEN_REPLAY`  
2. With grace=60 (or test env small), old access/refresh **inside** grace → OK  
3. Revoke `sid` → current and prev → `AUTH_SESSION_REVOKED`  
4. Legacy token (sid, no jti) while active → OK; after revoke → revoked  
5. `ASH_AUTH_TOKEN_GRACE_SEC=0` → prev rejected immediately after rotate  

## Verification

```bash
go test ./internal/api/ -count=1 -run 'TestAuthSession'
make openapi-check
```
