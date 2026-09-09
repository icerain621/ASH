# Sprint DX58 — Session 网关硬化（v3.2 · T3）

> **方案：** same-`sid` refresh + JWT scope 强制；**无新表**；控制台 → DX59  
> **状态：** ✅ 完成  
> **设计：** [`../../docs/superpowers/specs/2026-09-08-v32-dx58-session-harden-design.md`](../../docs/superpowers/specs/2026-09-08-v32-dx58-session-harden-design.md)  
> **计划：** [`../../docs/superpowers/plans/2026-09-09-dx58-session-harden.md`](../../docs/superpowers/plans/2026-09-09-dx58-session-harden.md)

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX58-1 | refresh API + audit_log 轮换字段 | ✅ |
| DX58-2 | scope 强制 + device mint 子集校验 | ✅ |
| DX58-3 | 测试 + OpenAPI + TODO / CHANGELOG | ✅ |

## 验收

```bash
go test ./internal/api/ -run 'AuthSession|OIDC|Login' -count=1
make openapi-check
```
