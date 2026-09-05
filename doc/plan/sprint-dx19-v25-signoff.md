# Sprint DX19：v2.5 范围冻结 + 签字门禁

> **方案：** 对齐 DX14/DX11 — 冻结 `v2.5-release-scope` + `make v2.5-signoff`  
> **Goal:** Doctor ALL **56** / M4 **9**；门禁含 waker + rag-hybrid + rag-vector + sandbox；**不自动**打 `v2.5.0`  
> **状态：** 本批完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX19-1 | `v2.5-release-scope` 冻结（§1–§6 + 已冻结） | ✅ |
| DX19-2 | `make v2.5-signoff` + scope-freeze 含 v2.5 | ✅ |
| DX19-3 | 清单 / 签字模板 | ✅ |
| DX19-4 | sprint / TODO / CHANGELOG / PLAN / smoke-index 水位 | ✅ |

## 行为

- `v2.5-signoff`：scope-freeze + openapi + Doctor ALL/M4 + waker + rag-hybrid + rag-vector + sandbox（默认 `ASH_SKIP_SANDBOX=1` 跳过 Docker）
- Cancel / 云 live / 真人 tag 不变（人工）

## 退出标准

- [x] `v2.5-release-scope` 含 §1–§6 与「已冻结」
- [x] `make scope-freeze-gate` 含 v2.5
- [x] `make v2.5-signoff` 可跑（tag 人工）
