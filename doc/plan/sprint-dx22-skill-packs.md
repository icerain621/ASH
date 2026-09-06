# Sprint DX22：私有签名 Skill packs（v2.6）

> **方案：** 已批准 **B3/B5** = zip pack + HMAC（复用 plugin-sign 密钥模式）；**无新表**；公网市场 Out  
> **Goal：** `POST /skills/packs/verify|install`；落盘 `.ash/skills/<name>/`；`make skill-pack-smoke`  
> **状态：** ✅ 完成（代码）

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX22-1 | Pack 格式 + HMAC 验签/安装（filesystem） | ✅ |
| DX22-2 | API verify/install + 路由顺序 | ✅ |
| DX22-3 | CLI `ash skill-pack build\|sign` | ✅ |
| DX22-4 | Automation 安装 UI + vitest mock | ✅ |
| DX22-5 | OpenAPI + smoke + docs | ✅ |

## 约定

- Zip 扁平：`ash-skill-pack.json` + `SKILL.md`（+ 可选同级文件）
- 签名材料：`publisher\nname\nversion\ndigestHex`（digest = sha256(SKILL.md)）
- 密钥：`ASH_SKILL_PACK_SIGNING_KEY`（回退 `ASH_PLUGIN_SIGNING_KEY`）；**无密钥则安装失败**（fail closed）
- Allowlist：`ASH_SKILL_PACK_ALLOWLIST`（publisher；空/`*`=任意）
- Spaces：`ASH_SKILL_PACK_SPACES`（空/`*`=任意）

## 退出标准

- [x] 坏签名 / 未知 publisher / 禁 space → 拒绝
- [x] 安装后 `GET /skills` 可见
- [x] `make skill-pack-smoke` 绿
- [x] 无 SQL/RLS bump
