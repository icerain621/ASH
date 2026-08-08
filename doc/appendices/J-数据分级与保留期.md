# 附录 J：数据分级与保留期（PRD §8）

> 状态：v0.1（2026-08-08）· 关闭 PRD-8 设计 TODO  
> 实现：`internal/config/data_policy.go` · `GET /api/v1/data-policy` · retention apply APIs

## 1. 数据分级（classification）

| 级别 | 含义 | 典型载体 | 外发观测 | 控制台展示 |
|------|------|----------|----------|------------|
| `normal` | 一般业务数据 | 普通事件 payload、公开产物摘要 | 允许（仍走 denyKeys） | 明文 |
| `restricted` | 组织内部敏感 | 评审意见、内部 ADR、受限记忆 | 默认脱敏后可外发 | 默认脱敏键 |
| `secret` | 密钥/凭证类 | token、私钥、含 secret 的记忆 | **禁止外发**；仅审计索引 | 强制 `RedactJSON` |

记忆层已落库 `sensitivity` 枚举（附录 C）；跨域策略以本表为准。

## 2. 默认保留期

| 域 | 默认 | 环境变量 | 应用方式 |
|----|------|----------|----------|
| Run 事件 (`run_events`) | **90d** | `ASH_RETENTION_EVENTS_DAYS` | `POST /api/v1/events/retention/apply`（按 run `created_at`） |
| 审计 (`audit_logs`) | **365d** | `ASH_RETENTION_AUDIT_DAYS`（新建 policy 默认） | `PUT /audit/policy` + `POST /audit/retention/apply` |
| 记忆 | L1 **90d** / L2 **365d** | `ASH_MEMORY_TTL_*` | TTL sweep → **弃用**（不硬删 body） |
| 产物索引 (`artifact_index`) | **30d** 或最近 **200** runs | `ASH_RETENTION_ARTIFACTS_DAYS` / `ASH_RETENTION_ARTIFACTS_MAX_RUNS` | `POST /api/v1/artifacts/retention/apply` |

非法或空环境变量回退到上表默认值。运行时快照：`GET /api/v1/data-policy`、`/readyz`、`/api/v1/scale/readiness`。

## 3. 样例脱敏规则

输入（节选）：

```json
{"token":"ghp_aaa","Authorization":"Bearer xyz","note":"ok"}
```

`security.RedactJSON` 输出（模式）：

```json
{"token":"***","Authorization":"***","note":"ok"}
```

观测配置另见 `config/ash-observability.yaml` 的 `redaction.denyKeys`（外发插件要求开启 redaction）。

## 4. 审计导出流程（已实现，PRD DoD）

1. （可选）`PUT /api/v1/audit/policy` 设置 `retentionDays` / `redactPayload`  
2. `GET /api/v1/compliance/secret-scan` 扫描审计与事件载荷  
3. `POST /api/v1/compliance/export` 导出 Doctor + secretScan +（可脱敏）审计日志  
4. 控制台：`/ui/compliance` 一键开启 redact；Automation 可触发 audit retention apply  

## 5. 验收对照

| PRD 验收项 | 证据 |
|------------|------|
| 分级表 | 本附录 §1 |
| 样例脱敏 | §3 + `internal/security` |
| 审计导出流程 | §4 + compliance API |
| 保留期可配置 | §2 + `data_policy` + retention apply |
