# Sprint DX61 — OIDC RS256 / JWKS（v4.0 · A）

> **方案：** discovery `jwks_uri` + RS256 校验；保留 HS256；**无新表**  
> **状态：** ✅ 完成  
> **设计：** [`../../docs/superpowers/specs/2026-09-09-v40-dx61-oidc-rs256-design.md`](../../docs/superpowers/specs/2026-09-09-v40-dx61-oidc-rs256-design.md)

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX61-1 | JWKS fetch/cache + RS256 verify | ✅ |
| DX61-2 | 测试（RS256 + HS256 回归） | ✅ |
| DX61-3 | sprint / TODO / CHANGELOG | ✅ |

## 验收

```bash
go test ./internal/idp/ ./internal/api/ -run 'OIDC|ClientExchange|RS256' -count=1
```
