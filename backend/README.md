# ASH Worker（M0）

> **说明**：现行源码在仓库根目录 `cmd/` + `internal/`（不必再迁入本目录）。本 README 仅作历史指引。

Go 编排 Worker：**Gin + GORM + SQLite/Postgres**，对齐 `../doc/appendices/` 与 `../doc/api/openapi-ash-v1.yaml`。

## 快速启动（仓库根）

```bash
cd ..
make tidy
make run
```

## Swagger

```bash
make swagger   # 在本目录
```

OpenAPI 产物：`internal/api/docs/`。
