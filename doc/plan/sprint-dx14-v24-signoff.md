# Sprint DX14：Waker 控制台 + v2.4 范围冻结（方案 C）

> **方案：** 已批准 **C** = Observability/Scale Waker 面板 + 冻结 `v2.4-release-scope` + `make v2.4-signoff`  
> **Goal:** 对齐 DX8/DX11；签字门禁含 `waker-smoke`；**不自动**打 `v2.4.0`  
> **状态：** 本批完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX14-1 | Waker API 客户端 + Observability 运维面板 | ✅ |
| DX14-2 | Scale 页 Waker 紧凑计数 | ✅ |
| DX14-3 | `v2.4-release-scope` 冻结 + `make v2.4-signoff` + 清单/模板 | ✅ |
| DX14-4 | sprint / TODO / CHANGELOG / PLAN / smoke-index 水位 | ✅ |

## 行为

- Observability：duties / queue / recent runs；dry-run sweep；cancel 三重闸门（`allowCancel` + confirm）
- Scale：enabled duties 数 + queue 长度 + ticker interval（无新 readiness 字段）
- Cancel UI 仅在 `/waker/status.allowCancel` 为真时展示

## 退出标准

- [x] `v2.4-release-scope` 含 §1–§6 与「已冻结」
- [x] `make scope-freeze-gate` 含 v2.4
- [x] `make v2.4-signoff` 可跑（tag 人工）
