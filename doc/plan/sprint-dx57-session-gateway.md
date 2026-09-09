# Sprint DX57 — 多端 Session 网关 POC（v3.2 · T3）

> **方案：** JWT `sid/did/typ` + `audit_log` 登记；device token 交换与撤销；**无新表**  
> **状态：** ✅ 完成  
> **设计：** [`../../docs/superpowers/specs/2026-09-08-v32-dx57-session-gateway-design.md`](../../docs/superpowers/specs/2026-09-08-v32-dx57-session-gateway-design.md)

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX57-1 | claims + auth.session 登记 / 撤销校验 | ✅ |
| DX57-2 | device / list / revoke API | ✅ |
| DX57-3 | 测试 + OpenAPI + TODO / CHANGELOG | ✅ |

## 验收

```bash
go test ./internal/api/ -run 'AuthSession|OIDC|Login' -count=1
make openapi-check
```
