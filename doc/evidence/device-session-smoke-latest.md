# Device session smoke (DX65)

- stamp: `2026-09-09T15:44:47Z`
- host: `MINGW64_NT-10.0-26200`
- status: **pass**
- openapi device path: **pass**
- frontend mint UI: **pass**

## Checks

| Check | Result |
|-------|--------|
| go test DeviceMint / Scope / Jti | pass |
| OpenAPI /auth/sessions/device | pass |
| SpacePage mint form + API client | pass |

## Test excerpt

```
ok  	github.com/ash-repwiki/ash/internal/api	5.995s
```
