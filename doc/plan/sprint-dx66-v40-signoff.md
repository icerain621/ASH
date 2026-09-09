# Sprint DX66 — v4.0 范围冻结 + 签字（A Auth）

> **方案：** 对齐 DX60 — 冻结 `v4.0-release-scope` + `make v4.0-signoff`  
> **状态：** ✅ 完成  
> **设计：** [`../../docs/superpowers/specs/2026-09-09-v40-dx66-signoff-design.md`](../../docs/superpowers/specs/2026-09-09-v40-dx66-signoff-design.md)

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX66-1 | 冻结 v4.0-release-scope | ✅ |
| DX66-2 | `make v4.0-signoff` + scope-freeze 含 v4.0 | ✅ |
| DX66-3 | 清单 / 签字模板 / TODO / CHANGELOG | ✅ |

## 验收

```bash
make scope-freeze-gate
make v4.0-signoff
```

- tag `v4.0.0` **人工**，门禁不自动打标
