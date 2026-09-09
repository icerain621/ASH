# Sprint DX62 — 独立 Refresh Token（v4.0 · A · Option B）

| # | 项 | 状态 |
|---|-----|------|
| DX62-1 | 登录/OIDC/device 签发 access+`refreshToken`（`typ=refresh`） | ✅ |
| DX62-2 | `POST /auth/sessions/refresh` 优先 refresh；双 token 同 `sid` 轮换 | ✅ |
| DX62-3 | 过期 access → `AUTH_REFRESH_REQUIRED`；refresh 误用 → `AUTH_REFRESH_TOKEN_MISUSE` | ✅ |
| DX62-4 | 测试 + OpenAPI + CHANGELOG | ✅ |

## 设计

- 见 `docs/superpowers/specs/2026-09-09-v40-dx62-refresh-token-design.md`
- Refresh TTL：默认 30d；`ASH_AUTH_REFRESH_TTL_SEC`（86400–7776000）
- Legacy：未过期 `primary`/`device` 仍可 refresh（弃用路径）
- **无新表**；`audit_log` `auth.session` 增 `refreshExp`

## 非目标

- jti / refresh 重放 → **DX63**
