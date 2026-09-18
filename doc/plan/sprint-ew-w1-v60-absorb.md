# Sprint EW · W1 v6.0 吸收任务板

> 波次：**W1 · v6.0 吸收**（[`ash-feature-inventory.md`](ash-feature-inventory.md)）· ~376 人时  
> 依据：四象限 P0 = P13 Hooks · P11 ExecPolicy · P07 Steer/Queue  
> 实现计划：[`docs/superpowers/plans/2026-09-19-w1-v60-absorb.md`](../../docs/superpowers/plans/2026-09-19-w1-v60-absorb.md)  
> 前置：**W0 ✅**（[`sprint-ew-w0-harden.md`](sprint-ew-w0-harden.md)）  
> 原则：fail-closed；不引入 Cordis；不做会话树/Compact UX（W2）；中文 commit

| # | Sprint | 人时 | 依赖 | 状态 |
|---|--------|------|------|------|
| **EW11** | Hooks 协议骨架 `ash.hooks.v1` | 40 | — | ✅ |
| **EW12** | PreToolUse 接入 Run 工具链 | 48 | EW11 | ✅ |
| **EW13** | Hooks 审计事件 + 最小管控投影 | 32 | EW12 | ✅ |
| **EW14** | ExecPolicy 声明 schema + 合并解析 | 48 | — | ✅ |
| **EW15** | ExecPolicy → sandbox 地板 + Doctor 位 | 40 | EW14 | ✅ |
| **EW16** | Steer 意图（运行中打断并续写） | 48 | — | ✅ |
| **EW17** | Queue 意图 + Chat 投影 | 48 | EW16 | ✅ |
| **EW18** | W1 签字：openapi · 测 · CHANGELOG · 板勾选 | 16 | EW13+EW15+EW17 | ⬜ |

**建议顺序：** EW11→EW12→EW13 ∥ EW14→EW15 ∥ EW16→EW17 → EW18  
**首切片：** EW11（今日开工）

---

## EW11 — Hooks 协议骨架（40h）

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| EW11-1 | 包 `internal/hooks`：`ash.hooks.v1` 文档类型（事件枚举、Decision、Rule） | 单测 | ✅ |
| EW11-2 | 从 SpacePolicy `BodyJSON.hooks` 或默认空配置加载 | GetEffective；缺省 = 无 hook（不阻断） | ✅ |
| EW11-3 | `Evaluate(PreToolUse, ctx)`：声明式匹配 tool/risk → allow\|deny\|ask | fail-closed：匹配 deny 则拒绝；匹配但 action 无效 → deny；未知 event 等跳过规则 | ✅ |
| EW11-4 | OpenAPI/附录短说明 + CHANGELOG 条目 | [`ash-hooks-v1.md`](../appendices/ash-hooks-v1.md) | ✅ |

**非目标：** 外部 shell 脚本 Hooks；SessionStart 全生命周期（EW12+ 再扩）。

---

## EW12 — PreToolUse 接入（48h）

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| EW12-1 | `runs/execute` 在 `callTool` 前调用 hooks.Evaluate | deny → 跳过工具并发 `hook.pre_tool_use` 事件 | ✅ |
| EW12-2 | ask → 走现有 waiting_approval（复用 EW04） | 单测 | ✅ |
| EW12-3 | 空间无 hooks 配置时行为与今日一致 | 回归测 | ✅ |

---

## EW13 — 审计 + 管控投影（32h）

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| EW13-1 | 事件类型稳定：`hook.pre_tool_use` / `hook.decision` | Trajectory 可见 | ✅ |
| EW13-2 | Reviews/管控侧只读列表或详情 KV（薄） | FE 最小 | ✅ |
| EW13-3 | 原型 `#gov/hooks` 能力说明对齐 README | 文档 | ✅ |

---

## EW14 — ExecPolicy schema（48h）

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| EW14-1 | `execpolicy.v1`：network/fs/process 声明字段（JSON） | 解析单测 | ✅ |
| EW14-2 | 挂 SpacePolicy BodyJSON 与/或 Harness Profile | get/put | ✅ |
| EW14-3 | 合并规则：更严者优先（V6-D3） | 单测 | ✅ |

---

## EW15 — ExecPolicy 执行地板 + Doctor（40h）

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| EW15-1 | 合并结果喂入 `sandbox.ResolveSandboxModeExt` 地板 | 单测 | ✅ |
| EW15-2 | Doctor 卡片：execpolicy 已加载 / 能力位 | 探针或 readiness 字段 | ✅ |
| EW15-3 | 无配置不降低现有隔离 | 回归 | ✅ |

---

## EW16 — Steer（48h）

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| EW16-1 | intent `steer`：运行中 cancel 当前 turn/step + 注入新 prompt | 协议 + 单测 | ✅ |
| EW16-2 | 事件 `session.steer` | 可审计 | ✅ |
| EW16-3 | Chat：运行中 Composer 语义为 Steer（文案） | FE | ✅ |

---

## EW17 — Queue（48h）

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| EW17-1 | intent `queue`：运行中入队；空闲视为 prompt；结束后自动消费一条 | meta queue + 单测 | ✅ |
| EW17-2 | Chat 队列 chip + Stop 仅停当前（不清空队列） | FE | ✅ |
| EW17-3 | 与 steer 互斥语义写进注释/OpenAPI | 文档 | ✅ |

---

## EW18 — W1 签字（16h）

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| EW18-1 | `make openapi-check` · 相关 go/vitest · web-build | 绿 | ⬜ |
| EW18-2 | CHANGELOG · TODO · inventory W1 状态 | 更新 | ⬜ |
| EW18-3 | 本板全 ✅ | — | ⬜ |

## 明确不做（W1）

- Cordis / 会话树 fork / Compact 对话 UX（→ W2）
- Hermes 全 IM；YOLO 默认
- 外部任意脚本 Hooks 无审计执行

## 修订

| 日期 | 说明 |
|------|------|
| 2026-09-19 | 初版 EW11–EW18 |
