# Sprint DX60 — v3.2 范围冻结 + 签字（T3）

> **方案：** 对齐 DX54 — 冻结 `v3.2-release-scope` + `make v3.2-signoff`  
> **状态：** ✅ 完成  
> **设计：** [`../../docs/superpowers/specs/2026-09-09-v32-dx60-signoff-design.md`](../../docs/superpowers/specs/2026-09-09-v32-dx60-signoff-design.md)

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX60-1 | 冻结 v3.2-release-scope | ✅ |
| DX60-2 | `make v3.2-signoff` + scope-freeze 含 v3.2 | ✅ |
| DX60-3 | 清单 / 签字模板 / TODO / CHANGELOG | ✅ |

## 验收

```bash
make scope-freeze-gate
make v3.2-signoff
```

- tag `v3.2.0` **人工**，门禁不自动打标
