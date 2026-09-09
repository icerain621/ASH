# Sprint DX55 — OIDC 薄切片（v3.2 · T3）

> **方案：** env-gated OIDC code flow → 现有 ASH JWT；email 匹配 / JIT user；**无新表**  
> **状态：** ✅ 完成  
> **设计：** [`../../docs/superpowers/specs/2026-09-08-v32-dx55-oidc-thin-design.md`](../../docs/superpowers/specs/2026-09-08-v32-dx55-oidc-thin-design.md)  
> **范围：** [`v3.2-release-scope.md`](v3.2-release-scope.md)

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX55-1 | `internal/idp` + config env | ✅ |
| DX55-2 | `/auth/oidc/login` + `/callback` | ✅ |
| DX55-3 | 测试 + OpenAPI + TODO / CHANGELOG | ✅ |

## 验收

```bash
go test ./internal/idp/ ./internal/api/ -run 'OIDC|Oidc|ClientExchange' -count=1
# ok
```

## 交付摘要

- `ASH_OIDC_ENABLED` + issuer/client/secret/redirect
- `GET /api/v1/auth/oidc/login` → IdP；`/callback` → ASH JWT
- DX55 支持 HS256；**DX61** 增加 RS256/JWKS；SAML → 后续
