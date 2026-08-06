# ASH API 错误码表 v1

HTTP 4xx/5xx 响应统一形状（M0）：

```json
{
  "error": {
    "code": "RUN_NOT_FOUND",
    "message": "human readable detail"
  }
}
```

实现：`internal/api/handlers.go` 的 `errorBody`；OpenAPI 类型 `APIErrorResponse`。

机器校验：`go test ./internal/apicodes` 确保 `internal/api` 中每个 `errorBody("CODE", …)` 均登记在 `internal/apicodes/catalog.go`。

## 域划分

| 域 | 说明 | 示例 |
|----|------|------|
| `common` | 通用请求校验 | `INVALID_REQUEST` |
| `auth` | 登录与会话 | `INVALID_CREDENTIALS`, `UNAUTHORIZED` |
| `authz` | RBAC / 空间隔离 | `FORBIDDEN`, `SPACE_ACCESS_DENIED` |
| `runs` | 运行生命周期 | `RUN_NOT_FOUND`, `RUN_NOT_RESUMABLE`, `RUN_NOT_APPROVABLE`, `RUN_NOT_REPLAYABLE`, `ILLEGAL_STATUS_TRANSITION`, `RUN_CANCELED` |
| `rules` | 场景 DSL | `SCENARIO_NOT_FOUND` |
| `memory` | 记忆治理 | `MEMORY_NOT_FOUND`, `MEMORY_REVIEW_FAILED` |
| `rag` | 检索索引 | `RAG_QUERY_FAILED` |
| `model` | 模型路由计量 | `MODEL_USAGE_RECORD_FAILED` |
| `tool` | MCP 工具注册 | `MCP_TOOL_CREATE_FAILED` |
| `improve` | 自我迭代 | `PROPOSAL_NOT_FOUND`, `IMPROVE_PROMOTE_FAILED` |
| `ci` | Repo / Actions | `CI_DIAGNOSE_FAILED`, `CI_PROVIDER_UNAVAILABLE`, `PLAINTEXT_TOKEN_REJECTED` |
| `metrics` | KPI 看板 | `METRICS_OVERVIEW_FAILED` |
| `observability` | 告警 / 追踪 | `ALERT_EVALUATE_FAILED`, `TRACE_LOOKUP_FAILED` |
| `feedback` | 用户反馈 | `FEEDBACK_NOT_FOUND` |
| `platform` | 组织 / 空间 / M2 | `PERMISSION_MATRIX_FAILED`, `INVALID_POLICY` |
| `audit` | 审计导出与保留 | `AUDIT_POLICY_LOCKED` |
| `plugins` | 插件 ABI | `PLUGIN_VERIFY_FAILED` |
| `secrets` | 空间密钥 | `SECRET_NOT_FOUND` |
| `releases` | 发布治理 | `RELEASE_GATE_FAILED` |
| `doctor` | 验证套件 | `DOCTOR_FAILED`, `REPORT_NOT_FOUND` |
| `approvals` | 人工审批 | `APPROVAL_NOT_FOUND` |

完整枚举见 `internal/apicodes/catalog.go`（约 120 项，随 handler 演进同步更新）。
