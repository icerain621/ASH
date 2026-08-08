# ASH 工程启动与联调手册（前后端）

## 1. 目标
- 提供统一的本地开发启动方式
- 降低前后端联调门槛
- 固化环境变量与排障流程，提升协作效率

## 2. 开发环境要求

### 2.1 基础依赖
- Go：`1.22+`
- Node.js：`20+`
- pnpm：`9+`（或 npm/yarn，建议统一 pnpm）
- MySQL：`8.0+`
- Redis：`7+`
- Git：`2.40+`

### 2.2 可选工具
- Air（Go 热重载）
- Docker / Docker Compose（本地依赖容器化）
- Make（命令统一入口）

## 3. 推荐工程目录（示例）
```txt
ash/
  backend/
  frontend/
  doc/
  scripts/
  .env.example
```

## 4. 环境变量模板

## 4.1 后端 `.env.backend.example`
```env
APP_ENV=dev
APP_PORT=8080

MYSQL_DSN=root:root@tcp(127.0.0.1:3306)/ash?charset=utf8mb4&parseTime=True&loc=Local
REDIS_ADDR=127.0.0.1:6379
REDIS_PASSWORD=
REDIS_DB=0

JWT_SECRET=replace_with_strong_secret
JWT_EXPIRE_HOURS=72

ASYNQ_CONCURRENCY=10
ASYNQ_QUEUE_CRITICAL=critical
ASYNQ_QUEUE_DEFAULT=default
ASYNQ_QUEUE_LOW=low

OTEL_ENABLED=false
OTEL_EXPORTER_OTLP_ENDPOINT=
```

## 4.2 前端 `.env.frontend.example`
```env
VITE_APP_ENV=dev
VITE_API_BASE_URL=http://127.0.0.1:8080
VITE_SSE_BASE_URL=http://127.0.0.1:8080
```

## 5. 数据库初始化

### 5.1 创建数据库
```sql
CREATE DATABASE ash DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
```

### 5.2 导入表结构
使用 `doc/db/ash_mvp_schema.sql` 初始化：
```bash
mysql -uroot -proot ash < doc/db/ash_mvp_schema.sql
```

## 6. 后端启动（Go）

## 6.1 安装依赖
```bash
cd backend
go mod tidy
```

## 6.2 启动 API 服务
```bash
go run ./cmd/ash-api
```

## 6.3 启动 Worker
```bash
go run ./cmd/ash-worker
```

## 6.4 热重载（可选）
```bash
air -c .air.toml
```

## 7. 前端启动（React + Vite）

## 7.1 安装依赖
```bash
cd frontend
pnpm install
```

## 7.2 启动开发环境
```bash
pnpm dev
```

## 7.3 构建与预览
```bash
pnpm build
pnpm preview
```

## 8. 前后端联调顺序（推荐）
1. 启动 MySQL/Redis
2. 初始化数据库表结构
3. 启动后端 API + Worker
4. 启动前端并确认登录态
5. 联调主链路：
   - 创建任务 `POST /v1/tasks`
   - 启动任务 `POST /v1/tasks/{taskId}/start`
   - 监听执行流 `GET /v1/runs/{runId}/stream`
   - 调试记忆体 `/v1/memories`、`/v1/memories/search`
   - 调试诊断 `/v1/ci/failures/diagnose`

## 9. 本地联调检查清单
- API 健康检查可访问（如 `/healthz`）
- 数据库连接成功，关键表可读写
- Redis 连通，任务队列可消费
- 前端能正确读取 `VITE_API_BASE_URL`
- SSE 流可连通且断线可重连

## 10. 常见问题与排障

### 10.1 数据库连接失败
- 检查 DSN、账号密码、端口、数据库名
- 确认 MySQL 已开启并允许本地连接

### 10.2 Redis 队列无消费
- 确认 Worker 进程已启动
- 检查 Asynq 队列名与配置是否一致

### 10.3 前端请求 401
- 检查 JWT 配置与登录流程
- 检查请求头 `Authorization: Bearer <token>`

### 10.4 SSE 无实时输出
- 检查后端是否正确 flush
- 检查反向代理是否禁用缓冲（生产环境）

## 11. 代码质量与提交流程（建议）
- 后端：
  - `go test ./...`
  - `go vet ./...`
- 前端：
  - `pnpm lint`
  - `pnpm test`（若已配置）
- 提交前执行最小冒烟：任务创建 -> 启动 -> 日志流可见

## 12. 生产前准备（MVP）
- 替换所有默认密钥与账号密码
- 开启日志采集与监控告警
- 设置数据库备份策略
- 确认回滚脚本可用
