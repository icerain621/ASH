# Sprint DX84 — v4.3 范围冻结 + 签字 / Freeze and sign-off

> **方案 / Approach：** 对齐 DX78 — 冻结 `v4.3-release-scope` + `make v4.3-signoff`  
> **状态 / Status：** ✅  
> **范围 / Scope：** [`v4.3-release-scope.md`](v4.3-release-scope.md)

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| DX84-1 | 冻结 v4.3-release-scope | ✅ |
| DX84-2 | `make v4.3-signoff` + scope-freeze 含 v4.3 | ✅ |
| DX84-3 | 清单 / 签字模板 / TODO / CHANGELOG | ✅ |

## 验收 / Verify

```bash
make scope-freeze-gate
make v4.3-signoff
```

- tag `v4.3.0` **人工**，门禁不自动打标。 / Tag is manual.
