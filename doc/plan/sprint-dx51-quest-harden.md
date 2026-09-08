# Sprint DX51 — Quest 工作台硬化（v3.1 · T2）

> **方案：** waiting_approval 门禁面板（Approve/Cancel）+ 产物列表；复用现有 API；**无新表**  
> **状态：** ✅ 完成  
> **设计：** [`../../docs/superpowers/specs/2026-09-08-v31-dx51-quest-harden-design.md`](../../docs/superpowers/specs/2026-09-08-v31-dx51-quest-harden-design.md)

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX51-1 | Quest 门禁面板 + Cancel | ✅ |
| DX51-2 | Quest 产物面板 + access | ✅ |
| DX51-3 | vitest + sprint / TODO / CHANGELOG | ✅ |

## 验收

```bash
cd frontend && npm test -- --run src/pages/QuestPage.test.tsx
# 10 tests passed
```

## 交付摘要

- `waiting_approval` 时展示 `quest-gate-panel`：timeline `gate.waiting_approval` 原因 + Approve / Cancel
- 选中 Run 后展示 `quest-artifacts-pane`：列表 + 签名链接（`getRunArtifactAccess`）
- 复用 `approveRun` / `cancelRun` / `getRunArtifacts`；**无新表 / 无新 API**
