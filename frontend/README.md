# ASH Web Console (M0)

零构建静态控制台，由 Worker 挂载在 `/ui/`。

## 功能

- **Runs**：列表、创建 feature_delivery run、SSE 事件流、artifacts
- **Memory**：候选列表、approve/reject、新建候选（可绑 runId 推 SSE）
- **Doctor**：一键 TR0

## 启动

```bash
# 仓库根目录
make run
# 打开 http://localhost:8080/ui/
```

环境变量 `ASH_WEB_DIR` 默认 `frontend/public`。

## 后续

可迁移至 Vite + React（见 `doc/08-frontend-architecture.md`）。
