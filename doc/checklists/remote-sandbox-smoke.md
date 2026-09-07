# Remote sandbox 烟测（Sprint DX40）

```bash
make remote-sandbox-smoke
# 可选 live（需 key）：
# ASH_SANDBOX_REMOTE_LIVE=1 ASH_SANDBOX_REMOTE_API_KEY=... make remote-sandbox-smoke
```

写入证据：[`../evidence/remote-sandbox-smoke-latest.md`](../evidence/remote-sandbox-smoke-latest.md)

| 覆盖 | 说明 |
|------|------|
| `./internal/sandbox` + `remote` 单测 | mock、E2B httptest、prefer/fallback/deny |
| Live E2B | 仅 `ASH_SANDBOX_REMOTE_LIVE=1`；无 key → **skip pass** |
| Doctor | **不**新增用例（D5：ALL 57 / M4 10） |

## Env

| 变量 | 说明 |
|------|------|
| `ASH_SANDBOX_REMOTE` | 默认关；`1`/`on`/`auto` → isolated prefer remote |
| `ASH_SANDBOX_REMOTE_BACKEND` | `mock`（默认）\| `e2b` |
| `ASH_SANDBOX_REMOTE_ON_FAIL` | 默认 fallback；`deny` 拒绝 |
| `ASH_SANDBOX_REMOTE_LIVE` | `1` 跑 `TestLiveE2B` |
| `ASH_SANDBOX_REMOTE_API_KEY` | live 必需；缺省 skip pass |
| `ASH_SANDBOX_REMOTE_URL` | 可选自建/测试网关 |

## 相关

- [`../plan/sprint-dx40-remote-sandbox-smoke.md`](../plan/sprint-dx40-remote-sandbox-smoke.md)
- [`../plan/v2.9-release-scope.md`](../plan/v2.9-release-scope.md)
- [`sandbox-smoke.md`](sandbox-smoke.md)（本机 Landlock/process；已含 remote 包测）
