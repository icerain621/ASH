# Agent Chat DSH 对标 — 后端缺失功能清单

> Status: **complete + polish**（2026-09-16）  
> Spec: [`2026-09-15-agent-chat-dsh-parity-design.md`](./2026-09-15-agent-chat-dsh-parity-design.md)  
> 原则：不引入 Cordis / 不嵌入 `dsh web`

## 进度

主路径已收口；续推交互/设置一等比对：

| 项 | 状态 |
|----|------|
| P0–P5 / MCP / 密度 / purge / 栏宽 / stop / DnD / Markdown | ✅ |
| Tools / MCP / **Skills** 一等设置面板 | ✅ |
| Composer **Enter 发送 / Shift+Enter 换行** + slash ↑↓ 键盘 | ✅ |
| Skills **pack 验签/安装 + 组织 Catalog**（Chat 设置内） | ✅ |
| Chat **工具卡合并**（called+result）+ Tools 目录筛选 | ✅ |
| Cordis / 完整 slash 目录一比一 | ❌ 明确不做 |

## 仍可选（非阻塞）

| 项 | 说明 |
|----|------|
| Tools 运行时开关 | 内置风险目录仍只读；需 space/scenario 策略 API 才可真正启停 |

## 修订

| 日期 | 说明 |
|------|------|
| 2026-09-16 | Markdown 表格/任务列表 |
| 2026-09-16 | Skills 面板 + Composer 键盘交互贴 DSH |
| 2026-09-16 | Skills 面板接入 pack verify/install 与 org catalog |
| 2026-09-16 | tool.called/result 合并为工具卡；Tools 面板搜索/风险筛选 |
