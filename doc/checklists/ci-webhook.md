# CI Webhook → Diagnose → Run（Sprint DS）

> 验收：GitHub Actions `workflow_run` 失败经 HMAC webhook 进入 ASH。

## 前置

1. 已创建 `repo_connections`（`owner`/`repo`/`secretId`）
2. Secret 明文用作 HMAC 密钥，或设置 `ASH_GITHUB_WEBHOOK_SECRET`
3. Worker 可达：`http://localhost:8080`

## 步骤

| # | 动作 | 期望 |
|---|------|------|
| 1 | 配置 GitHub webhook → `POST /api/v1/webhooks/github?connectionId=<id>`，Content-Type JSON，Secret = HMAC 密钥 | GitHub 投递成功 |
| 2 | 失败 workflow 投递（`X-Hub-Signature-256` + `X-GitHub-Delivery`） | `200`；`diagnosis.rootCause` 非空；`ciRunId`/`ciJobId` 有值 |
| 3 | 同 `X-GitHub-Delivery` 重放 | `duplicate=true`，不重复诊断 |
| 4 | `?autoRun=1&repoRoot=<path>` 失败投递 | `shouldStartRun=true`；`ashRunId` 返回（hotfix） |
| 5 | 错误签名 | `401 WEBHOOK_SIGNATURE_INVALID` |

## curl 示例（本地）

```bash
BODY='{"action":"completed","workflow_run":{"id":1,"name":"CI","status":"completed","conclusion":"failure","run_attempt":1,"html_url":"https://example/run/1","head_branch":"main","head_sha":"abc"},"repository":{"name":"REPO","owner":{"login":"OWNER"}}}'
SIG="sha256=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "$WH_SECRET" | awk '{print $2}')"
curl -sS -X POST "http://localhost:8080/api/v1/webhooks/github?connectionId=$CONN_ID" \
  -H "Content-Type: application/json" \
  -H "X-Hub-Signature-256: $SIG" \
  -H "X-GitHub-Event: workflow_run" \
  -H "X-GitHub-Delivery: $(uuidgen)" \
  -d "$BODY"
```

## 相关

- `internal/ci/webhook.go` · `internal/api/webhook.go`
- OpenAPI：`/api/v1/webhooks/github`
- 计划：`doc/plan/sprint-ds-webhook.md`
