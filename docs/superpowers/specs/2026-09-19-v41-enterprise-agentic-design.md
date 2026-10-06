# v4.1 企业 Agentic 设计（B 轨 · DX67–DX72）

> Status: **approved for scope write-up** (2026-09-19 · approach A)  
> Program: [`doc/plan/v4.x-program.md`](../../../doc/plan/v4.x-program.md) · Scope: [`doc/plan/v4.1-release-scope.md`](../../../doc/plan/v4.1-release-scope.md)

## Intent

After frozen v4.0 Auth, deliver **enterprise Agentic visibility and fail-closed placeholders**:

1. Org/space **quotas** (concurrent runs · token proxy budget) — configure + enforce on create/spawn  
2. **Thin audit report** — aggregate existing audit events per space (no new tables)  
3. **Spawn policy / budget** — SpacePolicy overrides Harness sub-run limits; surface on spawn/tree

Tag target: **`v4.1.0`** (manual; F6).

## Approach (A)

Six sprints mirroring v4.0 cadence. Prefer extending `SpacePolicyPack.bodyJson` + read APIs over new tables (F4).

## Non-goals

- Real billing / invoices / payment  
- Cross-org rollup dashboards  
- SAML / force-default OIDC  
- Plugin gRPC / external Indexer → **v4.2**  
- Marketplace / multi-region → **v4.3**  
- Force-kill of already-running Runs when quotas tighten  
- Voice / IM gateway

## Data & merge

| Key | Where | Notes |
|-----|--------|--------|
| `quotas.maxConcurrentRuns` | `bodyJson.quotas` | `0` / omit = unlimited |
| `quotas.tokenBudgetProxy` | `bodyJson.quotas` | soft proxy counter; create/spawn check |
| `subRun.maxDepth` | `bodyJson.subRun` | overrides Harness when set |
| `subRun.allowedTools` | `bodyJson.subRun` | default still fail-closed child allowlist |
| `subRun.tokenBudgetProxy` | `bodyJson.subRun` | per child; visible on spawn response |

Merge: platform/org-template defaults → space kind → pack → ResourceScope; **stricter wins**. Unconfigured quotas = no limit.

Enforcement: count active runs in space at Create/Spawn; reject with stable error code when over. Optional env kill-switch for local YOLO-off still fail-closed when configured.

## APIs (sketch)

| Method | Path | Sprint |
|--------|------|--------|
| (existing) PUT/GET space policy | `…/spaces/{id}/policy` | DX67 body shape |
| GET | `…/spaces/{id}/quotas` | DX68 usage + limits |
| GET | `…/spaces/{id}/audit-report?window=` | DX69 counts |
| (existing) POST sub-runs | enrich response/errors | DX71 |

OpenAPI + console cards in DX68/DX70/DX71.

## Sprint map

| ID | Theme |
|----|--------|
| DX67 | quotas schema + Effective + create/spawn gate |
| DX68 | quotas read API + Space/console projection |
| DX69 | audit-report API (scan audit_log) |
| DX70 | Compliance/Space report card |
| DX71 | bodyJson.subRun + spawn visibility |
| DX72 | freeze scope + `make v4.1-signoff` |

## Verification (generation)

```bash
make openapi-check
go test ./internal/spacepolicy/ ./internal/runs/ ./internal/api/ -count=1
make v4.1-signoff   # after DX72
```

## Self-review

- [x] No placeholders for In/Out  
- [x] F4 no-new-table explicit  
- [x] Out list matches program F5/F non-goals  
- [x] Sprint IDs contiguous DX67–72  
- [x] Does not reopen v4.0 Auth or v5 freeze
