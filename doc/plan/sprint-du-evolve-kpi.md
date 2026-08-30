# Sprint DU：演进 KPI + 移动审阅（方案 C）Implementation Plan

> **方案：** 已批准 **C** = KPI-17/18 + Metrics 演进区 + 移动审阅页  
> **Goal：** 看板覆盖 KPI-17~19；手机可审 Plan/Diff 式队列项。  
> **状态：** ✅ 本批完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DU-1 | KPI-17 编排评审积压 + KPI-18 Improve 回滚率 | ✅ |
| DU-2 | Metrics 页演进区（17/18/19） | ✅ |
| DU-3 | `/ui/m/reviews` 移动审阅 | ✅ |
| DU-4 | 清单 / docs / 测试 | ✅ |

## 口径

- **KPI-17**：当前 space 内 `harness_profile_versions` + `scenario_patch_drafts` 且 `status=in_review` 的数量；目标 &lt; 50；描述含 &gt;7 天未清数
- **KPI-18**：`improve_proposals` 中 `rolled_back / (promoted + rolled_back)`；目标 &lt; 2%
- **KPI-19**：已有（DX2）
