# Sprint VX46 — v6.4 范围冻结 + 签字 / Freeze and sign-off

> **前置 / Prerequisite：** VX41–VX45；[`v6.4-release-scope.md`](v6.4-release-scope.md)  
> **状态 / Status：** ⬜  
> **做法 / Approach：** 对齐 VX36 — 冻结范围 + `make v6.4-signoff`（不自动打 tag）。  

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| VX46-0 | 范围标「已冻结」+ 清单/签字模板 | ⬜ |
| VX46-1 | `scope-freeze-gate` 含 v6.4 | ⬜ |
| VX46-2 | `scripts/v64-signoff-gate.sh` + Makefile | ⬜ |
| VX46-3 | `make v6.4-signoff` 绿 | ⬜ |

## 验收 / Verify

```bash
make scope-freeze-gate
make v6.4-signoff
```
