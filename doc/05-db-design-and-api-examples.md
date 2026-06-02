# ASH 数据库字段级设计 + API 示例（联调版）

## 1. 核心数据表（MVP）
- 用户与空间：`users`, `spaces`, `space_members`
- 任务执行：`tasks`, `agent_runs`, `task_steps`
- 记忆体系：`memories`, `memory_events`
- 工具与反馈：`tool_calls`, `feedback_records`

## 2. 字段设计原则
- 全表统一 `created_at` / `updated_at`
- 所有关键状态使用枚举字符串，服务端做流转校验
- JSON 字段用于保存计划、风险、工具请求/响应
- 必要索引覆盖：状态查询、时间查询、目标查询、组合唯一约束

## 3. 关键约束
- Task 与 AgentRun 必须遵循状态机流转
- 高风险动作进入 `NEED_APPROVAL`
- 记忆体支持 TTL 过期与置信度管理
- 低评分反馈触发置信度调整候选事件

## 4. API 示例

### 4.1 创建任务
```json
POST /v1/tasks
{
  "title": "实现用户登录重构并补测试",
  "goal": "将登录模块改为token刷新机制，并确保CI通过",
  "projectId": "ask-core",
  "spaceId": 1001,
  "priority": "high",
  "mode": "auto"
}
```

### 4.2 启动任务
```json
POST /v1/tasks/90001/start
{
  "riskLevel": "medium",
  "toolPolicy": {
    "allow": ["repo.read", "repo.write", "ci.read", "kbase.search"],
    "requireApprovalFor": ["repo.write:main", "deploy.trigger"]
  }
}
```

### 4.3 写入记忆
```json
POST /v1/memories
{
  "scope": "project",
  "projectId": "ask-core",
  "content": "登录模块必须保留refresh_token过期回收逻辑",
  "tags": ["auth", "token", "backend"],
  "source": "agent",
  "confidence": 0.84
}
```

### 4.4 CI 失败诊断
```json
POST /v1/ci/failures/diagnose
{
  "pipelineId": "pipe_123456",
  "repoRef": "feature/login-refresh",
  "logRange": {"from": 0, "to": 1200}
}
```

### 4.5 提交反馈
```json
POST /v1/feedback
{
  "targetType": "agent_run",
  "targetId": "88001",
  "rating": 4,
  "category": "executability",
  "comment": "建议可执行，修复路径清晰"
}
```

## 5. 联调顺序建议
1. 任务与执行流：`/tasks` + `/agent-runs` + `/runs/{id}/stream`
2. 记忆体系：`/memories` + `/memories/search`
3. 工程诊断与反馈：`/ci/failures/diagnose` + `/feedback`
4. 团队空间与指标：`/spaces` + `/metrics/overview`
