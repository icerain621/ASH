# Sprint DX48：v3.0 范围冻结 + 签字门禁

> **方案：** 对齐 DX42 — 冻结 `v3.0-release-scope` + `make v3.0-signoff`  
> **Goal:** Doctor ALL **57** / M4 **10**；门禁含 rag-* + sandbox + skill-pack + rag-lsp + remote-sandbox；**不自动**打 `v3.0.0`  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX48-1 | `v3.0-release-scope` 冻结（§1–§6 + 已冻结） | ✅ |
| DX48-2 | `make v3.0-signoff` + scope-freeze 含 v3.0 | ✅ |
| DX48-3 | 清单 / 签字模板 | ✅ |
| DX48-4 | sprint / TODO / CHANGELOG / PLAN / smoke-index 水位 | ✅ |

## 行为

- `v3.0-signoff`：scope-freeze + openapi + Doctor ALL/M4 + rag-hybrid + rag-vector + sandbox（默认 `ASH_SKIP_SANDBOX=1`）+ skill-pack + rag-lsp + remote-sandbox-smoke
- Tag 仍人工（E2 / 分冻后本代）

## 退出标准

- [x] `v3.0-release-scope` 含 §1–§6 与「已冻结」
- [x] `make scope-freeze-gate` 含 v3.0
- [x] `make v3.0-signoff` 可跑（tag 人工）
