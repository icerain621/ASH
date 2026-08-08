# 文档归档说明

> 归档日期：2026-08-08  
> 现行归属索引：[`../README.md`](../README.md)（`design/` · `plan/` · `progress/`）

## 为何归档

仓库曾并存两套文档体系；草案与现行实现冲突，已迁入本目录，**不再作为实现与排期依据**。

| 体系 | 特征 | 现状 |
|------|------|------|
| **现行归属** | `doc/design` + `doc/plan` + `doc/progress` + appendices/api | 与代码对齐 |
| **product-mvp-draft/** | `/v1/tasks`、MySQL、微服务、10 周人天、Jira CSV | 已取代 |
| **legacy-db/** | MySQL `ash_mvp_schema.sql` | 已废弃 |

## `product-mvp-draft/` → 现行替代

| 归档文件 | 现行替代 |
|----------|----------|
| 调研 / 模板 PRD / 产品需求设计 | [`../design/PRD-需求文档.md`](../design/PRD-需求文档.md) |
| API/事件/DB 草案 | [`../api/`](../api/openapi-ash-v1.yaml) + [`../appendices/`](../appendices/README.md) |
| 10 周计划 / Jira | [`../plan/PLAN-进度与里程碑.md`](../plan/PLAN-进度与里程碑.md) |
| MySQL/Redis 选型与 backend/ 结构 | [`../design/ARCH-架构与技术选型.md`](../design/ARCH-架构与技术选型.md) |
| 工程搭建 | 根 [`README.md`](../../README.md) |
| 里程碑评审模板 | [`../progress/mvp-release-checklist.md`](../progress/mvp-release-checklist.md) |
| 商业分析报告 | 归档保留；产品决策以 PRD 为准 |

## 引用规则

- 新文档只指向 `doc/design|plan|progress` 与稳定路径 `appendices|api|checklists|evidence`。  
- 勿在归档目录继续迭代需求或排期。
