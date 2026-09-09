# OIDC / console auth smoke (DX59)

- stamp: `2026-09-09T15:44:28Z`
- host: `MINGW64_NT-10.0-26200`
- status: **pass**
- openapi markers: **pass**
- frontend markers: **pass**
- live IdP: **skipped** — set ASH_OIDC_ENABLED=1 + IdP env for live browser OIDC (optional)

## Checks

| Check | Result |
|-------|--------|
| go test OIDC/AuthSession/idp | pass |
| OpenAPI ui + sessions/refresh | pass |
| LoginPage + /ui/login + Sessions panel | pass |

## Test excerpt

```
ok  	github.com/ash-repwiki/ash/internal/api	16.833s
ok  	github.com/ash-repwiki/ash/internal/idp	0.943s
```
