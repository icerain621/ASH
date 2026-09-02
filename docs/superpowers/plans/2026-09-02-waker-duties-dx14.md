# Waker Console + v2.4 Freeze (DX14) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship Observability Waker ops panel + Scale compact counts, freeze `v2.4-release-scope`, and add `make v2.4-signoff` (no automatic tag).

**Architecture:** Frontend clients call existing `/waker/status|queue|duties|sweep|duties/{id}/run`. No new backend schema. Signoff gate mirrors `scripts/v23-signoff-gate.sh` with `waker-smoke` instead of rag-hybrid.

**Tech Stack:** React 18, TanStack Query, existing ASH UI patterns, Bash signoff gates, Makefile.

**Spec:** [`docs/superpowers/specs/2026-09-01-waker-duties-design.md`](../specs/2026-09-01-waker-duties-design.md) · Addendum [`docs/superpowers/specs/2026-09-02-waker-duties-dx14-addendum.md`](../specs/2026-09-02-waker-duties-dx14-addendum.md)

## Global Constraints

- Cancel gates unchanged: env `ASH_WAKER_ALLOW_CANCEL=1` + confirm `CANCEL_STALE_RUNS` + `dryRun=false`
- Background never cancels; UI dry-run sweep default; cancel only if `allowCancel`
- No new SQL / RLS; no Scale readiness OpenAPI field additions
- Do not auto-seed `doctor_subset` / `kpi_drift`
- Commit messages in Chinese; do not push unless asked
- Shell: Git Bash; `export GOPROXY=https://goproxy.cn,direct` if needed
- Tag `v2.4.0` is **manual** after human sign-off

## File map

| Path | Role |
|------|------|
| `frontend/src/modules/waker/api/waker.api.ts` | Typed clients for status/queue/duties/sweep/run |
| `frontend/src/pages/ObservabilityPage.tsx` (+ test) | Waker panel |
| `frontend/src/pages/ScalePage.tsx` | Compact waker counts |
| `scripts/v24-signoff-gate.sh`, `Makefile` | `make v2.4-signoff` |
| `scripts/scope-freeze-gate.sh` | Include v2.4 scope |
| `doc/plan/v2.4-release-scope.md` | **已冻结** |
| `doc/checklists/v2.4-signoff.md`, `doc/evidence/v2.4-signatures-template.md` | Checklist + template |
| `doc/plan/sprint-dx14-v24-signoff.md`, TODO/CHANGELOG/PLAN/smoke-index | Watermarks |

---

### Task 1: Waker API client + Observability panel

**Files:**
- Create: `frontend/src/modules/waker/api/waker.api.ts`
- Modify: `frontend/src/pages/ObservabilityPage.tsx`
- Modify: `frontend/src/pages/ObservabilityPage.test.tsx` (mock waker APIs)

**Interfaces (TypeScript):**

```ts
export type WakerStatus = {
  duties: Array<{ id: string; spaceId: string; kind: string; enabled: boolean; intervalMs: number; nextRunAt: string }>;
  recentRuns: Array<{ id: string; dutyId: string; kind: string; status: string; matched: number; flagged: number; canceled: number; summary: string; startedAt: string }>;
  allowCancel: boolean;
  interval?: string;
  intervalMs?: number;
};
export type WakerQueue = { items: Array<{ runId: string; spaceId: string; status: string; reason: string; kind?: string }>; count: number };
export function getWakerStatus(spaceId?: string, recent?: number): Promise<WakerStatus>;
export function getWakerQueue(params?: { spaceId?: string; limit?: number }): Promise<WakerQueue>;
export function listWakerDuties(spaceId?: string): Promise<{ duties: ... } | Array<...>>; // match API shape
export function postWakerSweep(body: { spaceId?: string; dryRun?: boolean; action?: string; confirm?: string; maxAge?: string }): Promise<unknown>;
export function postWakerDutyRun(id: string, body?: { dryRun?: boolean }): Promise<unknown>;
```

Inspect OpenAPI / `internal/waker` JSON tags for exact list duties response shape (`[]` vs `{duties:[]}`) and match the client.

- [ ] **Step 1: Add API module** using `api()` from `@/services/http/client` (same pattern as `scale.api.ts`).

- [ ] **Step 2: Observability panel UI**

Add a section titled **Waker** below existing RAG/scale blocks:

- Query `getWakerStatus` + `getWakerQueue` with `activeSpaceId`
- Table/list: duties kind, enabled, nextRunAt
- Recent runs: kind, status, matched/flagged, summary (truncate)
- Queue preview: up to 10 items (reason + kind)
- Buttons:
  - Refresh
  - Dry-run sweep → `postWakerSweep({ dryRun: true, action: "report" })`
  - Run selected/first enabled duty dry-run → `postWakerDutyRun`
- If `allowCancel`: show confirm input + Cancel stale button calling sweep with `action:"cancel", dryRun:false, confirm:"CANCEL_STALE_RUNS"`; disable unless input exact match
- If `!allowCancel`: show short note that cancel requires `ASH_WAKER_ALLOW_CANCEL=1`

Keep visual style consistent with Observability (tables, existing buttons) — no new design system.

- [ ] **Step 3: Vitest**

Extend `ObservabilityPage.test.tsx` to mock waker APIs and assert "Waker" heading / duty kind text appears.

- [ ] **Step 4: Verify**

```bash
cd frontend && npm test -- --run ObservabilityPage
# or repo: make web-test / targeted vitest
```

- [ ] **Step 5: Commit**

```bash
git commit -m "$(cat <<'EOF'
feat(web): DX14 Observability Waker 运维面板

展示 duties/queue/recent runs；dry-run sweep；cancel 三重闸门可见。
EOF
)"
```

---

### Task 2: Scale compact Waker counts

**Files:**
- Modify: `frontend/src/pages/ScalePage.tsx`
- Optional: Scale page test if one exists

- [ ] **Step 1:** On ScalePage, `useQuery` `getWakerStatus` + `getWakerQueue({ limit: 5 })`.

- [ ] **Step 2:** Render a small row/card: `Waker duties enabled: N` · `queue: M` · ticker interval from status.

- [ ] **Step 3:** No backend Scale schema changes.

- [ ] **Step 4: Commit**

```bash
git commit -m "$(cat <<'EOF'
feat(web): DX14 Scale 页 Waker 紧凑计数

仅展示 enabled duties 与 queue 长度，无新 readiness 字段。
EOF
)"
```

---

### Task 3: Freeze v2.4 + signoff gate

**Files:**
- Modify: `doc/plan/v2.4-release-scope.md` → status **已冻结**; fill signoff table placeholders like v2.3; §4 freeze rules active
- Create: `scripts/v24-signoff-gate.sh` (copy v23; replace rag-hybrid with `waker-smoke`; check v2.4 scope/checklist/template)
- Modify: `Makefile` — `v2.4-signoff` target + `.PHONY`
- Modify: `scripts/scope-freeze-gate.sh` — `check_scope` for `v2.4-release-scope.md`
- Create: `doc/checklists/v2.4-signoff.md`
- Create: `doc/evidence/v2.4-signatures-template.md`

- [ ] **Step 1: Write gate script**

Must:

```bash
run_step scope-freeze-gate bash scripts/scope-freeze-gate.sh
run_step openapi-check bash scripts/openapi-check.sh
run_step doctor-all ...
run_step doctor-m4 ...
run_step waker-smoke bash scripts/waker-smoke.sh
# grep 已冻结 on v2.4-release-scope.md
# require checklist + signatures template
# print tag instructions for v2.4.0 — do NOT tag
```

- [ ] **Step 2: Freeze scope doc** — header `状态：**已冻结**`; In table DX12–DX14; link checklist.

- [ ] **Step 3: Run**

```bash
make scope-freeze-gate
make v2.4-signoff
```

Expected: `OK v2.4-signoff`.

- [ ] **Step 4: Commit**

```bash
git commit -m "$(cat <<'EOF'
docs: DX14 冻结 v2.4 并加入 v2.4-signoff 门禁

范围已冻结；签字门禁含 waker-smoke；不自动打 tag。
EOF
)"
```

---

### Task 4: Sprint note + watermarks

**Files:**
- Create: `doc/plan/sprint-dx14-v24-signoff.md`
- Modify: `doc/plan/TODO.md` (DX14 ✅; conclusion v2.4 frozen)
- Modify: `CHANGELOG.md`, `doc/plan/PLAN-进度与里程碑.md`, `doc/plan/README.md`, `doc/checklists/smoke-index.md`
- Commit addendum + this plan under `docs/superpowers/`

- [ ] **Step 1:** Watermarks — Doctor/SQL unchanged; **v2.4 DX12–DX14 已冻结**; next open for post-v2.4 / P0 cloud

- [ ] **Step 2:** `make v2.4-signoff` green one more time if docs-only changes don't affect gate

- [ ] **Step 3: Commit**

```bash
git commit -m "$(cat <<'EOF'
docs: DX14 sprint 与 v2.4 水位收口

控制台 + 签字门禁文档对齐；tag 仍人工。
EOF
)"
```

---

## Plan self-review

| Spec | Task |
|------|------|
| Observability Waker panel + actions | T1 |
| Scale compact counts | T2 |
| Freeze + `make v2.4-signoff` | T3 |
| Docs / TODO / CHANGELOG | T4 |
| No new Scale OpenAPI fields | T2 constraint |
| No auto tag | T3 |
