# v3.2 DX56 — 联邦用户绑定（oidc sub + 默认 Space）

> Status: **implemented** (2026-09-08)  
> Program: T3 IdP · **no new tables** · SQL **32**（`users` 列扩展）· RLS **51** 不变

## Goals

- Persist IdP link on `users`: `oidc_issuer` + `oidc_subject`
- Resolve order: **(issuer, subject)** → **email** → JIT create
- Email match backfills issuer/subject; conflict if subject already bound to another user
- Optional `ASH_OIDC_DEFAULT_SPACE_ID` as preferred login space when callback omits `spaceId`

## Non-goals

- Multi-IdP concurrent configs  
- RS256 id_token（仍 DX55 限制）  
- Session gateway（DX57+）  
- Auto org/role provisioning beyond preferred space selection  

## Schema

```sql
ALTER TABLE users ADD COLUMN oidc_issuer VARCHAR(512);
ALTER TABLE users ADD COLUMN oidc_subject VARCHAR(256);
CREATE UNIQUE INDEX uidx_users_oidc_issuer_subject
  ON users (oidc_issuer, oidc_subject)
  WHERE oidc_subject IS NOT NULL AND oidc_subject <> '';
```
