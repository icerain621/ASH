# ASH 吸收交互原型（收敛版 · 记忆全页）

> 状态：中保真可切换原型 · 2026-09-17  
> 依据：[platform-quad-comparison.md](../../plan/platform-quad-comparison.md) · [三主题 IA](../../../docs/superpowers/specs/2026-09-14-console-three-pillar-ia-design.md)  
> 对照实现：`MemoryPage` · `KnowledgePanel` · `MemoryLinkPanel`

## 打开

浏览器打开 [`index.html`](./index.html)。顶栏切换 **Agent · 记忆 · 管控&评审**；记忆内用「记忆体 | 知识」Tab + 左侧子导航。

深链示例：`#memory/layer` · `#memory/know-wiki` · `#gov/hooks` · `#agent`

### `#gov/hooks`（与实现 EW13 对齐）

- **已实现（W1）**：SpacePolicy `bodyJson.hooks`（`ash.hooks.v1`）· Run **PreToolUse** 求值 · 审计事件 `hook.pre_tool_use` / `hook.decision`（Agent **Trajectory** + **Details** KV）· Reviews **编排流程** 只读规则表。
- **原型扩展（非目标）**：Hook 生命周期条（SessionStart / PostToolUse 等）为交互占位；W1 不落地外部 shell Hooks / Cordis。
- 字段与 payload 说明：[`doc/appendices/ash-hooks-v1.md`](../../appendices/ash-hooks-v1.md)。

## 三主题

| 主题 | 内容 |
|------|------|
| **Agent** | 编程/通用切换 · ASH 空态水印 · Cursor 式 Composer（审批模式 + 模型 + 发送/暂停合一）· 记忆深链 |
| **记忆** | 下文全页（本修订补全） |
| **管控&评审** | 原吸收 1–7 + v5 Workbench 入口 |

## 记忆 · 记忆体

| 子页 | 对应能力 | 说明 |
|------|----------|------|
| 分层浏览 | L0/L1/L2 + 候选列表 + 治理边 | 主轴；薄通过/拒绝；厚评审深链 |
| 场景视角 | perspective=scenario | 按 Scenario 聚合（字段待回填提示） |
| Skill 视角 | perspective=skill | Skill 绑定记忆 |
| Tools 视角 | perspective=tools | 工具引用记忆 |
| 项目视角 | perspective=project | repo / 项目 scope |
| 新建候选 | createCandidate 表单 | 层级 · 证据 · runId |
| 检索已批准 | queryMemory | topK · hit_used |
| TTL 复核 | TTL queue + sweep | 到期弃用 / 复核 |
| MemoryLink | 线程关联 · seal/replay | 服务评审三性 |
| 资产登记 | MemoryAsset Registry | 启停厚操作在管控台 |

## 记忆 · 知识

| 子页 | 对应能力 |
|------|----------|
| 总览 | RAG Hybrid Profile · 重建索引 · Repo Profile |
| Wiki 投影 | 列表 + 正文 + contextRef |
| LSP 探针 | hover / definition / references |

## 原则

- 记忆页做 **薄批准**；多签 / Workbench **深链管控&评审**。  
- 知识并入记忆 Tab，不占顶栏。  
- 不引入 Cordis；不在 Agent 堆叠记忆治理 UI。

## 修订

| 日期 | 说明 |
|------|------|
| 2026-09-17 | 初稿 8 屏平铺 → 收敛 Agent / 管控&评审 |
| 2026-09-17 | **补全记忆全页**：记忆体 10 子页 + 知识 3 子页 |
| 2026-09-17 | Agent 会话头右上角 **记忆** → 当前会话 MemoryLink（`#memory/links?from=agent`） |
| 2026-09-17 | Agent Composer 对标 Cursor：胶囊输入、+ 菜单、**发送/暂停合一** |
| 2026-09-17 | 顶栏：账号登录态 + 个人/团队空间切换；**更多 → 设置**（聚合运维/合规入口） |
| 2026-09-17 | Agent 看板：编程/通用切换 · ASH 空态水印 · 审批模式（询问/自动/完全）· 模型切换 |
| 2026-09-18 | 品牌图：`assets/ash-icon.png`（图标）· `assets/ash-chat-bg.png`（Chat 空态背景） |
| 2026-09-18 | Chat 背景重绘：保留代码雨；强化面部/头部；身躯简化为淡轮廓 |
| 2026-09-19 | `#gov/hooks` 与 EW13 实现对齐说明（PreToolUse 审计 · Trajectory/Reviews 薄投影） |
