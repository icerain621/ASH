# ASH Jira 拆解（Epic / Story / Task）

## 1. 使用方式
- 本文按 MVP 范围拆解为可直接录入 Jira 的结构。
- 建议层级：`Epic -> Story -> Task -> Sub-task`。
- 优先级说明：`P0（必须）/ P1（重要）/ P2（增强）`。

## 2. Epic 总览

| Epic ID | Epic 名称 | 目标 | 优先级 | 对应里程碑 |
| --- | --- | --- | --- | --- |
| E-01 | 平台基础与鉴权 | 完成基础框架、账号权限、网关能力 | P0 | W1-W2 |
| E-02 | 助理与任务编排 | 打通任务创建、执行、状态流转 | P0 | W3-W4 |
| E-03 | 记忆体与知识协同 | 实现记忆写入、检索、治理 | P0 | W3-W4 |
| E-04 | 工程自动化链路 | Repo/CI 接入与失败诊断 | P0 | W5-W6 |
| E-05 | 团队空间与协作 | 空间、成员、任务共享 | P1 | W7-W8 |
| E-06 | 反馈与指标看板 | 反馈闭环与 KPI 看板 | P1 | W7-W8 |
| E-07 | 稳定性与发布治理 | 可观测、压测、灰度、发布 | P0 | W9-W10 |

## 3. Epic 详细拆解

## E-01 平台基础与鉴权（P0）

### Story E01-S01：项目基础脚手架
- **验收标准**
  - 前后端工程可本地启动
  - 文档目录、环境变量模板齐全
- **Tasks**
  - T1：创建前端 Vite + React + TS 项目骨架
  - T2：创建后端 Gin 项目骨架（API/Worker）
  - T3：接入 ESLint/Prettier/Go fmt 规范
  - T4：补齐 `doc/10-engineering-setup.md` 对应脚本

### Story E01-S02：登录与权限体系
- **验收标准**
  - JWT 鉴权可用
  - 角色权限（owner/developer/viewer）生效
- **Tasks**
  - T1：实现登录、签发 token、中间件鉴权
  - T2：实现 RBAC 中间件与空间权限校验
  - T3：新增审计日志字段与拦截器

## E-02 助理与任务编排（P0）

### Story E02-S01：任务管理 API
- **验收标准**
  - 可创建/查询/启动/取消任务
  - 状态机流转正确
- **Tasks**
  - T1：实现 `POST /v1/tasks`
  - T2：实现 `GET /v1/tasks/{id}`
  - T3：实现 `POST /v1/tasks/{id}/start`
  - T4：实现 `POST /v1/tasks/{id}/cancel`

### Story E02-S02：Agent 运行编排
- **验收标准**
  - 支持 Planner/Executor/Reviewer 基本流程
  - 异常状态可追踪（BLOCKED/ERROR）
- **Tasks**
  - T1：定义 `agent_runs` 与 `task_steps` 状态机
  - T2：实现编排服务（同步 + 异步任务）
  - T3：实现高风险动作审批接口

### Story E02-S03：执行流实时展示
- **验收标准**
  - 前端可实时查看执行日志与状态推进
- **Tasks**
  - T1：实现 SSE `GET /v1/runs/{id}/stream`
  - T2：前端任务详情页接入 SSE
  - T3：断线重连与异常提示

## E-03 记忆体与知识协同（P0）

### Story E03-S01：记忆 CRUD 与检索
- **验收标准**
  - 支持写入、检索、更新、删除
  - 支持 scope（user/project/session）
- **Tasks**
  - T1：实现 `POST /v1/memories`
  - T2：实现 `POST /v1/memories/search`
  - T3：实现 `PATCH/DELETE /v1/memories/{id}`

### Story E03-S02：记忆治理机制
- **验收标准**
  - TTL 到期处理生效
  - 命中事件可追踪
- **Tasks**
  - T1：实现 `memory_events` 记录
  - T2：实现过期清理与置信度调整任务
  - T3：前端记忆管理页面（置顶、删除、筛选）

## E-04 工程自动化链路（P0）

### Story E04-S01：Repo/CI 连接能力
- **验收标准**
  - 可读取流水线状态与日志
- **Tasks**
  - T1：实现 Repo/CI 连接配置接口
  - T2：实现 pipeline 查询与日志拉取
  - T3：失败重试与超时控制

### Story E04-S02：CI 失败诊断
- **验收标准**
  - 可输出根因与修复建议
- **Tasks**
  - T1：实现 `POST /v1/ci/failures/diagnose`
  - T2：建立高频失败模板与规则
  - T3：前端展示诊断结果与采纳操作

## E-05 团队空间与协作（P1）

### Story E05-S01：空间与成员管理
- **验收标准**
  - 支持空间创建、成员添加、角色分配
- **Tasks**
  - T1：实现 `POST /v1/spaces`
  - T2：实现 `POST /v1/spaces/{id}/members`
  - T3：前端空间管理页面

### Story E05-S02：空间任务协作
- **验收标准**
  - 空间维度可查看任务列表与状态
- **Tasks**
  - T1：实现 `GET /v1/spaces/{id}/tasks`
  - T2：前端空间任务视图

## E-06 反馈与指标看板（P1）

### Story E06-S01：反馈闭环
- **验收标准**
  - 支持评分、分类、文本反馈
- **Tasks**
  - T1：实现 `POST /v1/feedback`
  - T2：前端反馈组件统一接入
  - T3：低分反馈告警规则

### Story E06-S02：KPI 看板
- **验收标准**
  - 核心指标可查询、可按项目/空间过滤
- **Tasks**
  - T1：实现 `GET /v1/metrics/overview`
  - T2：按 `doc/14-kpi-dashboard-definition.md` 建模
  - T3：前端指标看板页

## E-07 稳定性与发布治理（P0）

### Story E07-S01：可观测与告警
- **验收标准**
  - trace/log/metrics 可关联查询
- **Tasks**
  - T1：接入 OTel 与 traceId 透传
  - T2：接入 Prometheus 指标上报
  - T3：配置核心告警规则

### Story E07-S02：发布与灰度
- **验收标准**
  - 支持灰度发布与可回滚
- **Tasks**
  - T1：完善发布脚本与版本说明
  - T2：执行 `doc/11-mvp-release-checklist.md`
  - T3：完成灰度与回滚演练记录

## 4. 跨职能任务（建议独立 Epic 或 Label）
- 测试自动化：接口契约测试、主链路 E2E
- 安全合规：鉴权、越权、审计、密钥治理
- 文档治理：OpenAPI、架构文档、Runbook 持续更新

## 5. Story 模板（可复制到 Jira）
```txt
Title: [ASH][模块] xxx
Description:
  - 背景：
  - 目标：
  - 范围：
  - 非目标：
Acceptance Criteria:
  1) ...
  2) ...
  3) ...
Tech Notes:
  - 依赖：
  - 风险：
  - 回滚：
Priority: P0/P1/P2
Estimate: x 人天
```

## 6. 排期建议（对应 10 周计划）
- W1-W2：E-01
- W3-W4：E-02、E-03
- W5-W6：E-04
- W7-W8：E-05、E-06
- W9-W10：E-07 + 全链路收敛

## 7. 完成定义（DoD）
- 代码合并并通过 CI
- 验收标准全部满足
- 文档更新完成（API/架构/联调）
- 可观测埋点与告警生效
- 至少 1 名非开发角色可复现验证
