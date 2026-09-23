# Sprint VX52 — v6.5 范围冻结 + 签字 / Freeze and sign-off

> **前置 / Prerequisite：** VX51；[`v6.5-release-scope.md`](v6.5-release-scope.md)  
> **状态 / Status：** ⬜  
> **做法 / Approach：** 对齐 VX46 — 冻结范围 + `make v6.5-signoff`（不自动打 tag）。  

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| VX52-0 | 范围标「已冻结」+ 清单/签字模板 | ⬜ |
| VX52-1 | `scope-freeze-gate` 含 v6.5 | ⬜ |
| VX52-2 | `scripts/v65-signoff-gate.sh` + Makefile | ⬜ |
| VX52-3 | `make v6.5-signoff` 绿 | ⬜ |

## 验收 / Verify

```bash
make scope-freeze-gate
make v6.5-signoff
```
