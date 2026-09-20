# Sprint VX36 — v6.3 范围冻结 + 签字 / Freeze and sign-off

> **前置 / Prerequisite：** VX31–VX35；[`v6.3-release-scope.md`](v6.3-release-scope.md)  
> **状态 / Status：** ✅  
> **做法 / Approach：** 对齐 EW130 — 冻结范围 + `make v6.3-signoff`（不自动打 tag）。  

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| VX36-0 | 范围标「已冻结」+ 清单/签字模板 | ✅ |
| VX36-1 | `scope-freeze-gate` 含 v6.3 | ✅ |
| VX36-2 | `scripts/v63-signoff-gate.sh` + Makefile | ✅ |
| VX36-3 | `make v6.3-signoff` 绿 | ✅ |

## 验收 / Verify

```bash
make scope-freeze-gate
make v6.3-signoff
```

- tag `v6.3.0` **人工**。 / Tag is manual.
