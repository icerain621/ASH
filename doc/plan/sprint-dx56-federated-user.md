# Sprint DX56 — 联邦用户绑定（v3.2 · T3）

> **方案：** `users.oidc_issuer/subject`；查找 sub→email→JIT；`ASH_OIDC_DEFAULT_SPACE_ID`；**无新表**（SQL **32**）  
> **状态：** ✅ 完成  
> **设计：** [`../../docs/superpowers/specs/2026-09-08-v32-dx56-federated-user-design.md`](../../docs/superpowers/specs/2026-09-08-v32-dx56-federated-user-design.md)

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX56-1 | SQL 32 + User 模型列 | ✅ |
| DX56-2 | ensureOIDCUser 绑定逻辑 + default space | ✅ |
| DX56-3 | 测试 + TODO / CHANGELOG / scope 锚点 | ✅ |

## 验收

```bash
go test ./internal/api/ -run 'OIDC' -count=1
go test ./internal/store/sqlmigrations/ -count=1
```
