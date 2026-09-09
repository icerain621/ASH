# Sprint DX63 — jti 吊销硬化（v4.0 · A）

| # | 项 | 状态 |
|---|-----|------|
| DX63-1 | registry 存 access/refresh + prev jti；rotate 推移 | ✅ |
| DX63-2 | middleware/refresh 校验；`AUTH_TOKEN_REPLAY`；grace env | ✅ |
| DX63-3 | 测试（grace=0 / 宽限内 / revoke / legacy 无 jti） | ✅ |
| DX63-4 | OpenAPI + CHANGELOG + TODO | ✅ |

## 设计

- [`docs/superpowers/specs/2026-09-09-v40-dx63-jti-revoke-design.md`](../../docs/superpowers/specs/2026-09-09-v40-dx63-jti-revoke-design.md)
- `ASH_AUTH_TOKEN_GRACE_SEC` 默认 60，clamp 0–300；0 = 只认当前对
- **无新表**

## 非目标

- 完整 jti 黑名单表 → 不做  
- 控制台强制登录 → **DX64**
