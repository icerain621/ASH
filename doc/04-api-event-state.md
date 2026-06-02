# ASH 模块级接口清单（API / 事件 / 状态机）

## 1. 服务边界
- gateway-api：网关、鉴权、限流
- assistant-service：任务入口与会话
- agent-orchestrator：Planner/Executor/Reviewer 编排
- memory-service：记忆读写检索
- automation-gateway：Repo/CI/Artifact 连接
- space-service：空间与成员管理
- feedback-service：反馈与指标

## 2. 核心 API（MVP）

### 2.1 Tasks
- `POST /v1/tasks`：创建任务
- `GET /v1/tasks/{taskId}`：任务详情
- `POST /v1/tasks/{taskId}/start`：启动任务
- `POST /v1/tasks/{taskId}/cancel`：取消任务
- `GET /v1/runs/{runId}/stream`：执行流（SSE/WS）

### 2.2 Agent Runs
- `POST /v1/agent-runs`：创建执行实例
- `GET /v1/agent-runs/{runId}`：执行详情
- `POST /v1/agent-runs/{runId}/approve-step`：审批高风险步骤

### 2.3 Memories
- `POST /v1/memories`：写入记忆
- `POST /v1/memories/search`：检索记忆
- `PATCH /v1/memories/{id}`：更新记忆
- `DELETE /v1/memories/{id}`：删除记忆
- `POST /v1/memories/{id}/pin`：置顶记忆

### 2.4 Automation
- `POST /v1/integrations/repo/connect`
- `POST /v1/integrations/ci/connect`
- `GET /v1/ci/pipelines/{pipelineId}`
- `GET /v1/ci/pipelines/{pipelineId}/logs`
- `POST /v1/ci/failures/diagnose`

### 2.5 Spaces & Feedback
- `POST /v1/spaces`
- `POST /v1/spaces/{spaceId}/members`
- `GET /v1/spaces/{spaceId}/tasks`
- `POST /v1/feedback`
- `GET /v1/metrics/overview`

## 3. 事件定义（MVP）
- `task.created`
- `task.started`
- `agent.plan.generated`
- `agent.step.started`
- `agent.step.succeeded`
- `agent.step.failed`
- `agent.run.completed`
- `memory.saved`
- `ci.pipeline.failed`
- `ci.failure.diagnosed`
- `feedback.submitted`

事件头建议统一：`eventId`, `eventType`, `occurredAt`, `traceId`, `actorId`, `projectId`, `spaceId`

## 4. 状态机定义

### 4.1 Task 状态
`CREATED -> READY -> RUNNING -> NEED_APPROVAL -> RUNNING -> COMPLETED`

异常分支：
- `RUNNING -> FAILED`
- `RUNNING -> CANCELED`

### 4.2 Agent Run 状态
`PLANNING -> EXECUTING -> REVIEWING -> DONE`

异常分支：
- `EXECUTING -> BLOCKED`
- `EXECUTING -> NEED_HUMAN`
- `ANY -> ERROR`

## 5. 权限模型（MVP）
- owner：空间与集成管理，高风险审批
- developer：任务执行与反馈
- viewer：只读查看
