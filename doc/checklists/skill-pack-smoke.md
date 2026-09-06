# Skill pack / catalog 烟测（Sprint DX22 / DX28）

```bash
make skill-pack-smoke
```

| 覆盖 | 说明 |
|------|------|
| `TestBuildSignVerifyInstallPack` 等 | 构建 / 验签 / 安装 / allowlist |
| `TestCatalog*` / `TestInstallFromCatalog*` | 组织 catalog 签名、过滤、file/HTTP 安装（**DX28**） |
| `TestSkillPackVerifyAndInstallAPI` | HTTP verify + install + 坏签名 |
| `TestSkillCatalogListAndInstallAPI` | `GET /skills/catalog` + `POST /skills/catalog/install` |
| CLI `ash skill-pack build\|sign` | zip + 64-hex 签名 |

## Env

| 变量 | 说明 |
|------|------|
| `ASH_SKILL_PACK_SIGNING_KEY` | HMAC 密钥（可回退 `ASH_PLUGIN_SIGNING_KEY`） |
| `ASH_SKILL_PACK_ALLOWLIST` | publisher 白名单；空/`*`=任意 |
| `ASH_SKILL_PACK_SPACES` | 允许安装的 spaceId；空/`*`=任意 |
| `ASH_SKILL_CATALOG_PATH` | 组织 catalog 本地 JSON（覆盖默认 `.ash/skill-catalog.json`） |
| `ASH_SKILL_CATALOG_URL` | 组织 catalog HTTPS（优先于 PATH） |

## 相关

- [`../plan/sprint-dx22-skill-packs.md`](../plan/sprint-dx22-skill-packs.md)
- [`../plan/sprint-dx28-skill-catalog.md`](../plan/sprint-dx28-skill-catalog.md)
- [`../plan/v2.6-release-scope.md`](../plan/v2.6-release-scope.md)
- [`../plan/v2.7-release-scope.md`](../plan/v2.7-release-scope.md)
