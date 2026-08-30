# Sprint DM：Space Rules + Goal 路由（方案 C）Implementation Plan

> **方案：** 已批准 **C** = DB 持久化 + `.ash/rules.yaml` 双向同步 + Goal 路由闭环  
> **Goal:** Space 级路由/填槽规则可 CRUD；Import/Export 与仓库文件同步；`from-goal` 优先使用 Space Rules。  
> **状态：** ✅ 完成（方案 C）

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DM-1 | `space_rules` 表 SQL 26 / RLS 46 + `internal/spacerules` | ✅ |
| DM-2 | API GET/PUT + import/export + preview | ✅ |
| DM-3 | Goal `Route` 注入 Space Rules | ✅ |
| DM-4 | Space 页 Rules 面板 + vitest | ✅ |
| DM-5 | openapi / docs / 回归 | ✅ |

## 规则文档（JSON/YAML）

```yaml
version: 1
preferScenario: ""          # 可选强制场景
route:
  security_patch: [security, cve, 漏洞]
  hotfix: [hotfix, 线上, 热修]
  feature_delivery: []
defaults:
  policyProfile: default
  inputs: {}                # 合并进 plan inputs
```

文件路径：`{repoRoot}/.ash/rules.yaml`

## API

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/v1/spaces/{spaceId}/rules` | 读规则（无则内置默认） |
| PUT | `/api/v1/spaces/{spaceId}/rules` | 写规则到 DB |
| POST | `/api/v1/spaces/{spaceId}/rules/import` | `.ash/rules.yaml` → DB |
| POST | `/api/v1/spaces/{spaceId}/rules/export` | DB → `.ash/rules.yaml` |
| POST | `/api/v1/spaces/{spaceId}/rules/preview` | 预览 Goal 路由结果 |

## 退出标准

- [x] SQL 26 / RLS 46；openapi-check 绿
- [x] Space Rules 影响 from-goal（`routeReason` 含 `space_rule:`）
- [x] import/export 往返一致
- [x] Space 页可编辑/同步；无规则时回退 DJ 内置关键词
