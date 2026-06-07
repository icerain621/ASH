# OpenAPI 双文档对齐策略

ASH 同时维护两份 OpenAPI 相关产物，职责不同，**不追求字节级一致**。

## 1. 文档角色

| 产物 | 路径 | 角色 |
|------|------|------|
| **实现真相（SoT）** | `internal/api/docs/swagger.{yaml,json}` | 由 `swaggo/swag` 从 Go handler 注释生成；Worker 通过 `/openapi.json` 对外提供 |
| **产品契约草稿** | `doc/api/openapi-ash-v1.yaml` | 对外/前端联调用的精简契约；含 M0 愿景端点（`/v1/*`）与已落地子集（`/api/v1/*`） |
| **端点边界清单** | `doc/appendices/G-OpenAPI-端点清单(M0).md` | 里程碑级端点范围与验收说明 |

**运行时与集成测试以 swag 输出为准。** 手写 YAML 描述「产品承诺」与「规划路径」，其中 `/api/v1/*` 段落必须在实现中存在。

## 2. 路径前缀约定

- **`/api/v1/*`**：当前平台已实现的真实 Base Path（与 Gin 路由一致）。手写草稿中此前缀下的 path+method **受 `make openapi-check` 强制校验**。
- **`/v1/*`**：M0 产品愿景（Tasks、AgentRuns、旧版 Memories/Spaces 等），**尚未实现或路径已迁移**。仅作规划参考，检查器会统计但不失败。
- **运维面**：`GET /healthz`、`GET /readyz`、`GET /metrics` 等由 swag 覆盖，不要求出现在手写草稿中。

常见迁移对照（规划 → 实现）：

| 契约草稿（规划） | 实现（swag） |
|------------------|--------------|
| `POST /v1/feedback` | `POST /api/v1/feedback` |
| `GET /v1/agent-runs/{runId}` | `GET /api/v1/runs/{runId}` |
| `GET /v1/runs/{runId}/stream` | `GET /api/v1/runs/{runId}/stream` |
| `POST /v1/memories` | `POST /api/v1/memory/candidates` 等治理面 |

## 3. 变更工作流

新增或修改 HTTP API 时按顺序：

1. **实现**：在 `internal/api` 增加 handler，并补齐 swag 注释（`@Router`、`@Summary` 等）。
2. **再生**：`make swagger`（或 `bash scripts/regenerate-swagger.sh`）。
3. **契约**：若该端点属于对外承诺，在 `doc/api/openapi-ash-v1.yaml` 的 `/api/v1/*` 段补充 path/method（可只写摘要 schema，不必复制全部 swag 类型）。
4. **校验**：`make openapi-check`（见下节）。
5. **清单**（可选）：更新 `doc/appendices/G-OpenAPI-端点清单(M0).md` 里程碑说明。

仅内部管理面、短期实验端点可只走步骤 1–2；晋升对外契约时再补步骤 3。

## 4. `make openapi-check` 做什么

`scripts/openapi-check.sh` 执行两项检查：

1. **Swag 确定性**：复制 `internal/api/docs` → 临时目录 → 重新 `swag init` → `diff`，确保已提交的 swagger 与当前代码一致（防止忘记 `make swagger`）。
2. **契约子集**：`go test ./internal/openapicheck -run TestContractMatchesSwagger`，确保手写草稿中每个 `/api/v1/*` 的 HTTP 方法在 `swagger.yaml` 中均存在。

手写草稿应覆盖全部已实现的 `/api/v1/*` 端点；若 swag 新增路由而草稿未更新，`TestContractMatchesSwagger` 会失败。补草稿时可运行：

```bash
OPENAPI_EMIT=1 go test ./internal/openapicheck -run TestEmitContractBackfill -v
```

将输出追加到 `doc/api/openapi-ash-v1.yaml` 的 `paths` 段（勿带入测试日志行）。

## 5. 本地与 CI 建议

```bash
make swagger        # 改 handler 注释后
make openapi-check  # 提交前
make verify         # 含 swagger regen + 主要测试（可在 verify 中追加 openapi-check）
```

### 离线 / 受限网络（Windows Git Bash）

`openapi-check` 使用 `scripts/_swag.sh`：固定 `github.com/swaggo/swag@v1.16.4`（与 `go.mod` 一致，不用 `@latest`）。本地 Go 版本已满足 `go.mod` 时自动 `GOTOOLCHAIN=local`，避免拉取 `golang.org/toolchain`。

**`go.mod` 要求 Go 1.26**：若本机低于 1.26，需先安装/升级 Go，或联网让 `GOTOOLCHAIN=auto` 下载 toolchain。访问 `sum.golang.org` 超时时可试：

```bash
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn
make openapi-check
```

模块已在缓存、仅 sumdb 校验失败时：`ASH_GOSUMDB_OFF=1 make openapi-check`（跳过校验，仅限可信网络环境）。

已安装 `swag` CLI：`ASH_SWAG_USE_PATH=1 make swagger`

## 6. 长期演进

- **P1**：逐步将 `/v1/*` 规划端点从草稿中迁出到独立 `openapi-ash-v2-planned.yaml`，或标注 `deprecated`。
- **P2**：若需对外只发布精简契约，可从 swag 输出过滤生成 `openapi-ash-v1.yaml`（codegen），手写稿转为 overlay；当前阶段以「手写子集 + swag 全量」成本最低。
- **Schema 深度对齐**：不在 `openapi-check` 范围；依赖 handler 测试与前端类型生成。必要时可对关键 DTO 增加 JSON schema 快照测试。
