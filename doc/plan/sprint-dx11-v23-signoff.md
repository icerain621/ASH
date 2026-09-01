# Sprint DX11：v2.3 范围冻结 + Hybrid 签字门禁

> **方案：** 已批准 **B** = 冻结 `v2.3-release-scope` + `make v2.3-signoff`（含 DX9 Minor 收口）  
> **Goal:** 对齐 DX8；签字门禁含 `rag-hybrid-smoke`；**不自动**打 `v2.3.0`  
> **状态：** 本批完成 · **无新表**（000028 补 digest 索引）

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX11-A | DX9 Minor：digest 索引、prefer=path、lane 错误传播、rebuild 200 测 | ✅ |
| DX11-B | `v2.3-release-scope` 冻结 + scope-freeze 纳入 | ✅ |
| DX11-C | `make v2.3-signoff` + 清单/签字模板 | ✅ |

## 退出标准

- [x] `v2.3-release-scope` 含 §1–§6 与「已冻结」
- [x] `make scope-freeze-gate` 含 v2.3
- [x] `make v2.3-signoff` 可跑（tag 人工）
