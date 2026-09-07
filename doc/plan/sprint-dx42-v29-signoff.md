# Sprint DX42：v2.9 范围冻结 + 签字门禁

> **方案：** 对齐 DX36 — 冻结 `v2.9-release-scope` + `make v2.9-signoff`  
> **Goal:** Doctor ALL **57** / M4 **10**；门禁含 rag-* + sandbox + skill-pack + rag-lsp + remote-sandbox；**不自动**打 `v2.9.0`  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX42-1 | `v2.9-release-scope` 冻结（§1–§6 + 已冻结） | ✅ |
| DX42-2 | `make v2.9-signoff` + scope-freeze 含 v2.9 | ✅ |
| DX42-3 | 清单 / 签字模板 | ✅ |
| DX42-4 | sprint / TODO / CHANGELOG / PLAN / smoke-index 水位 | ✅ |

## 行为

- `v2.9-signoff`：scope-freeze + openapi + Doctor ALL/M4 + rag-hybrid + rag-vector + sandbox（默认 `ASH_SKIP_SANDBOX=1`）+ skill-pack + rag-lsp + remote-sandbox-smoke
- Tag 仍人工（D7）

## 退出标准

- [x] `v2.9-release-scope` 含 §1–§6 与「已冻结」
- [x] `make scope-freeze-gate` 含 v2.9
- [x] `make v2.9-signoff` 可跑（tag 人工）
