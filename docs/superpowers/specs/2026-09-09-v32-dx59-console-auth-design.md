# v3.2 DX59 — 控制台登录入口 + Session 面板 + smoke

> Status: **done** (2026-09-09)  
> Program: T3 IdP/Session · Approach **Option 1** · **no new tables**  
> Freeze → **DX60**

## Goals

- Console login entry at `/ui/login`：密码登录 + OIDC 浏览器闭环
- Space 页 **Sessions** 面板：list / revoke / refresh current（DX57–58 API）
- `make oidc-console-smoke` → `doc/evidence/oidc-console-smoke-latest.md`
- vitest 覆盖 hash 入库与 Sessions 面板（mock）

## Non-goals

- 全站强制登录门闸  
- Device mint 可视化 scope 编辑器  
- 删除密码登录或 Space `dev-login`  
- v3.2 范围冻结（DX60）

## Login `/ui/login`

- 表单：email / password / optional spaceId → `POST /api/v1/auth/login` → `setAuthSession` → navigate `/ui/space`
- 按钮「OIDC 登录」→ `window.location = /api/v1/auth/oidc/login?ui=1`
- Mount：若 `location.hash` 含 `token`（及可选 `spaceId`）→ `setAuthSession` → 清 hash → `/ui/space`
- AppLayout / Space：链到 `/ui/login`；保留现有 `dev-login`

## OIDC UI redirect（后端薄改）

- `GET /auth/oidc/login?ui=1`：state 登记 `ui=true`（进程内 map，无新表）
- Callback 成功且 state 带 `ui`：`302` → `/ui/login#token=<jwt>&spaceId=<id>`（URL-encode）
- 无 `ui`：仍 `200` JSON `AuthSessionResponse`（API/测试兼容）
- OIDC 未启用：登录按钮点击后看到后端 `404 OIDC_DISABLED` JSON（可接受 POC）

## Sessions 面板（SpacePage）

| 动作 | API |
|------|-----|
| 列表 | `GET /auth/sessions` |
| 撤销 | `DELETE /auth/sessions/{sid}` |
| 刷新当前 | `POST /auth/sessions/refresh` → 更新 localStorage token |
| 当前会话 | `/auth/me` 的 `session`（若有） |

列：sid、typ、did、status、exp（本地格式化）。

## Client API

`frontend/src/modules/platform/api/platform.api.ts`（或 `auth.api.ts`）：

- `passwordLogin` · `listAuthSessions` · `revokeAuthSession` · `refreshAuthSession`
- 扩展 `AuthMe` / session 类型与 DX57–58 对齐

## Smoke

```bash
make oidc-console-smoke
# → doc/evidence/oidc-console-smoke-latest.md
```

- 无 live IdP：检查路由注册、OpenAPI 含 oidc/sessions、前端符号；记录 skip  
- 可选 env 齐备时冒烟 login?ui=1 链（不强制）

## Verification

```bash
cd frontend && npm test -- --run LoginPage SessionsPanel   # or equivalent
go test ./internal/api/ -run 'OIDC|AuthSession' -count=1
make oidc-console-smoke
make openapi-check   # if oidc login query documented
```
