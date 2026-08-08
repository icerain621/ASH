# ASH MVP v0.1 发布范围（冻结草案）

> 状态：**已冻结**（2026-07-07，产品 `产品负责人（占位）` 确认）（对应 [`../progress/mvp-release-checklist.md`](../progress/mvp-release-checklist.md) §1）  
> 归属：[`plan/`](README.md)  
> 版本锚点：Doctor ALL **43/43** · SQL rev **20** · RLS **41**

## 1. 发布目标

交付 ASH Worker + 控制台的 **M0–M3 + PRD MVP** 能力，支持 Feature 交付闭环、记忆治理、合规/规模化控制台，以及 Postgres 切换前的全量自动化门禁。

## 2. 包含范围（P0）

| 域 | 交付物 |
|----|--------|
| 核心闭环 | Runs/SSE/DSL/ToolBus/Artifacts/Checkpoint |
| 记忆 | 候选→评审→merge、TTL 队列/sweep、治理 derive |
| 合规 TR2 | Secret 扫描、审计导出、脱敏策略 |
| 规模化 TR3 | Scale/readyz、OpenAPI 契约、Provenance |
| M3 Postgres | SQL 迁移、RLS、本地 e2e 门禁 |
| PRD MVP | CI 诊断、KPI、反馈/告警、发布灰度记录 |
| 烟测 H-04~H-09 | 静态 + live 脚本（[`smoke-index.md`](../checklists/smoke-index.md)） |
| 发布门禁 | `make mvp-signoff` / `make web-gate` / `make queue-gate` |

## 3. 不包含范围（本版本）

- 真实生产发布执行（仅策略与证据记录）
- 多区域 Active-Active / 读写分离
- 组织级计费与外部 IdP 联邦（M2 基础权限已含）
- ExecGo/Codex 托管服务本身（仅集成与健康检查）

## 4. 变更冻结规则

- 冻结后仅接受 **P0 缺陷** 与 **文档/门禁** 修正
- 新表 DDL 必须同步 RLS + Doctor M3-11
- 破坏性 API 变更需 OpenAPI + `openapicheck` 同步

## 5. 验收门禁（自动化）

```bash
make mvp-signoff
make production-config-gate
make queue-gate
make t0-alert-gate
# 云切换前
cp config/cloud-rds.env.example config/cloud-rds.env
bash scripts/source-cloud-rds-env.sh
make cloud-acceptance
```

## 6. 评审签字

| 角色 | 姓名 | 日期 | 结论 |
|------|------|------|------|
| 产品 | 产品负责人（占位） | 2026-07-06 | 范围确认 / 冻结 |
| 技术 | 技术负责人（占位） | 2026-07-06 | 门禁达标 |
| 测试 | 测试负责人（占位） | 2026-07-06 | 回归通过 |
