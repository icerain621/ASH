# Sprint DX78 — v4.2 范围冻结 + 签字 / Freeze and sign-off

> **方案 / Approach：** 对齐 DX72 — 冻结 `v4.2-release-scope` + `make v4.2-signoff`  
> **状态 / Status：** ✅  
> **范围 / Scope：** [`v4.2-release-scope.md`](v4.2-release-scope.md)

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| DX78-1 | 冻结 v4.2-release-scope | ✅ |
| DX78-2 | `make v4.2-signoff` + scope-freeze 含 v4.2 | ✅ |
| DX78-3 | 清单 / 签字模板 / TODO / CHANGELOG | ✅ |

## 验收 / Verify

```bash
make scope-freeze-gate
make v4.2-signoff
```

- tag `v4.2.0` **人工**，门禁不自动打标。 / Tag is manual.
