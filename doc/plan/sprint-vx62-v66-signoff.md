# Sprint VX62 — v6.6 范围冻结 + 签字 / Freeze and sign-off

> **前置 / Prerequisite：** VX61；[`v6.6-release-scope.md`](v6.6-release-scope.md)  
> **状态 / Status：** ⬜  
> **做法 / Approach：** 对齐 VX52 — 冻结范围 + `make v6.6-signoff`（不自动打 tag）。  

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| VX62-0 | 范围标「已冻结」+ 清单/签字模板 | ⬜ |
| VX62-1 | `scope-freeze-gate` 含 v6.6 | ⬜ |
| VX62-2 | `scripts/v66-signoff-gate.sh` + Makefile | ⬜ |
| VX62-3 | `make v6.6-signoff` 绿 | ⬜ |

## 验收 / Verify

```bash
make scope-freeze-gate
make v6.6-signoff
```
