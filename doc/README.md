# ASH 文档归属索引

> 更新：2026-08-08  
> 代码水位：tag **`v0.1.0-mvp`** · Doctor ALL **43/43** · SQL rev **20** · RLS **41**

文档按**归属**分三类；契约与门禁路径保持稳定，避免打断脚本。

```txt
doc/
  design/       # 设计归属：需求 / 总体 / 架构 / 演进
  plan/         # 计划归属：排期 / 待办 / 范围 / 风险 / KPI
  progress/     # 进度归属：发布清单（证据见 evidence/，勾选见 checklists/）
  appendices/   # 设计规范资产（事件/DSL/Memory/…）— 路径稳定
  api/          # OpenAPI 契约 — 路径稳定
  checklists/   # 验收操作清单 — 路径稳定（脚本依赖）
  evidence/     # 门禁证据 — 路径稳定（脚本依赖）
  archive/      # 历史草案与废弃 DDL
```

## 1. 设计归属 · `design/`

| 文档 | 职责 | Owner 角色 |
|------|------|------------|
| [`PRD-需求文档.md`](design/PRD-需求文档.md) | 做什么、In/Out、成功指标 | 产品 |
| [`ORG-组织样板与商业落地.md`](design/ORG-组织样板与商业落地.md) | 付费/决策/审批与三套组织样板（PRD §3） | 产品 |
| [`HLD-总体设计.md`](design/HLD-总体设计.md) | 系统拆分、数据与安全概览 | 架构/后端 |
| [`ARCH-架构与技术选型.md`](design/ARCH-架构与技术选型.md) | 为何这样选、可替换项 | 平台/架构 |
| [`M3-多租户与Postgres演进.md`](design/M3-多租户与Postgres演进.md) | 多租户 / Postgres / RLS | 后端/运维 |
| [`appendices/`](appendices/README.md) | 协议、Schema、Doctor、Artifacts | 对应域负责人 |

## 2. 计划归属 · `plan/`

| 文档 | 职责 | Owner 角色 |
|------|------|------------|
| [`PLAN-进度与里程碑.md`](plan/PLAN-进度与里程碑.md) | **排期与完成度真相源** | 项目经理/技术 |
| [`TODO.md`](plan/TODO.md) | 未完成短清单（P0–P3） | 全员更新 |
| [`mvp-release-scope.md`](plan/mvp-release-scope.md) | MVP 范围冻结 | 产品 |
| [`risk-register.md`](plan/risk-register.md) | 风险台账 | 项目经理 |
| [`kpi-dashboard-definition.md`](plan/kpi-dashboard-definition.md) | KPI 口径 | 产品/后端 |

## 3. 进度归属 · `progress/` + 稳定运维路径

| 文档 / 目录 | 职责 | Owner 角色 |
|-------------|------|------------|
| [`mvp-release-checklist.md`](progress/mvp-release-checklist.md) | MVP 发布勾选总表 | 发布 |
| [`checklists/`](checklists/smoke-index.md) | 烟测 / RDS / 签字 runbook | 发布/运维 |
| [`evidence/`](evidence/README.md) | 门禁产出证据（自动写入） | 发布门禁 |

日常进度叙事写在 `plan/PLAN` 与 `plan/TODO`；勾选与证据落在 `progress/` + `checklists/` + `evidence/`。

## 4. 契约（跨归属，路径不变）

| 路径 | 说明 |
|------|------|
| [`api/openapi-ash-v1.yaml`](api/openapi-ash-v1.yaml) | 手写 OpenAPI（`make openapi-check`） |
| [`api/openapi-alignment.md`](api/openapi-alignment.md) | 与 swag 对齐 |
| [`api/error-codes.md`](api/error-codes.md) | HTTP 错误码 |

## 5. 归档 · `archive/`

| 路径 | 内容 |
|------|------|
| [`archive/product-mvp-draft/`](archive/README.md) | 早期 tasks/MySQL/Jira 草案（勿作实现依据） |
| [`archive/legacy-db/`](archive/legacy-db/ash_mvp_schema.sql) | 历史 MySQL DDL（现行 schema 在 `internal/store/sqlmigrations`） |

## 6. 维护约定

1. **计划/进度只改** `plan/PLAN` + `plan/TODO`；变更事实进 `CHANGELOG.md`。  
2. **设计改契约**：先 `appendices/` 或 `api/openapi-ash-v1.yaml`，再改代码，再 `make openapi-check`。  
3. **禁止**在 `archive/` 或平行编号文档另起 PRD/计划。  
4. **脚本硬编码路径**仅允许：`doc/checklists/*`、`doc/evidence/*`、`doc/api/*`、`doc/plan/mvp-release-scope.md`、`doc/progress/mvp-release-checklist.md`。  
5. 各子目录 `README.md` 写明本归属的入口与禁止事项。
