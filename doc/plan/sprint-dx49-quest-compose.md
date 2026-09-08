# Sprint DX49 — Quest 原生 Goal/Plan 撰写+审批（v3.1 · T2）

> **方案：** 选项 1 — `/ui/quest` 内联 Goal→Plan→Approve；Runs quest-pane 保留  
> **Goal:** 用户不必跳到 Runs 即可完成委派；复用现有 API；**无新表**  
> **状态：** ✅ 完成 · **无新表**  
> **设计：** [`../../docs/superpowers/specs/2026-09-08-v31-dx49-quest-compose-design.md`](../../docs/superpowers/specs/2026-09-08-v31-dx49-quest-compose-design.md)  
> **计划：** [`../../docs/superpowers/plans/2026-09-08-dx49-quest-compose.md`](../../docs/superpowers/plans/2026-09-08-dx49-quest-compose.md)

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX49-1 | `getGoalPlan` 客户端 | ✅ |
| DX49-2 | QuestPage 测试（compose / approve / board plan） | ✅ |
| DX49-3 | QuestPage compose UI + board Plan 动作 | ✅ |
| DX49-4 | sprint / TODO / CHANGELOG / v3.1 scope | ✅ |

## 验收

```bash
cd frontend && npm test -- --run src/pages/QuestPage.test.tsx
```
