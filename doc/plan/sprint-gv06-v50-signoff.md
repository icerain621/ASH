# Sprint GV06 — v5.0 范围冻结草案 / Freeze draft

> **前置 / Prerequisite：** v4.x 已收口；[`v5.0-release-scope.md`](v5.0-release-scope.md)  
> **状态 / Status：** 草案（本 Sprint 仅起草范围；冻结实现见后续「继续推进」）  
> Status: draft (scope only; freeze implementation follows on the next continue).  
> **做法 / Approach：** 对齐 DX84 — 先写范围，再接 `scope-freeze-gate` + 签字模板。  

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| GV06-0 | 起草 `v5.0-release-scope` | ✅ |
| GV06-1 | 冻结范围 + 签字模板 | ⬜ |
| GV06-2 | `scope-freeze-gate` 含 v5.0 | ⬜ |
| GV06-3 | `make v5-signoff` 绿作签字前置 | ⬜ |

## 验收 / Verify（冻结时）

```bash
make v5-signoff
make scope-freeze-gate
```
