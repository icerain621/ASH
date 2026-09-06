# Skill pack 烟测（Sprint DX22）

```bash
make skill-pack-smoke
```

| 覆盖 | 说明 |
|------|------|
| `TestBuildSignVerifyInstallPack` 等 | 构建 / 验签 / 安装 / allowlist |
| `TestSkillPackVerifyAndInstallAPI` | HTTP verify + install + 坏签名 |
| CLI `ash skill-pack build\|sign` | zip + 64-hex 签名 |

## Env

| 变量 | 说明 |
|------|------|
| `ASH_SKILL_PACK_SIGNING_KEY` | HMAC 密钥（可回退 `ASH_PLUGIN_SIGNING_KEY`） |
| `ASH_SKILL_PACK_ALLOWLIST` | publisher 白名单；空/`*`=任意 |
| `ASH_SKILL_PACK_SPACES` | 允许安装的 spaceId；空/`*`=任意 |

## 相关

- [`../plan/sprint-dx22-skill-packs.md`](../plan/sprint-dx22-skill-packs.md)
- [`../plan/v2.6-release-scope.md`](../plan/v2.6-release-scope.md)
