# Sprint DX59 — 控制台登录 + Session 面板（v3.2 · T3）

> **方案：** `/ui/login` + OIDC `?ui=1` hash 回跳 + Space Sessions；smoke 证据；**无新表**  
> **状态：** ✅ 完成  
> **设计：** [`../../docs/superpowers/specs/2026-09-09-v32-dx59-console-auth-design.md`](../../docs/superpowers/specs/2026-09-09-v32-dx59-console-auth-design.md)  
> **计划：** [`../../docs/superpowers/plans/2026-09-09-dx59-console-auth.md`](../../docs/superpowers/plans/2026-09-09-dx59-console-auth.md)

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX59-1 | OIDC `ui=1` state + callback 302 hash | ✅ |
| DX59-2 | `/ui/login` + platform auth client | ✅ |
| DX59-3 | Space Sessions 面板 | ✅ |
| DX59-4 | smoke + vitest + TODO / CHANGELOG | ✅ |

## 验收

```bash
go test ./internal/api/ ./internal/idp/ -run 'OIDC|AuthSession|ClientExchange' -count=1
cd frontend && npm test -- --run loginHash SpacePage
make oidc-console-smoke
make openapi-check
```
