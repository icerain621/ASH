# Sprint DX8：v2.2 范围冻结 + Waker 签字门禁（方案 C）

> **方案：** 已批准 **C** = merge PR #1 → 冻结 `v2.2-release-scope` + `make v2.2-signoff`  
> **Goal:** 对齐 DX5；签字门禁含 `waker-smoke`；**不自动**打 `v2.2.0`  
> **状态：** 本批完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX8-0 | merge DX3–DX7 → `main`（本地） | ✅ |
| DX8-1 | `v2.2-release-scope` 冻结结构 + scope-freeze 纳入 | ✅ |
| DX8-2 | `make v2.2-signoff` + 清单/签字模板 | ✅ |
| DX8-3 | TODO/PLAN/CHANGELOG | ✅ |

## 退出标准

- [x] `v2.2-release-scope` 含 §1–§6 与「已冻结」
- [x] `make scope-freeze-gate` 含 v2.2
- [x] `make v2.2-signoff` 可跑（tag 人工）
