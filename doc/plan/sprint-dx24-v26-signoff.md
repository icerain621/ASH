# Sprint DX24：v2.6 范围冻结 + 签字门禁

> **方案：** 对齐 DX19 — 冻结 `v2.6-release-scope` + `make v2.6-signoff`  
> **Goal:** Doctor ALL **57** / M4 **10**；门禁含 rag-hybrid + sandbox + skill-pack + region；**不自动**打 `v2.6.0`  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX24-1 | `v2.6-release-scope` 冻结（§1–§6 + 已冻结） | ✅ |
| DX24-2 | `make v2.6-signoff` + scope-freeze 含 v2.6 | ✅ |
| DX24-3 | 清单 / 签字模板 | ✅ |
| DX24-4 | sprint / TODO / CHANGELOG / PLAN / smoke-index 水位 | ✅ |

## 行为

- `v2.6-signoff`：scope-freeze + openapi + Doctor ALL/M4 + rag-hybrid + sandbox（默认 `ASH_SKIP_SANDBOX=1`）+ skill-pack + region 探针
- Cancel / 云 live / 真人 tag 不变（人工）

## 退出标准

- [x] `v2.6-release-scope` 含 §1–§6 与「已冻结」
- [x] `make scope-freeze-gate` 含 v2.6
- [x] `make v2.6-signoff` 可跑（tag 人工）
