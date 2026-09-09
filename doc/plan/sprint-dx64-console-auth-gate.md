# Sprint DX64 — 控制台可选强制登录（v4.0 · A）

| # | 项 | 状态 |
|---|-----|------|
| DX64-1 | `ASH_CONSOLE_AUTH_REQUIRED` → `/readyz.consoleAuthRequired` | ✅ |
| DX64-2 | SPA `beforeLoad` 无 token → `/login`；401 清会话并跳转（门闸开） | ✅ |
| DX64-3 | 门闸开时隐藏 Dev Token | ✅ |
| DX64-4 | 测试 + OpenAPI + CHANGELOG | ✅ |

## 设计

- [`docs/superpowers/specs/2026-09-09-v40-dx64-console-auth-gate-design.md`](../../docs/superpowers/specs/2026-09-09-v40-dx64-console-auth-gate-design.md)
- 默认 **off**；不改 `ASH_AUTH_MODE`

## 非目标

- Device mint UI → **DX65**
