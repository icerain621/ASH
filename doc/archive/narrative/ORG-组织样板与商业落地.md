# 组织样板与商业落地（PRD §3）

> 状态：v1.0（Sprint CO，2026-08-08）  
> 关闭 PRD §3 TODO：谁付费 / 谁决策 / 谁审批；2–3 个组织样板与价值指标。  
> 实现：`internal/orgtemplates` · `GET/POST /api/v1/org-templates…`

## 1. 部署形态

| 形态 | 适用 | 付费主体倾向 |
|------|------|----------------|
| **内部平台** | 企业研发效能中心统一采购 Worker/Postgres | 平台 / 工程效率预算 |
| **SaaS / 私有化** | 小团队自助开通；强合规多为私有化 | 团队预算或合规预算 |
| **either** | 小团队两种皆可 | 工程负责人 |

ASH v0.1 技术上已具备 Org/Space/RBAC/RLS；商业落地不阻塞 MVP 发布，本文件给出**可执行样板**而非计费系统。

## 2. 样板总览

| ID | 标签 | Space | 决策分离 |
|----|------|-------|----------|
| `small_team` | 小团队 / 创业交付组 | 1（Delivery） | 低：Tech Lead 兼任 |
| `mid_enterprise` | 中大型研发组织 | 2（Product + Platform） | 中：事业部决策 / Release 审批 |
| `strong_compliance` | 强合规 / 受监管 | 2（Controlled + Audit） | 高：CISO + 双人门禁 |

## 3. 样板明细

### 3.1 `small_team`

| 维度 | 约定 |
|------|------|
| 谁付费 | 工程负责人（团队预算） |
| 谁决策 | Tech Lead / 工程负责人 |
| 谁审批 | 同一 Tech Lead（场景 human step） |
| 角色 | admin、operator、reviewer |
| 推荐场景 | feature_delivery、hotfix |
| 价值指标 | KPI-01 成功率、KPI-02 时长、KPI-06 低分、KPI-11 场景稳定率 |

### 3.2 `mid_enterprise`

| 维度 | 约定 |
|------|------|
| 谁付费 | 工程效率 / 平台预算 Owner |
| 谁决策 | 事业部研发负责人 + 架构委员会 |
| 谁审批 | Release Manager（发版）+ Reviewer（合并） |
| 角色 | admin、operator、reviewer、release_manager、librarian |
| 推荐场景 | feature_delivery、hotfix、security_patch |
| 价值指标 | KPI-01/03/04/05/07/11（交付 + CI + 记忆采纳） |

### 3.3 `strong_compliance`

| 维度 | 约定 |
|------|------|
| 谁付费 | 合规 / 信息安全委员会预算 |
| 谁决策 | CISO / 合规官 + 业务 Owner |
| 谁审批 | Security approver + 双人发版门禁 |
| 角色 | admin、operator、reviewer、security、auditor |
| 推荐场景 | security_patch 优先，其次 hotfix / feature_delivery |
| 价值指标 | KPI-01/06/08/09/11（质量 + 稳定性 + 错误率） |

## 4. API / UI

```bash
# 列出样板
curl -s "$ASH_WORKER_URL/api/v1/org-templates"

# 一键开通（需 org:write；dev 模式可用）
curl -s -X POST "$ASH_WORKER_URL/api/v1/org-templates/mid_enterprise/provision" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Acme Engineering","slug":"acme-eng"}'
```

控制台 **空间** 页提供「组织样板」卡片：选择样板 → 开通 → 刷新 Org/Space 列表。

Provision 行为：创建 Org + admin 成员 + 额外角色 + 各 Space 的 resource_scope（含场景策略种子）+ 审计 `org.template_provisioned`。

## 5. 验收对照（PRD §3）

| 验收项 | 证据 |
|--------|------|
| 2–3 个组织样板 | 本文件 §3 + Catalog 三件套 |
| 谁付费/决策/审批 | 各样板字段 `payer` / `decisionMaker` / `approver` |
| 价值指标 | `recommendedKpis` 对齐 KPI 看板 |
| 可落地 | `make` 无关；`POST …/provision` + 单测 `TestListAndProvisionOrgTemplates` |
