# v3.2 DX58 — Session 网关硬化（refresh + scope 强制）

> Status: **done** (2026-09-09; Option 1)  
> Program: T3 Session gateway · **no new tables** · extend DX57 `audit_log` (`auth.session`)  
> Approach: **Option 1** — same-`sid` access JWT rotation + `requirePermission` scope intersection  
> Console UI → **DX59**

## Goals

- `POST /api/v1/auth/sessions/refresh`：在 **未过期** 的 active session 上轮换 access JWT（沿用 `sid`/`did`/`typ`/`scope`/`spaceId`/`role`）
- 更新 `audit_log` payload 的 `exp` + `rotatedAt` / `rotateCount`（仍无新表）
- JWT `scope` **非空**时，在 `hasPermission` / `hasOrgPermission` 中强制交集；空 scope 不收窄（兼容 primary）
- device mint：`scope` 必须 ⊆ 调用方已有权限；refresh **不得扩大** scope

## Non-goals

- 独立 `typ=refresh` token / 过期后 refresh  
- `auth_sessions` 表 / jti 单票黑名单  
- 控制台 Session UI（DX59）  
- 改变 `ASH_AUTH_MODE` 或默认密码登录

## API

| Method | Path | Notes |
|--------|------|-------|
| POST | `/api/v1/auth/sessions/refresh` | Bearer primary\|device；可选 `{ "ttlSeconds": N }`（clamp 同 DX57） |
| （既有） | `/auth/sessions/device` | 新增：非空 scope 须 ⊆ 调用方权限 → 否则 `AUTH_SCOPE_INVALID` |

响应：`AuthSessionResponse`（含 `session`）。

## Refresh 语义

1. 解析 Bearer；要求 `sid` 非空且 registry `status=active`  
2. JWT 必须 **未过期**（POC：不做 stale refresh）  
3. 签发新 JWT：同一 `sid`/`did`/`typ`/`scope`；TTL 按 typ 默认或请求 clamp  
4. 写回 `audit_log.payload_json`：`exp`、`rotatedAt`、`rotateCount++`  
5. 旧 JWT 在原 `exp` 前仍可用；**revoke `sid` 立即废止**整条会话（含已轮换）

## Scope 强制

- middleware：claims.`scope` → context（`authTokenScope`）  
- `hasPermission` / `hasOrgPermission`：角色或成员权限通过后，若 scope 非空，则 `permission` 须匹配 scope 项（精确或 `*`）  
- 失败：`403` + `AUTH_SCOPE_DENIED`  
- 空 / 缺省 scope：行为与 DX57 前一致

## Errors（apicodes）

| Code | When |
|------|------|
| `AUTH_SESSION_REVOKED` | （既有）revoked sid |
| `AUTH_SESSION_EXPIRED` | refresh 时 JWT 已过期 |
| `AUTH_SCOPE_DENIED` | token scope 不含所需 permission |
| `AUTH_SCOPE_INVALID` | device mint scope 超出调用方权限 |

## Tests

- refresh 更新 `exp`/`rotateCount`；revoke 后新旧 JWT 均拒  
- scoped device：允许 scope 内权限、拒绝越权  
- 空 scope primary：不回归现有权限测例  
- device mint 非法 scope → `AUTH_SCOPE_INVALID`

## Verification

```bash
go test ./internal/api/ -run 'AuthSession|OIDC|Login' -count=1
make openapi-check
```
