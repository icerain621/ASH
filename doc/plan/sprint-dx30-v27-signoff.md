# Sprint DX30：v2.7 范围冻结 + 签字门禁

> **方案：** 对齐 DX24 — 冻结 `v2.7-release-scope` + `make v2.7-signoff`  
> **Goal:** Doctor ALL **57** / M4 **10**；门禁含 rag-hybrid + rag-vector + sandbox + skill-pack + thin LSP；**不自动**打 `v2.7.0`  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX30-1 | `v2.7-release-scope` 冻结（§1–§6 + 已冻结） | ✅ |
| DX30-2 | `make v2.7-signoff` + scope-freeze 含 v2.7 | ✅ |
| DX30-3 | 清单 / 签字模板 | ✅ |
| DX30-4 | sprint / TODO / CHANGELOG / PLAN / smoke-index 水位 | ✅ |

## 行为

- `v2.7-signoff`：scope-freeze + openapi + Doctor ALL/M4 + rag-hybrid + rag-vector + sandbox（默认 `ASH_SKIP_SANDBOX=1`）+ skill-pack + thin-LSP 包测
- Cancel / 云 live / 真人 tag 不变（人工）

## 退出标准

- [x] `v2.7-release-scope` 含 §1–§6 与「已冻结」
- [x] `make scope-freeze-gate` 含 v2.7
- [x] `make v2.7-signoff` 可跑（tag 人工）
