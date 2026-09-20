# Sprint EW130 — v6.0 范围冻结 + 签字 / Freeze and sign-off

> **前置 / Prerequisite：** v5.0 已冻结；EW W0–W12 ✅；[`v6.0-release-scope.md`](v6.0-release-scope.md)  
> **状态 / Status：** ✅  
> **做法 / Approach：** 对齐 GV06 / DX84 — 冻结 `v6.0-release-scope` + `make v6-signoff`（不自动打 tag）。  

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| EW130-0 | 起草 `v6.0-release-scope` + `v6.x-program` | ✅ |
| EW130-1 | 冻结范围 + 签字模板 + 清单 | ✅ |
| EW130-2 | `scope-freeze-gate` 含 v6.0 | ✅ |
| EW130-3 | `make v6-signoff` 绿作签字前置 | ✅ |

## 验收 / Verify

```bash
make scope-freeze-gate
make v6-signoff
```

- tag `v6.0.0` **人工**，门禁不自动打标。 / Tag is manual.
