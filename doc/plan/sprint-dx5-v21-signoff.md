# Sprint DX5：v2.1 范围冻结 + ACP Doctor（方案 C）

> **方案：** 已批准 **C** = 冻结 `v2.1-release-scope` + M4-ACP 探针 + `make v2.1-signoff`  
> **Goal:** Doctor ALL **55**（M4 **8**）；签字门禁对齐 DV；**不自动**打 `v2.1.0`  
> **状态：** 本批完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX5-1 | M4-ACP-01 schema + M4-ACP-02 Probe 未配置 | ✅ |
| DX5-2 | `v2.1-release-scope` 冻结结构 + scope-freeze 纳入 | ✅ |
| DX5-3 | `make v2.1-signoff` + 清单/签字模板 | ✅ |
| DX5-4 | TODO/PLAN/CHANGELOG 计数 55 | ✅ |

## 退出标准

- [x] `TestM4Suite` pass=8；`TestALLSuite` pass=55
- [x] `make scope-freeze-gate` 含 v2.1
- [x] `make v2.1-signoff` 可跑（tag 人工）
