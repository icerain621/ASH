# Sprint DX54：v3.1 范围冻结 + 签字门禁

> **方案：** 对齐 DX48 — 冻结 `v3.1-release-scope` + `make v3.1-signoff`  
> **Goal:** Doctor ALL **57** / M4 **10**；Quest 回归 + v3.0 基线 smoke；**不自动**打 `v3.1.0`  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX54-1 | `v3.1-release-scope` 冻结（§1–§6 + 已冻结） | ✅ |
| DX54-2 | `make v3.1-signoff` + scope-freeze 含 v3.1 | ✅ |
| DX54-3 | 清单 / 签字模板 | ✅ |
| DX54-4 | sprint / TODO / CHANGELOG / PLAN / smoke-index 水位 | ✅ |

## 行为

- `v3.1-signoff`：scope-freeze + openapi + Doctor ALL/M4 + Quest（webhook / ListAfterSpace / diffreview / FE）+ rag-* + sandbox + skill-pack + rag-lsp + remote-sandbox-smoke
- Tag 仍人工

## 退出标准

- [x] `v3.1-release-scope` 含 §1–§6 与「已冻结」
- [x] `make scope-freeze-gate` 含 v3.1
- [x] `make v3.1-signoff` 可跑（tag 人工）
