# Sprint EW · W0 硬化任务板

> 波次：**W0 · 硬化**（[`ash-feature-inventory.md`](ash-feature-inventory.md)）· ~160 人时 · ~3 周 / 1 人  
> 实现计划：[`docs/superpowers/plans/2026-09-18-w0-harden.md`](../../docs/superpowers/plans/2026-09-18-w0-harden.md)  
> 原型对照：[`doc/prototypes/agent-absorb`](../prototypes/agent-absorb/)  
> 原则：不改产品边界；不做 Hooks / Steer / 会话树（属 W1+）；禁 YOLO；中文 commit

| # | Sprint | 人时 | 依赖 | 状态 |
|---|--------|------|------|------|
| **EW01** | OpenAPI 漂移清理 | 16 | — | ✅ 完成 |
| **EW02** | permissionMode ↔ 询问/自动/完全 UX | 24 | — | ✅ 完成 |
| **EW03** | Agent 空态品牌 + 编程/通用模式 | 24 | EW02 可选并行 | ⬜ |
| **EW04** | 审批预设 once / 会话 / 写入策略 | 48 | EW02 | ⬜ |
| **EW05** | MCP 注册工具 HTTP 执行 + 门禁 | 48 | EW04 建议先合（门禁复用） | ⬜ |

**合计：** 160h · 建议顺序 EW01∥EW02 → EW03 → EW04 → EW05

---

## EW01 — OpenAPI 漂移清理（16h）

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| EW01-1 | 删除或标注废弃 8 条 legacy `/v1/*`（tasks/agent-runs/memories…） | yaml 无未实现路径；`make openapi-check` | ✅ |
| EW01-2 | 补 `GET /api/v1/spaces` · `GET /api/v1/orgs` | 与 `handlers.go` 一致；swagger 同步 | ✅ |
| EW01-3 | 补 `POST /api/v1/auth/dev-login`（或文档声明仅非 prod） | openapi + handler 注释一致 | ✅ |
| EW01-4 | `make swagger` · `make openapi-check` · CHANGELOG | 绿 | ✅ |

**非目标：** 实现 legacy Tasks API。

---

## EW02 — 审批席位文案与 fail-closed（24h）

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| EW02-1 | ChatSeats：中文标签「询问审批 / 自动审批 / 完全访问」映射 `read-only` / `workspace-write` / `full` | vitest ChatSeats；默认仍 `read-only` | ✅ |
| EW02-2 | 完全访问二次确认文案对齐原型（非默认、风险明示） | 现有 confirm 强化；测试保留 | ✅ |
| EW02-3 | Composer/状态条展示当前审批模式（可选短文案） | AgentChatShell 可见 | ✅ |
| EW02-4 | Doc：permissionMode ↔ 原型 ask/auto/full 对照表 | inventory 或本板备注 | ✅ |

**permissionMode ↔ 原型对照（PATCH/JSON 仍用左列 enum）：**

| API `permissionMode` | 原型 `data-approve` | UI 席位标签 | Composer 短文案 |
|----------------------|---------------------|-------------|-----------------|
| `read-only`（默认） | `ask` | 询问审批 | 审批：询问 |
| `workspace-write` | `auto` | 自动审批 | 审批：自动 |
| `full` | `full` | 完全访问 | 审批：完全访问 |

**非目标：** 改 enum 值为 ask/auto（保持后端契约）；Hooks。

---

## EW03 — 空态品牌 + 编程/通用（24h）

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| EW03-1 | 拷贝 `ash-chat-bg.png` / `ash-icon.png` → `frontend/public` 或模块 assets | 构建可引用 | ⬜ |
| EW03-2 | 无会话 / 空白会话：ASH 空态水印（对标原型） | `agent-chat-empty` 有背景；可关 | ⬜ |
| EW03-3 | session meta `agentMode`: `coding` \| `general`（PATCH） | API + 默认 coding | ⬜ |
| EW03-4 | 左栏或顶栏模式切换；空态 tagline 随模式变 | FE + vitest | ⬜ |
| EW03-5 | `make web-build` + session 单测 | 绿 | ⬜ |

**非目标：** 语音输入；任务板新产品面。

---

## EW04 — 审批预设 once / 会话 / SpacePolicy（48h）

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| EW04-1 | 协议：gate 响应动作 `allow_once` · `allow_session` · `deny` · `cancel_run`（intent / approve 扩展） | 单测；事件可审计 | ⬜ |
| EW04-2 | 会话级允许集合写入 session meta（工具名/风险档） | 同会话二次同类工具不重复 gate（策略内） | ⬜ |
| EW04-3 | SpacePolicyPack `BodyJSON` 增加 `toolApprovalPresets`（或并列字段） | get/put + Reviews/Space 可编辑最小 UI | ⬜ |
| EW04-4 | IntentBar / gate：四按钮对齐原型（允许一次 / 本会话允许此类 / 拒绝 / 取消 Run） | FE + deriveGate | ⬜ |
| EW04-5 | Doctor 或 API 测：fail-closed 默认；无 YOLO | 证据或单测 | ⬜ |

**非目标：** ash.hooks.v1；execpolicy 完整沙箱声明（W1）。

---

## EW05 — MCP HTTP 执行 + 门禁（48h）

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| EW05-1 | `POST /api/v1/mcp/tools/{toolId}/execute`：查库 → `toolbus` `mcp.call` | OpenAPI + handler 测 | ⬜ |
| EW05-2 | 执行走风险档：medium+ 需 approval / session preset（复用 EW04） | 未批拒绝；审计事件 | ⬜ |
| EW05-3 | FE：McpToolsPanel「试执行」或 Chat 内调用路径 | 最小可用；非 YOLO | ⬜ |
| EW05-4 | slash/`mcp.call` 与 HTTP 路径共用校验（URL 白名单等） | 不重复漏洞面 | ⬜ |
| EW05-5 | `make openapi-check` · 相关 go test · CHANGELOG | 绿 | ⬜ |

**非目标：** 任意远程 MCP 无登记执行；Cordis。

---

## 验收总门

- [ ] `make openapi-check`
- [ ] `go test ./internal/session/... ./internal/api/... ./internal/toolbus/... ./internal/spacepolicy/... ./internal/runs/... -count=1`
- [ ] frontend：`npm test`（agent-session / platform MCP 相关）+ `make web-build`
- [ ] CHANGELOG + 本板全 ✅
- [ ] 不引入 Hooks / Steer / 会话树（留给 W1）

## 修订

| 日期 | 说明 |
|------|------|
| 2026-09-18 | 初版：W0 → EW01–EW05 |
