# Sprint VX61 — Webhook 投递走 Notifier / Webhook delivery via Notifier

> **前置 / Prerequisite：** v6.5 已冻结；[`v6.6-release-scope.md`](v6.6-release-scope.md)  
> **状态 / Status：** ✅  
> **做法 / Approach：** 非重复入库后 `notify.FromEnv`；默认 null 不失败。无新表。  

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| VX61-1 | Webhook 成功后 Notify `ci.webhook` | ✅ |
| VX61-2 | 重复投递不再通知 | ✅ |
| VX61-3 | 响应 `notifier` + OpenAPI | ✅ |

## 验收 / Verify

```bash
go test ./internal/api/ -count=1 -run 'TestGitHubWebhookIngressSeam'
```
