# DX13 Addendum: Multi-duty Probes (方案 A)

> Status: **approved** (2026-09-02)  
> Parent: [`2026-09-01-waker-duties-design.md`](2026-09-01-waker-duties-design.md)  
> Version lane: **v2.4** (DX12 done; DX14 UI/freeze still out)

## Decisions (方案 A)

- **Do not** auto-seed `doctor_subset` / `kpi_drift` on Worker boot or `/status`.
- Duties are created **only** via explicit ensure helpers (tests / future admin API / documented package calls).
- Default config when ensuring:
  - `doctor_subset`: `{"suite":"M4"}`；optional `caseIds: string[]` to filter results
  - `kpi_drift`: `{"metric":"KPI-17","threshold":50}`；optional `baseline` (absolute count); if baseline omitted, compare to `evolveBacklogTarget`-style threshold only (absolute), or relative to previous ok duty_run `matched` when `mode":"delta"` present
- Never cancel; never auto-heal.
- No new SQL revision (reuse `waker_duties` / `waker_duty_runs`).

## Queue merge

`GET /waker/queue` returns stale_run candidates **plus** synthetic items from recent `waker_duty_runs` where `kind IN (doctor_subset, kpi_drift)` and `flagged > 0` (or `status=failed`), with `Item.kind` and `reason` as approved (`doctor_subset:<caseId>`, `kpi_drift:<metric>`).

## Wiring

Inject optional runners into `waker.Service` (avoid hard circular imports with full Doctor graph in unit tests):

- `DoctorRunner` — runs a named suite; returns pass/fail per case
- KPI probe may query backlog counts via DB (same SQL as KPI-17) without requiring `metrics.Service`

## Out

- Observability UI, `v2.4-signoff`, new tables, default-enabled noisy probes
