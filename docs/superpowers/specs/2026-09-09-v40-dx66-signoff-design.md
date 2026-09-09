# v4.0 DX66 — 范围冻结 + `make v4.0-signoff`

> Status: **done** (2026-09-09)  
> Program: A Auth · freeze only · **no new tables** · **no auto git tag**  
> Depends: DX61–DX65 delivered · mirror DX60 / `v32-signoff-gate.sh`

## Intent

Freeze `doc/plan/v4.0-release-scope.md` and land a release sign-off gate that proves Doctor/platform baseline plus v4.0 Auth harden evidence. Tag `v4.0.0` remains a **manual** human step after signatures.

## Decisions (locked)

| # | Choice |
|---|--------|
| Gate suite | **A**: v3.2-signoff baseline + Auth smokes (`oidc-console-smoke`, `device-session-smoke`, Auth/OIDC/Session tests, console FE tests) |
| Freeze style | **1**: Mark scope **已冻结** (product placeholder OK); signatures template separate; no auto tag |
| Approach | Clone DX60 template (dedicated `v40-signoff-gate.sh`), not parameterized multi-version script |

## Non-goals

- `git tag` / `git push` from the gate  
- Filling real four-person signatures  
- New product features beyond freeze/docs/gates  
- New SQL tables

## Deliverables

1. **Freeze** `doc/plan/v4.0-release-scope.md`
   - Status → **已冻结** (date + note DX66)
   - §2: DX61–DX66 all ✅
   - §5: drop「拟」; commands stay `scope-freeze-gate` + `v4.0-signoff`
2. **`scripts/v40-signoff-gate.sh`**
   - Copy structure from `v32-signoff-gate.sh`
   - Keep: scope-freeze, openapi, Doctor ALL/M4, OIDC|AuthSession|ClientExchange, oidc-console-smoke, Space/login vitest, rag-*/sandbox/skill/rag-lsp/remote-sandbox
   - Add: `device-session-smoke`; FE `consoleGate.test.ts` (with SpacePage/loginHash as appropriate)
   - Require: `v4.0-release-scope` contains `已冻结`; checklist + signatures template present; evidence files for oidc-console + device-session (+ remote-sandbox as in v3.2)
3. **`make v4.0-signoff`** → script; add to `.PHONY`
4. **`scripts/scope-freeze-gate.sh`**: `check_scope` for `v4.0-release-scope.md`
5. **Docs**
   - `doc/checklists/v4.0-signoff.md`
   - `doc/evidence/v4.0-signatures-template.md`
   - `doc/plan/sprint-dx66-v40-signoff.md`
   - CHANGELOG / TODO / plan README
6. Run `make v4.0-signoff` once in implementation (or document if env blocks sandbox); fix failures

## Tag (human only)

```bash
git tag -a v4.0.0 -m "ASH v4.0.0 Auth harden RS256 refresh revoke console gate (DX61–DX66 frozen)"
git push origin v4.0.0
```

Gate must print that it does **not** create the tag.

## Verification

```bash
make scope-freeze-gate
make v4.0-signoff
```
