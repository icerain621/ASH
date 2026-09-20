# Sprint DX83 — 多区域探测清单 / Multi-region probe checklist

> **前置 / Prerequisite：** DX82；[`v4.3-release-scope.md`](v4.3-release-scope.md)  
> **状态 / Status：** ✅  
> **做法 / Approach：** 双语更新 [`../checklists/single-region-ha.md`](../checklists/single-region-ha.md)：如何读 `region`，并写明 `multiRegion=disabled`（非 Active-Active）。无代码变更。  
> Bilingual update to the single-region checklist: how to read `region`, and that `multiRegion=disabled` means not Active-Active. No code change.

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| DX83-1 | 清单双语：`region` 解读 | ✅ |
| DX83-2 | 写明 `multiRegion=disabled` 与 Out | ✅ |
| DX83-3 | v4.3 签字勾选段 | ✅ |

## 验收 / Verify

人工阅读清单；本机探针：

```bash
curl -s http://127.0.0.1:8080/readyz | jq '{region, multiRegion}'
```
