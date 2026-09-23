# Sprint VX51 — Webhook 走 Ingress 缝 / Live webhook through ingress

> **前置 / Prerequisite：** v6.4 已冻结；[`v6.5-release-scope.md`](v6.5-release-scope.md)  
> **状态 / Status：** ✅  
> **做法 / Approach：** `ASH_INGRESS=webhook-github` 时 HMAC 之后 `Accept`；默认 null 不拦截。无新表。  

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| VX51-1 | Webhook 调用 `ingress.FromEnv` | ✅ |
| VX51-2 | 空投递 fail-closed（`INGRESS_REJECTED`） | ✅ |
| VX51-3 | 响应标记 + OpenAPI | ✅ |

## 验收 / Verify

```bash
go test ./internal/api/ -count=1 -run 'TestGitHubWebhook'
```
