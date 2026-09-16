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
| Slash **点选填入不发送**（可补参数） | ✅ |
| 工作区 **改名/关闭** + 侧栏会话搜索 | ✅ |
| Transcript **贴底滚动** + 助手 **已停止** 徽章 | ✅ |
| **可展开工具卡** IN/OUT + Details 结构化 + fence复制 | ✅ |
| Composer 旁 **Tools / MCP / Skills** 快捷入口 | ✅ |
| Tools **会话级运行时启停**（`disabledTools`） | ✅ |
| Chat **过滤 step.***（仅 Trajectory）+ 气泡↔Trajectory **seq 同步** | ✅ |
| Permission **full 确认** + 侧栏 **显示已关闭** | ✅ |
| Cordis / 完整 slash 目录一比一 | ❌ 明确不做 |

## 仍可选（非阻塞）

| 项 | 说明 |
|----|------|
| （暂无阻塞项） | 主路径与文档可选 polish 已收口 |

## 修订

| 日期 | 说明 |
|------|------|
| 2026-09-16 | Markdown 表格/任务列表 |
| 2026-09-16 | Skills 面板 + Composer 键盘交互贴 DSH |
| 2026-09-16 | Skills 面板接入 pack verify/install 与 org catalog |
| 2026-09-16 | tool.called/result 合并为工具卡；Tools 面板搜索/风险筛选 |
| 2026-09-16 | slash fill-not-fire；工作区改名/关闭；侧栏搜索；贴底+stopped |
| 2026-09-16 | 可展开工具卡 / Details 工具段 / Composer 快捷入口 / fence 复制 |
| 2026-09-16 | session `disabledTools` + Tools 面板启停 + run 执行拦截 |
| 2026-09-16 | Chat 去 step；气泡 seq→Trajectory；full 确认；includeClosed |
