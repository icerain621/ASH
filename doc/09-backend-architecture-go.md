# ASH 后端架构设计（Go，可开工版）

## 1. 设计目标
- 快速交付：MVP 阶段优先可实现、可联调、可观测
- 易维护：分层明确，避免业务逻辑散落在 handler
- 易扩展：按业务域演进，支持后续服务拆分
- 稳定可靠：任务可重试、可追踪、可回放排查

## 2. 技术栈（推荐落地）
- Go 1.22+
- Gin（HTTP 框架）
- GORM（ORM）+ 手写 SQL（关键路径）
- MySQL 8（主存储）
- Redis（缓存 + 队列）
- Asynq（异步任务）
- Zap（结构化日志）
- OpenTelemetry + Prometheus + Grafana（可观测）
- JWT + RBAC（鉴权）

## 3. 后端目录结构建议
```txt
backend/
  cmd/
    ash-api/
      main.go
    ash-worker/
      main.go
  internal/
    app/
      bootstrap/
      config/
      server/
      middleware/
    domain/
      task/
      agent/
      memory/
      space/
      feedback/
      automation/
    service/
      tasksvc/
      agentsvc/
      memorysvc/
      spacesvc/
      feedbacksvc/
      autosvc/
    repository/
      mysql/
      redis/
    transport/
      http/
        handler/
        dto/
        router/
      sse/
    worker/
      jobs/
      scheduler/
    pkg/
      logx/
      trace/
      errors/
      response/
      auth/
  migrations/
  scripts/
  go.mod
```

## 4. 分层职责
- `transport`：请求解析、参数校验、响应封装，不写业务逻辑
- `service`：业务编排与事务边界
- `domain`：核心模型、状态机、领域规则
- `repository`：数据持久化与查询优化
- `worker`：异步任务执行、重试与补偿
- `pkg`：跨模块公共能力

## 5. 服务边界（与前序文档一致）
- `assistant-service`：任务入口、任务查询、执行历史
- `agent-orchestrator`：Planner/Executor/Reviewer 编排、风险审批
- `memory-service`：记忆写入/检索/治理（TTL/置信度）
- `automation-gateway`：Repo/CI/Artifact 接入与诊断
- `space-service`：团队空间与成员权限
- `feedback-service`：反馈采集与指标聚合

## 6. 关键流程设计

## 6.1 任务执行主流程
1. `POST /v1/tasks` 创建任务
2. `POST /v1/tasks/{id}/start` 启动任务
3. 写入 `agent_runs`，状态进入 `PLANNING`
4. Planner 生成步骤后进入 `EXECUTING`
5. Executor 按步骤调工具（Repo/CI/KBase/MCP）
6. Reviewer 汇总结果与风险，状态 `DONE/NEED_HUMAN/ERROR`
7. 执行结果通过 SSE 推送前端

## 6.2 异步队列策略（Asynq）
- 队列划分：
  - `critical`：任务核心执行、状态推进
  - `default`：日志聚合、诊断分析
  - `low`：反馈统计、记忆降权候选
- 重试策略：
  - 指数退避，最大 5 次
  - 超过阈值进入死信队列并报警

## 7. 状态机落地规范
- Task 状态：`CREATED -> READY -> RUNNING -> NEED_APPROVAL -> COMPLETED`
- AgentRun 状态：`PLANNING -> EXECUTING -> REVIEWING -> DONE`
- 所有状态流转封装在 `domain/*/state_machine.go`
- 非法流转返回业务错误码，不允许直接更新 DB 绕过校验

## 8. API 规范
- 接口契约以实现为准（`internal/api/docs`，`make swagger`）；产品草稿 `doc/api/openapi-ash-v1.yaml` 与校验见 `doc/api/openapi-alignment.md`
- 返回结构统一：
```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```
- 错误码分层：
  - `1xxx` 参数/校验错误
  - `2xxx` 鉴权与权限错误
  - `3xxx` 业务状态错误
  - `5xxx` 系统与依赖错误

## 9. 数据访问规范
- Repository 不暴露 GORM 对象到 service 层
- 复杂查询使用 SQL Builder/手写 SQL，避免 ORM 生成低效语句
- 所有写操作必须带 `updated_at` 更新与审计上下文（actor/trace）
- 事务原则：只在 service 层开启

## 10. 安全与权限
- JWT 中包含：`userId`、`roles`、`spaceScopes`
- 中间件校验：登录、角色、空间权限、敏感动作审批
- 高风险动作（主干写入/发布触发）必须二次确认

## 11. 可观测性规范
- 日志字段统一：`traceId`, `taskId`, `runId`, `userId`, `spaceId`
- 指标建议：
  - `task_run_duration_seconds`
  - `agent_step_failure_total`
  - `ci_diagnose_latency_ms`
  - `memory_hit_ratio`
- 链路：API -> Service -> Queue -> Worker 全链路透传 traceId

## 12. 配置与环境
- 配置来源：环境变量 + 配置文件（Viper）
- 环境划分：`dev` / `test` / `prod`
- 必要配置：
  - MySQL DSN
  - Redis Addr
  - JWT Secret
  - Asynq Queue Config
  - OTel Exporter Endpoint

## 13. 测试策略
- 单元测试：状态机、service 规则、repository 关键查询
- 集成测试：任务全链路、CI 诊断、记忆读写
- 回归测试：接口契约与错误码稳定性
- 压测：任务并发执行、SSE 连接稳定性

## 14. 发布与运维建议
- 发布方式：API 与 Worker 分离部署
- 灰度策略：按 space/project 开关放量
- 回滚策略：版本回滚 + 队列暂停 + 幂等重放
- 运维看板：接口错误率、队列积压、任务成功率、平均耗时

## 15. 演进路径
- 阶段 1：模块化单体（当前）
- 阶段 2：按域拆服务（assistant/memory/automation）
- 阶段 3：引入统一网关治理与多 Agent 并行调度优化
