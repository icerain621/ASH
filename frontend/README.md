# ASH 控制台前端

Vite + React + TypeScript 前端控制台，路由 basepath 为 `/ui/`。

## 技术栈

- TanStack Router：页面路由与 `/ui/` basepath
- TanStack Query：服务端状态与接口缓存
- TanStack Table：运行表格渲染
- lucide-react：操作图标

- **运行**：列表、创建 feature_delivery 运行、SSE 事件流、产物
- **记忆**：候选列表、通过/拒绝、新建候选（可绑定运行 ID 推送 SSE）
- **诊断**：一键 TR0

## 启动

```bash
cd frontend
npm install
npm run dev
```

打开 `http://127.0.0.1:5173/ui/`。

后端 API 通过 Vite proxy 转发到 `http://localhost:8080`。
