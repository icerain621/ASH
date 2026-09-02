# DX14 Addendum: Waker Console + v2.4 Freeze

> Status: **approved** (2026-09-02)  
> Parent: [`2026-09-01-waker-duties-design.md`](2026-09-01-waker-duties-design.md)  
> Version lane: **v2.4** (DX12–DX13 code done)

## Decisions

- **Observability** is the primary Waker ops surface (status / duties / queue / dry-run sweep / gated cancel).
- **Scale** shows compact counts only (enabled duties + queue length) via existing Waker APIs — **no** new Scale readiness schema fields.
- `v2.4-release-scope` marked **已冻结**; `make v2.4-signoff` mirrors v2.3 gate (scope + openapi + doctor + waker-smoke); **no automatic tag**.
- Cancel UI only when `allowCancel` from `/waker/status`; confirm phrase `CANCEL_STALE_RUNS`.

## Out

- New SQL tables; auto-seed doctor/kpi duties; changing cancel gates; cloud live as hard gate
