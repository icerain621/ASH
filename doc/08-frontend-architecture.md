# ASH 前端架构设计（可开工版）

## 1. 设计目标
- 简单易上手：降低新成员理解与改动成本
- 风格简洁：统一页面骨架与组件风格
- 易扩展：按业务域拆分，支持后续模块增加
- 高可维护：状态边界清晰、接口契约稳定

## 2. 技术栈（与选型一致）
- React + TypeScript + Vite
- React Router
- Zustand（客户端状态）
- TanStack Query（服务端状态）
- Tailwind CSS + shadcn/ui
- React Hook Form + Zod
- ESLint + Prettier

## 3. 目录结构建议
```txt
frontend/
  src/
    app/
      router/
      providers/
      layout/
    pages/
      assistant/
      space/
      memory/
      automation/
      settings/
    modules/
      assistant/
        components/
        hooks/
        store/
        api/
        types/
      space/
      memory/
      automation/
      feedback/
    shared/
      components/
      ui/
      hooks/
      utils/
      constants/
      types/
    services/
      http/
      sse/
      auth/
    styles/
      globals.css
  index.html
  vite.config.ts
```

## 4. 分层与职责
- `app`：应用级基础设施（路由、Provider、布局）
- `pages`：页面入口，组合业务模块，不写复杂业务逻辑
- `modules`：核心业务域（assistant/space/memory/automation/feedback）
- `shared`：跨域复用能力（组件、hooks、工具）
- `services`：网络与外部交互（HTTP/SSE/Auth）

## 5. 页面与路由规划（MVP）
- `/assistant`：个人工作台（任务列表、任务详情、执行日志）
- `/space`：团队空间（成员管理、任务共享）
- `/memory`：记忆管理（检索、置顶、删除）
- `/automation`：CI 诊断与修复建议
- `/settings`：账号、偏好、API 配置

## 6. 状态管理策略

### 6.1 TanStack Query（服务端状态）
- 管理 API 数据：任务详情、执行状态、空间成员、反馈统计
- 统一查询键命名：`['tasks', taskId]`、`['runs', runId]`
- 使用 `staleTime` 控制缓存策略，减少重复请求

### 6.2 Zustand（客户端状态）
- 管理 UI 状态：侧栏折叠、当前空间、筛选条件、弹窗状态
- 不存储可由 Query 获取的远端数据，避免双写冲突

## 7. 接口层规范
- 统一请求入口：`services/http/client.ts`
- 所有 API 由模块内 `api` 文件夹导出，页面不直连底层 HTTP
- 错误处理统一在拦截器做基础归一（鉴权失败、网络异常）
- 契约来源：运行时 `/openapi.json`（swag）；产品草稿 `doc/api/openapi-ash-v1.yaml`（见 `doc/api/openapi-alignment.md`）

## 8. 实时能力（任务执行流）
- 任务详情与步骤结果：REST 获取
- 执行日志与状态推进：SSE 订阅 `/v1/runs/{runId}/stream`
- 断线重连：前端最多重试 3 次，失败后提示手动刷新

## 9. UI 设计规范（简洁风格）
- 页面布局：顶部导航 + 左侧模块导航 + 主内容区
- 组件风格：优先使用 shadcn/ui 原子组件，限制自定义样式分叉
- 色彩策略：中性色为主，状态色仅用于成功/警告/失败
- 卡片策略：任务卡、步骤卡、反馈卡统一结构与间距

## 10. 表单与校验规范
- 统一使用 React Hook Form 管理表单状态
- 使用 Zod 定义表单 schema，与 API 参数结构对齐
- 表单错误提示一致化：字段下方短句，避免弹窗堆叠

## 11. 工程规范
- 命名：
  - 组件：PascalCase
  - hooks：`useXxx`
  - store：`xxx.store.ts`
  - API：`xxx.api.ts`
- 单文件建议不超过 300 行，超过后拆分 hooks/components
- 严禁页面层直接操作全局状态与网络请求实现细节

## 12. 测试建议（MVP）
- 单元测试：Vitest + Testing Library（核心 hooks、store、工具函数）
- 集成测试：关键流程（创建任务 -> 启动 -> 查看日志）
- E2E（可选）：Playwright 覆盖主链路冒烟

## 13. 开发节奏建议
- 第 1 周：完成框架、路由、布局、基础组件
- 第 2 周：assistant 与任务流页面
- 第 3 周：memory 与 automation 页面
- 第 4 周：space 与 feedback 页面 + 联调修复

## 14. 与后端协作约定
- 后端字段命名采用 camelCase 返回
- 状态枚举值由后端统一定义，前端仅做展示映射
- 错误码由后端统一维护，前端按码表做提示映射
- 任务链路必须透传 traceId，便于日志排查
