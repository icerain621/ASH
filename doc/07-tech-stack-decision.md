# ASH 前后端技术选型决策文档

## 1. 选型目标
- 简单易上手：新成员可在 1-3 天内参与开发与提测
- 风格简洁：默认视觉统一、组件复用高、设计负担低
- 易扩展：支持后续工作台、空间、记忆体、智能体、自动化链路持续扩展
- 后端 Go 语言：保证并发能力、稳定性与交付效率

## 2. 前端技术选型

## 2.1 推荐方案（MVP）
- 框架：React + TypeScript + Vite
- 路由：React Router
- 状态管理：Zustand
- 数据请求：TanStack Query
- UI：Tailwind CSS + shadcn/ui
- 表单与校验：React Hook Form + Zod
- 图标与图表：Lucide React + Recharts
- 规范：ESLint + Prettier

## 2.2 选型依据
- React + Vite：生态成熟、模板丰富、启动与构建速度快，上手成本低。
- TypeScript：对中长期扩展和多人协作友好，能提前暴露边界问题。
- Zustand：API 简洁、样板代码少，适合 ASH 当前中等复杂度状态场景。
- TanStack Query：内置缓存/重试/轮询策略，天然适配任务状态流和执行看板。
- Tailwind + shadcn/ui：风格简洁统一、组件可定制，避免重型 UI 框架耦合。
- RHF + Zod：表单开发快，校验规则可读可复用。

## 2.3 前端备选与不选理由

| 类型 | 备选 | 不作为首选的原因 |
| --- | --- | --- |
| 框架 | Vue 3 + Vite | 团队若以 React 经验为主，迁移和统一成本更高 |
| 状态管理 | Redux Toolkit | 规范完整但样板较多，MVP 阶段心智负担偏高 |
| UI 库 | Ant Design | 功能全面但默认视觉偏重，不利于“极简风格”目标 |
| 全栈框架 | Next.js | SSR/路由约束更复杂，当前后台工作台类场景收益不明显 |

## 3. 后端技术选型（Go）

## 3.1 推荐方案（MVP）
- Web 框架：Gin
- API 协议：REST + SSE（必要时补 WebSocket）
- 数据库：MySQL 8
- 缓存：Redis
- 队列/异步任务：Asynq
- ORM：GORM（关键路径可混合手写 SQL）
- 配置：Viper
- 日志：Zap
- 可观测性：OpenTelemetry + Prometheus + Grafana
- 鉴权：JWT + RBAC（后续可接 SSO）

## 3.2 选型依据
- Gin：学习曲线平缓、生态完善、开发速度快，适合 MVP 快速交付。
- REST + SSE：实现复杂度低，能满足任务流与实时状态更新。
- MySQL + Redis：稳定成熟、运维经验广、成本可控。
- Asynq：适配 Agent 执行、重试、延迟任务和失败补偿等工程自动化场景。
- GORM：提升开发效率，兼顾复杂查询时的灵活优化空间。
- OTel + Prometheus：保证任务链路可观测，便于定位失败与性能瓶颈。

## 3.3 后端备选与不选理由

| 类型 | 备选 | 不作为首选的原因 |
| --- | --- | --- |
| Web 框架 | Fiber | 性能优秀但团队经验与生态通用性通常不如 Gin |
| 框架体系 | GoFrame | 工程化完整，但框架绑定较强，迁移和学习成本更高 |
| 通信协议 | gRPC 全覆盖 | 联调门槛更高，前端直连与调试复杂度上升 |
| 消息队列 | Kafka | 运维与部署复杂度较高，MVP 阶段性价比不足 |

## 4. 前后端协作规范建议
- 接口优先：以 `doc/api/openapi-ash-v1.yaml` 为单一契约源。
- 错误码统一：业务错误码、鉴权错误码、系统错误码分层。
- 实时更新规范：任务详情走 REST，执行日志走 SSE。
- 可观测标准：关键接口必须携带 traceId，任务事件全量记录。

## 5. 阶段性架构建议
- 阶段 1（MVP）：模块化单体优先（便于快速迭代）
- 阶段 2（增长期）：按域拆分服务（assistant/memory/automation）
- 阶段 3（规模化）：再引入更重的中间件和治理体系

## 6. 最终结论
- 前端采用：React + TypeScript + Vite + Zustand + TanStack Query + Tailwind + shadcn/ui
- 后端采用：Gin + MySQL + Redis + Asynq + GORM + Zap + OpenTelemetry
- 该组合最符合 ASH 当前“简单易上手、界面简洁、可扩展、偏工程自动化”的目标。
