# Design: Waker Duties Plane (v2.4 · DX12–DX14)

> Status: approved for planning (2026-09-01)  
> Version lane: **v2.4 draft** (v2.3 frozen DX9–DX11; builds on v2.2 Waker DX6–DX8)  
> Related: `internal/waker`, `/api/v1/waker/*`, agentic roadmap「Waker 雏形 → 持续职责」

## Goal

Upgrade Waker from **TTL stale-run scan** into a **schedulable, auditable, observable continuous-duty plane**: duty ledger + probes (`stale_run`, `doctor_subset`, `kpi_drift`) + ops console. Preserve v2.2 cancel safety (background never cancels; human cancel still requires env + confirm phrase + `dryRun=false`).

## Non-goals

- Landlock / E2B OS sandbox
- Production-default auto cancel (gates unchanged)
- Cloud RDS / GitHub·ExecGo live as hard release blockers
- Vector DB, Skill Marketplace, microservice split of Waker
- Changing Hybrid RAG schema or v2.3 frozen In-scope APIs (except Scale/Obs display hooks if needed)

## Sprint split

| Sprint | Theme | Delivers |
|--------|-------|----------|
| **DX12** | Duty ledger + migrate `stale_run` | Schema + CRUD/status APIs + ticker reads ledger; existing queue/sweep keep working |
| **DX13** | Multi-duty probes | `doctor_subset` + `kpi_drift` → unified queue items + duty_run rows |
| **DX14** | Console + freeze | Observability/Scale Waker panel; `v2.4-release-scope` + `make v2.4-signoff` |

Tag `v2.4.0` remains **manual** after human sign-off (same culture as v2.1–v2.3).

## Architecture

```
  ASH_WAKER_INTERVAL ticker ──► DutyScheduler ──► Probe(kind) ──► waker_duty_runs
                                         │
                                         ▼
                              Queue items (unified) ◄── GET /waker/queue
                                         │
  POST /waker/sweep (report|cancel) ─────┘   cancel: DX7 gates only
                                         │
  GET /waker/status · GET|POST /waker/duties │
                                         ▼
                              Console (Observability / Scale)
```

- Package: extend `internal/waker` (`duty.go`, `probes.go`, scheduler); keep `Queue`/`Sweep` entrypoints.
- Probes are **read-mostly**; only `action=cancel` mutates run status, and only under DX7 gates.
- When ledger tables are empty (fresh migrate), seed default enabled `stale_run` duty per space (or global+space override — prefer **per space_id**, with `space_id=""` meaning install-wide default only if RLS allows; **prefer always scoped `space_id`**, including `local`).

## Schema (SQL rev 28 → 29, RLS 48 → 50)

Mirror Hybrid pattern: create tables + inline RLS policies + `ash_app` / `ash_rls_tester` grants (see `000028_rag_hybrid_index.up.sql`). Update `expectedVersion` and `PostgresRLSExpectedPolicyCount()`.

### `waker_duties`

| Column | Notes |
|--------|--------|
| id | PK (text/uuid) |
| space_id | tenant; RLS |
| kind | `stale_run` \| `doctor_subset` \| `kpi_drift` |
| enabled | bool |
| interval_ms | schedule cadence (min clamp ≥ 60s) |
| config_json | probe-specific JSON |
| next_run_at | timestamptz / unix ms consistent with store style |
| updated_at | |

Unique: `(space_id, kind)`.

### `waker_duty_runs`

| Column | Notes |
|--------|--------|
| id | PK |
| space_id | tenant; RLS |
| duty_id | FK → waker_duties |
| kind | denormalized for queries |
| status | `ok` \| `failed` \| `skipped` |
| matched | int |
| flagged | int |
| canceled | int (usually 0 for background) |
| summary | short text |
| started_at / finished_at | |

Indexes: `(space_id, started_at DESC)`, `(duty_id, started_at DESC)`.

SQLite: GORM AutoMigrate or SQL pilot parity as existing store path requires; Postgres migration is source of truth for rev bump.

## Probes

### `stale_run` (DX12)

- Same candidates as today: `status IN (running, waiting_approval)` older than TTL (`config.maxAge` or `ASH_WAKER_RUN_TTL`).
- Background: **report/flag only** → write `waker_duty_runs`; emit `waker.duty_completed` (and keep `waker.sweep_completed` when `/sweep` is used).

### `doctor_subset` (DX13)

- `config.suite` whitelist (e.g. `M4` subset case IDs); run in-process Doctor subset or CLI-equivalent service call.
- Failure → queue item `reason=doctor_subset:<caseId>`; no auto-remediation beyond flag.

### `kpi_drift` (DX13)

- `config.metric` + `config.threshold` (relative or absolute vs baseline snapshot stored in config or last ok run).
- Drift beyond threshold → queue item `reason=kpi_drift:<metric>`; no auto cancel.

## API

Keep:

- `GET /api/v1/waker/queue` — unify items; extend `Item.reason` / optional `kind` field for probe origin (backward compatible: new fields additive).
- `POST /api/v1/waker/sweep` — stale_run path; cancel gates **unchanged**.

Add:

- `GET /api/v1/waker/status` — last duty runs, enabled duties, ticker config (interval, allowCancel flag without secrets).
- `GET /api/v1/waker/duties?spaceId=` — list duties.
- `POST /api/v1/waker/duties/{id}/run` — force one report pass (`dryRun` default true).

OpenAPI + swagger + `make openapi-check`. Frontend clients under `frontend/src/modules/` if UI needs them.

## UI (DX14)

- Panel on **Observability** (primary) and compact counts on **Scale** readiness if cheap:
  - duties enabled / next_run
  - last duty_run summary
  - queue preview
  - dry-run sweep button
- Cancel UI: only if `allowCancel` visible from status; require confirm phrase input matching `CANCEL_STALE_RUNS` (never hide gates).

## Env / compatibility

| Env | Behavior |
|-----|----------|
| `ASH_WAKER_INTERVAL` | Ticker on/off (unchanged meaning); scheduler iterates due duties |
| `ASH_WAKER_RUN_TTL` | Default for `stale_run` |
| `ASH_WAKER_ALLOW_CANCEL` | Cancel gate (unchanged) |

Legacy installs without rows: first Worker boot or first `/status` **ensures** default `stale_run` duty.

## Acceptance

```bash
go test ./internal/waker/... -count=1
make openapi-check
make waker-smoke          # extended for status + duty run
# DX14:
make v2.4-signoff         # scope + doctor + waker-smoke (+ openapi)
```

Docs: `doc/plan/v2.4-release-scope.md` (draft → frozen in DX14), sprint notes DX12–DX14, smoke checklist update, TODO/CHANGELOG.

## Risks

| Risk | Mitigation |
|------|------------|
| Doctor probe too slow/heavy | Cap suite; timeout; mark `skipped` |
| KPI false positives | Conservative defaults; flag-only |
| RLS / rev drift | Same gate pattern as Hybrid 28/48 |
| Scope creep into auto-heal | Explicit Non-goal; cancel gates stay |

## Out of this design (later)

- Persistent ack/snooze of queue items
- Cross-space global operator role beyond existing authz
- External cron / message bus
