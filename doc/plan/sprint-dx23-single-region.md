# Sprint DX23：单区域就绪（v2.6）

> **方案：** 已批准 **B4** = `ASH_REGION` + `/readyz`/`scale` 字段 + HA/备份 runbook；Active-Active Out  
> **Goal：** 区域身份可观测；文档明示单区域；**无新表**  
> **状态：** ✅ 完成（代码）

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX23-1 | `config.Region()` + HealthResponse.region | ✅ |
| DX23-2 | Scale readiness `region` + ops parity | ✅ |
| DX23-3 | Scale UI 展示 + OpenAPI | ✅ |
| DX23-4 | `single-region-ha` 清单 | ✅ |
| DX23-5 | sprint / CHANGELOG / TODO | ✅ |

## 约定

- `ASH_REGION` 空/unset → `default`
- 不引入拓扑、复制或 Doctor 新测例（探针字段 + 文档即可）

## 退出标准

- [x] `TestReadyzRegionFromEnv` / `TestRegionDefaultAndOverride`
- [x] OpenAPI `HealthResponse.region` / `ScaleReadinessResponse.region`
- [x] 清单 [`../checklists/single-region-ha.md`](../checklists/single-region-ha.md)
