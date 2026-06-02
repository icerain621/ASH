# ASH Worker（M0）

Go 编排 Worker：**Gin + GORM + SQLite**，对齐 `../doc/design/appendices/` 中的 OpenAPI 与事件协议 v0.1。

## 快速启动

```bash
cd backend
go mod tidy
go run ./cmd/worker
```

## Swagger

```bash
make swagger   # 在本目录
```

OpenAPI 产物：`internal/api/docs/`。
