# Task EW13 Report — Hooks 审计事件 + 薄管控投影

**Status:** ✅ Complete  
**Date:** 2026-09-19

## Summary

EW12 已在 Run 侧发出 `hook.pre_tool_use` / `hook.decision`。EW13 稳定文档化 payload 字段，并在控制台最小可观测：Agent Trajectory + Details KV，Reviews 编排流程只读 hooks 规则表。

## Deliverables

| # | Item | Result |
|---|------|--------|
| EW13-1 | 事件类型与 payload 文档 | [`doc/appendices/ash-hooks-v1.md`](../../../doc/appendices/ash-hooks-v1.md) §审计事件 |
| EW13-2 | FE 薄投影 | `eventVisibility` · `conversationNodes` · `DetailsPane` · `ReviewsPage` orchestrate |
| EW13-3 | 原型 README | [`doc/prototypes/agent-absorb/README.md`](../../../doc/prototypes/agent-absorb/README.md) `#gov/hooks` |

## Commits

- `feat: Hooks 决策事件可观测`（见下方 git log）

## Tests

```text
npx vitest run \
  src/modules/agent-session/api/session.api.test.ts \
  src/modules/agent-session/components/conversationNodes.test.ts \
  src/modules/registry/hooksPolicy.test.ts
# 3 files, 27 tests passed
```

Go backend unchanged（EW12 已覆盖事件发射单测）。

## Concerns / Follow-ups

- **Reviews ThreadTimeline** 仍为 Interaction 折叠节点，不单独高亮 `hook.*`；Run 探查需 Agent Trajectory 或 raw 事件 API。
- **SSE**：`runStream` 已订阅 `hook.*` / `gate.waiting_approval`；session stream 依赖服务端转发同名 event type。
- **EW14+** 未启动（按 brief）。

## Paths (primary)

- `doc/appendices/ash-hooks-v1.md`
- `frontend/src/modules/agent-session/api/session.api.ts`
- `frontend/src/modules/agent-session/components/conversationNodes.ts`
- `frontend/src/modules/agent-session/components/DetailsPane.tsx`
- `frontend/src/modules/registry/hooksPolicy.ts`
- `frontend/src/pages/ReviewsPage.tsx`
- `doc/plan/sprint-ew-w1-v60-absorb.md`
- `CHANGELOG.md`
