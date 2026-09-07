# v2.9 optional remote sandbox design (DX37–DX42)

## Goal

Optional E2B-class remote isolation behind env opt-in. Local Landlock/process remains default. No new tables; Doctor stays ALL 57 / M4 10.

## Confirmed (D1–D7)

See `doc/plan/v2.9-release-scope.md` §7.

## DX37 — Contract

- Package `internal/sandbox/remote`: `Backend` extends `sandbox.Executor` with `Name` / `Available` / `Capabilities`
- Env config; default **disabled**
- `mock` backend for tests and dry-run
- `e2b` HTTP client (DX38)
- Runs dispatch handles `executor=remote`

## DX39 — Prefer remote

- `DefaultRouter`: isolated + `ASH_SANDBOX_REMOTE` enabled + Available → `executor=remote`
- `ASH_SANDBOX_REMOTE_ON_FAIL=deny` refuses; default falls back to landlock/docker/process
- `RouteRequest.PreferRemote` per-call override (`remote` / `local`)
- Local remains default when remote env is off

## Later

- **DX40** smoke + evidence
- **DX41** console status
- **DX42** freeze + signoff
