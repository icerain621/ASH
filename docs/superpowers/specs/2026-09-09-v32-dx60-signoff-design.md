# v3.2 DX60 — 范围冻结 + 签字门禁

> Status: **done** (2026-09-09)  
> Program: T3 IdP/Session · Approach: mirror DX54 / v3.1-signoff · **no new tables** · **no auto tag**

## Goals

- Freeze `doc/plan/v3.2-release-scope.md`
- `make v3.2-signoff` + `scope-freeze-gate` includes v3.2
- Checklist + signatures template; tag `v3.2.0` remains manual

## Gate contents

- scope-freeze · openapi · Doctor ALL/M4
- OIDC / AuthSession tests · `oidc-console-smoke`
- Baseline: rag-hybrid · rag-vector · sandbox · skill-pack · rag-lsp · remote-sandbox

## Non-goals

- Auto `git tag`
- Doctor count bumps / new tables
