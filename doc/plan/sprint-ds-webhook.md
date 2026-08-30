# Sprint DS：Webhook 触发（CI→Run）（方案 C）Implementation Plan

> **方案：** 已批准 **C** = HMAC 入站 + 失败诊断 + 可选 autoRun Create + 幂等 + OpenAPI/清单  
> **Goal：** GitHub Actions 失败可经 webhook 进入 ASH：验签 → upsert CI → Diagnose → 可选 hotfix Run。  
> **状态：** ✅ 本批完成 · **无新表**（幂等靠 `audit_log` + `ci_*` FirstOrCreate）

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DS-1 | `POST /api/v1/webhooks/github` HMAC（公开路径） | ✅ |
| DS-2 | `workflow_run` failure → upsert + Diagnose | ✅ |
| DS-3 | `autoRun=1` → Create hotfix Run | ✅ |
| DS-4 | `X-GitHub-Delivery` 幂等 + OpenAPI/清单/docs | ✅ |

## 行为

- 公开路径：不走 JWT；`connectionId` 查询参数绑定 `repo_connections`
- 验签：`X-Hub-Signature-256`；密钥优先 `ASH_GITHUB_WEBHOOK_SECRET`，否则 connection `secretId` 明文
- 仅处理 `workflow_run` 且 `action=completed` + `conclusion=failure|timed_out|cancelled`（cancelled 仍诊断、默认不 autoRun）
- 幂等：`audit_log.event_type=ci.webhook_delivery` + `actor_id=deliveryId`
- `autoRun=true|1`：Create `hotfix` / `policyProfile=hotfix`，`issueOrSpec` 含诊断摘要
