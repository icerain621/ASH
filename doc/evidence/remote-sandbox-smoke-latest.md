# Remote sandbox smoke evidence (DX40)

| Field | Value |
|-------|--------|
| Status | **pass** (exit 0) |
| Platform | MINGW64_NT-10.0-26200 |
| Date | 2026-09-09T13:24:49Z |
| Scope | remote unit / mock / prefer+fallback / e2b httptest |
| Live E2B | **skipped** — set ASH_SANDBOX_REMOTE_LIVE=1 + ASH_SANDBOX_REMOTE_API_KEY for live E2B |
| Doctor | unchanged ALL 57 / M4 10 (no new case; D5) |

## Env

| Variable | Default / note |
|----------|----------------|
| `ASH_SANDBOX_REMOTE` | off (opt-in prefer for isolated) |
| `ASH_SANDBOX_REMOTE_BACKEND` | `mock` \| `e2b` |
| `ASH_SANDBOX_REMOTE_ON_FAIL` | fallback (or `deny`) |
| `ASH_SANDBOX_REMOTE_LIVE` | `1` enables optional live segment |
| `ASH_SANDBOX_REMOTE_API_KEY` | required for live; absent → skip pass |

## Raw excerpt

```
ok  	github.com/ash-repwiki/ash/internal/sandbox	0.506s
ok  	github.com/ash-repwiki/ash/internal/sandbox/remote	2.978s
```
