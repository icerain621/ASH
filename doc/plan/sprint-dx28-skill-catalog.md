# Sprint DX28 — 组织 Skill catalog + UI（v2.7）

> 状态：**完成**（2026-09-06）  
> 范围：[`v2.7-release-scope.md`](v2.7-release-scope.md)

## 交付

| # | 项 | 状态 |
|---|-----|------|
| DX28-1 | Catalog JSON（`ash.skill.catalog.v1`）+ 可选 catalog HMAC + publisher 过滤 | ✅ |
| DX28-2 | `GET /skills/catalog` · `POST /skills/catalog/install`（复用 pack verify/install） | ✅ |
| DX28-3 | Automation：验签干跑、错误提示、catalog 已安装提示/安装 | ✅ |
| DX28-4 | smoke / OpenAPI / docs；**无新表** | ✅ |

## 约定

| 变量 / 路径 | 说明 |
|-------------|------|
| `.ash/skill-catalog.json` | 默认本地 catalog（相对 repoRoot） |
| `ASH_SKILL_CATALOG_PATH` | 覆盖本地路径 |
| `ASH_SKILL_CATALOG_URL` | HTTPS catalog（优先） |
| 条目字段 | `name,version,publisher,url,digest?,signature?` |
| catalog `signature` | 可选；材料见 `CatalogSignMaterial` |
| pack 验签 | 仍用 `ASH_SKILL_PACK_SIGNING_KEY` + allowlist/spaces |

## 验收

```bash
go test ./internal/skills/ -run 'TestCatalog|TestInstallFromCatalog' -count=1
go test ./internal/api/ -run TestSkillCatalog -count=1
make skill-pack-smoke
make openapi-check
```
