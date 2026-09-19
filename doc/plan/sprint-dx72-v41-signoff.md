# Sprint DX72 — v4.1 范围冻结 + 签字

> **方案：** 对齐 DX66 — 冻结 `v4.1-release-scope` + `make v4.1-signoff`  
> **状态：** ⬜  
> **设计：** [`../../docs/superpowers/specs/2026-09-19-v41-enterprise-agentic-design.md`](../../docs/superpowers/specs/2026-09-19-v41-enterprise-agentic-design.md)

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX72-1 | 冻结 v4.1-release-scope（草案→已冻结） | ⬜ |
| DX72-2 | `make v4.1-signoff` + scope-freeze 含 v4.1 | ⬜ |
| DX72-3 | 清单 / 签字模板 / TODO / CHANGELOG | ⬜ |

## 验收

```bash
make scope-freeze-gate
make v4.1-signoff
```

- tag `v4.1.0` **人工**，门禁不自动打标
