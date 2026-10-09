# ASH 文档归属索引

> 更新：2026-10-08  
> 代码水位：**v6.7 已冻结**（`main` `7b474964`）。冻结后已合入推理等级 / P02 无障碍等（不升代）。tag 人工，尚未打 `v6.*`。  
> Watermark: **v6.7 frozen** at `7b474964`. Post-freeze thin work is on main (no new generation). Tags are manual. No `v6.*` tag yet.

人读文档按软件交付的标准类别归类。门禁路径不搬。  
Reader docs follow standard delivery classes. Gate paths stay put.

| 标准类别 | 现行入口 |
|----------|----------|
| 需求 | [`design/SRS-需求规格.md`](design/SRS-需求规格.md) |
| 设计 | [`design/SDD-总体设计.md`](design/SDD-总体设计.md)（吸收、前端原型、后端模块层级） |
| 计划 | [`plan/PLAN-进度与里程碑.md`](plan/PLAN-进度与里程碑.md) · [`plan/GA-上线推进计划.md`](plan/GA-上线推进计划.md) · [`plan/TODO.md`](plan/TODO.md) · `plan/*-release-scope.md` |
| 测试与验收 | [`checklists/`](checklists/smoke-index.md) |
| 发布证据 | [`evidence/`](evidence/README.md) · [`progress/`](progress/README.md) |
| 接口与协议 | [`api/openapi-ash-v1.yaml`](api/openapi-ash-v1.yaml) · [`appendices/`](appendices/README.md) |

已被总册取代的长文在 [`archive/narrative/`](archive/narrative/)。旧文件名只留指向。  
Long texts replaced by the classified set are in `archive/narrative/`. Old filenames are pointers only.

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
| [`SRS-需求规格.md`](design/SRS-需求规格.md) | 需求：场景、功能点、Out | 产品 |
| [`SDD-总体设计.md`](design/SDD-总体设计.md) | 建构演进、吸收、前端原型、后端模块层级 | 架构/后端 |
| [`appendices/`](appendices/README.md) | 协议、Schema、Doctor、Artifacts（路径稳定） | 对应域负责人 |
| [`archive/narrative/`](archive/narrative/) | 已归档的旧 PRD / HLD / ARCH（不作为现行依据） | — |

## 2. 计划归属 · `plan/`

| 文档 | 职责 | Owner 角色 |
|------|------|------------|
| [`PLAN-进度与里程碑.md`](plan/PLAN-进度与里程碑.md) | **计划与进度**（v6.7 已冻结 + 冻结后水位） | 项目经理/技术 |
| [`GA-上线推进计划.md`](plan/GA-上线推进计划.md) | 打包上线六步（签字、云库、切换日） | 项目经理/发布 |
| [`TODO.md`](plan/TODO.md) | 未完成短清单（P0–P3） | 全员更新 |
| [`v4.x-program.md`](plan/v4.x-program.md) | **v4.x** 四代分冻（Auth → 企业 Agentic → Stage-1 → 生态） | 项目经理/架构 |
| [`v4.0-release-scope.md`](plan/v4.0-release-scope.md) | v4.0 Auth 硬化范围（**已冻结**） | 产品/发布 |
| [`v5-governance-program.md`](plan/v5-governance-program.md) | **v5** 双核管控 × 厚评审 × 薄交互（设计评审中） | 项目经理/架构 |
| [`platform-quad-comparison.md`](plan/platform-quad-comparison.md) | 吸收决策长证据（结论在 SDD） | 架构/产品 |
| [`sprint-dh-harness-implementation.md`](plan/sprint-dh-harness-implementation.md) | Sprint DH Harness 骨架任务板 | 后端 |
| [`sprint-di-loop-implementation.md`](plan/sprint-di-loop-implementation.md) | Sprint DI Loop Adapter 任务板 | 后端 |
| [`sprint-dx-sandbox-implementation.md`](plan/sprint-dx-sandbox-implementation.md) | Sprint DX Sandbox POC 任务板 | 后端/安全 |
| [`sprint-dy-evolve-implementation.md`](plan/sprint-dy-evolve-implementation.md) | Sprint DY 演进平面任务板 | 后端 |
| [`mvp-release-scope.md`](plan/mvp-release-scope.md) | MVP 范围冻结 | 产品 |
| [`risk-register.md`](plan/risk-register.md) | 风险台账 | 项目经理 |
| [`kpi-dashboard-definition.md`](plan/kpi-dashboard-definition.md) | KPI 口径 | 产品/后端 |

## 3. 进度归属 · `progress/` + 稳定运维路径

| 文档 / 目录 | 职责 | Owner 角色 |
|-------------|------|------------|
| [`mvp-release-checklist.md`](progress/mvp-release-checklist.md) | MVP 发布勾选总表 | 发布 |
| [`checklists/`](checklists/smoke-index.md) | 烟测 / RDS / 签字 runbook（含 `v4.0-signoff`） | 发布/运维 |
| [`evidence/`](evidence/README.md) | 门禁产出证据（自动写入） | 发布门禁 |
| [`diagrams/archify/`](diagrams/archify/README.md) | 可交互架构 / 时序 / 场景图 | 架构 |

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
