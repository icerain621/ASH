# v4.0 DX64 — 控制台可选强制登录门闸

> Status: **done** (2026-09-09)  
> Program: A Auth · SPA gate only · **no new tables**  
> Depends: DX59 `/ui/login` · DX62–63 session tokens

## Intent

Optional console login gate: when enabled, SPA routes (except `/login`) require a stored access token; Dev Token UI is hidden; auth API failures clear the local session and redirect to login. Default remains **off** so local `ASH_AUTH_MODE=dev` workflows stay unchanged.

## Decisions (locked)

| # | Choice |
|---|--------|
| Scope | **B+C**: SPA gate + hide Dev Token when on; API auth semantics unchanged |
| Flag surface | Extend public **`GET /readyz`** with `consoleAuthRequired` |
| Logged-in check | **C**: token present for route gate; on auth `401` clear session; redirect to `/login` only when gate is on |
| Approach | Router `beforeLoad` / root guard + `/readyz` (not Layout-only, not cookie gate on `/ui`) |

## Non-goals

- Changing default `ASH_AUTH_MODE`  
- Forcing OIDC  
- Server-side cookie gate on static `/ui` assets  
- Auto-refresh interceptor (may use stored refresh later; out of DX64)

## Config

| Env | Default | Meaning |
|-----|---------|---------|
| `ASH_CONSOLE_AUTH_REQUIRED` | unset/off | `1` / `true` / `yes` (case-insensitive) → on |

Wire via `internal/config` or `readyzResponse` env read (prefer config field for testability).

## Backend

- Add `ConsoleAuthRequired bool` to `HealthResponse` / `readyz` JSON as `consoleAuthRequired`.
- OpenAPI + swagger models; scale parity **does not** require this field on `/scale/readiness` (console-only; document as readyz-only unless parity already demands identical shapes — prefer readyz-only to avoid scale noise).
- Unit test: env on/off flips field.

## Frontend

1. Extend `ReadyzResponse` with `consoleAuthRequired?: boolean`.
2. Small helper `isConsoleAuthRequired(readyz)` + cached fetch (module-level promise or short TTL).
3. Route guard (root or non-login routes `beforeLoad`):
   - if required && no `getAuthToken()` && path ≠ `/login` → redirect `/login` (optional `?redirect=`).
4. `api()` in `client.ts`:
   - on `401` with auth-ish codes → `clearAuthSession()` (remove token + refresh + space keys as appropriate);
   - if gate on → soft navigate to `/login` (avoid hard loop on login page itself).
5. SpacePage: hide Dev Token / `dev-login` controls when gate on.
6. LoginPage: when gate on, omit “也可在空间用 Dev Token” copy.

## Error / UX notes

- Gate off: no redirect on 401 (current behavior); still OK to clear bad token optionally — **spec: clear token on auth 401 always; redirect only when gate on**.
- Login route always reachable when gate on.

## Tests

- Go: `TestReadyzConsoleAuthRequired` (off by default; on with env).
- Frontend: unit/router test or component test — no token + required → redirect; Dev Token hidden when required.
- Manual: `ASH_CONSOLE_AUTH_REQUIRED=1` + jwt mode → open `/ui/runs` → lands on login.

## Verification

```bash
go test ./internal/api/ -count=1 -run 'TestReadyz'
# frontend: vitest for gate helper / login hash unchanged
make openapi-check
```
