# Sprint GV06 — v5.0 范围冻结 + 签字 / Freeze and sign-off

> **前置 / Prerequisite：** v4.x 已收口；[`v5.0-release-scope.md`](v5.0-release-scope.md)  
> **状态 / Status：** ✅  
> **做法 / Approach：** 对齐 DX84 — 冻结 `v5.0-release-scope` + `make v5-signoff`（不自动打 tag）。  

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| GV06-0 | 起草 `v5.0-release-scope` | ✅ |
| GV06-1 | 冻结范围 + 签字模板 | ✅ |
| GV06-2 | `scope-freeze-gate` 含 v5.0 | ✅ |
| GV06-3 | `make v5-signoff` 绿作签字前置 | ✅ |

## 验收 / Verify

```bash
make scope-freeze-gate
make v5-signoff
```

- tag `v5.0.0` **人工**，门禁不自动打标。 / Tag is manual.
