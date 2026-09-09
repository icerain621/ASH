# v3.2 DX57 — 多端 Session 网关 POC（scoped device token）

> Status: **done** (2026-09-09)  
> Program: T3 Session gateway · **no new tables** · reuse `audit_log` (`auth.session`)

## Goals

- Extend ASH JWT claims: `sid` / `did` / `typ` (`primary`|`device`) / optional `scope[]`
- Persist session registry in `audit_log` (`id=asess_*`, `event_type=auth.session`)
- Login / OIDC mint **primary** session
- `POST /auth/sessions/device` mint shorter-lived **device** token from primary
- `GET /auth/sessions` · `DELETE /auth/sessions/{sid}` (revoke; middleware rejects revoked `sid`)

## Non-goals

- Refresh tokens / rotation（DX58）  
- Durable `auth_sessions` table（DX58 if needed）  
- Scope enforcement in `requirePermission`（DX58）  
- Console UI（DX59）

## API

| Method | Path | Notes |
|--------|------|-------|
| POST | `/api/v1/auth/sessions/device` | Bearer primary → device JWT |
| GET | `/api/v1/auth/sessions` | list caller sessions |
| DELETE | `/api/v1/auth/sessions/{sid}` | revoke |

## TTL

- primary: 24h（unchanged）  
- device: default 4h（`ttlSeconds` clamp 300–86400）
