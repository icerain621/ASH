# v4.0 DX62 — 独立 Refresh Token（Option B）

> Status: **done** (2026-09-09)  
> Program: A Auth · **no new tables** · extend `audit_log` auth.session

## Goals

- Issue `refreshToken` (`typ=refresh`) alongside access (`primary`|`device`) on login/OIDC/device mint
- `POST /auth/sessions/refresh` prefers refresh Bearer/body; rotates access+refresh same `sid`
- Access expired without refresh → `AUTH_REFRESH_REQUIRED`（legacy primary/device unexpired still allowed）
- Refresh on non-refresh routes → `AUTH_REFRESH_TOKEN_MISUSE`

## TTL

- access: unchanged (primary 24h / device 4h)
- refresh: default 30d (`ASH_AUTH_REFRESH_TTL_SEC`, clamp 86400–7776000)

## Non-goals

- jti replay detection（DX63）  
- New tables  
