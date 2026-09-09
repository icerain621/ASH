# v3.2 DX55 — OIDC 薄切片（登录 → ASH JWT）

> Status: **implemented** (2026-09-08)  
> Program: T3 IdP · **no new tables** · env-gated

## Goals

- Optional OIDC Authorization Code flow when `ASH_OIDC_ENABLED=1`
- `GET /api/v1/auth/oidc/login` → IdP authorize redirect（state）
- `GET /api/v1/auth/oidc/callback` → code exchange → map user by **email** → issue existing ASH JWT
- JIT create `users` row if email unknown（empty password hash；status=active）
- Default space via existing `defaultLoginSpace` / `local`

## Env

| Var | Required when enabled |
|-----|------------------------|
| `ASH_OIDC_ENABLED` | `1` / `true` |
| `ASH_OIDC_ISSUER` | OIDC issuer URL |
| `ASH_OIDC_CLIENT_ID` | client id |
| `ASH_OIDC_CLIENT_SECRET` | client secret |
| `ASH_OIDC_REDIRECT_URL` | e.g. `http://localhost:8080/api/v1/auth/oidc/callback` |
| `ASH_OIDC_SCOPES` | optional；default `openid email profile` |

## Non-goals

- Multi-end session gateway（DX57+）  
- Federated `sub` durable link table（DX56 may add claim storage without DDL if possible）  
- SAML / multiple IdPs  
- Frontend SSO button（DX59）  
- Changing default `ASH_AUTH_MODE`

## Tests

- httptest mock IdP (discovery + token + HS256 id_token)  
- Callback issues ASH JWT; JIT user created  
- Disabled OIDC → 404 / not configured
