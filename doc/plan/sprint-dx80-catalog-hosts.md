# Sprint DX80 — 包源主机白名单 / Catalog host allowlist

> **前置 / Prerequisite：** DX79；[`v4.3-release-scope.md`](v4.3-release-scope.md)  
> **状态 / Status：** ✅  
> **做法 / Approach：** HTTP(S) 只允许 `ASH_SKILL_CATALOG_HOSTS` 中的主机。未配置则拒绝远程 URL。本地相对路径与 `file://` 不变。无新表。  
> HTTP(S) is allowed only for hosts in `ASH_SKILL_CATALOG_HOSTS`. Unset refuses remote URLs. Local paths and `file://` stay as they are. No new table.

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| DX80-1 | `fetchCatalogBytes` 在拨号前校验主机 | ✅ |
| DX80-2 | 未配置或未列入则拒绝；本地路径仍可读 | ✅ |
| DX80-3 | 已有 HTTP 安装测试登记测试服主机 | ✅ |

## 验收 / Verify

```bash
go test ./internal/skills/ -count=1 -run 'TestFetchCatalogBytes|TestInstallFromCatalogHTTP'
```
