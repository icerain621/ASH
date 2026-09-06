# v2.8 LSP slice design (DX31–DX36)

## Goal

Productize a **bounded LSP surface** on top of DX29 thin `documentSymbol`: workspace-scoped short-lived sessions, then hover / definition / references wired into RAG + console. No public `/lsp/*` product API, no E2B, no new tables by default.

## Confirmed (C1–C7)

See `doc/plan/v2.8-release-scope.md` §7.

## DX31 — Session pool

- Key: `abs(workspaceRoot) + "|" + serverBinary`
- One stdio process per key; serialize RPC with a mutex
- Open docs tracked with version bumps (`didOpen` / `didChange`)
- Idle reclaim via `ASH_RAG_LSP_IDLE_SEC` (default 30s); shutdown+exit+kill
- Escape hatch: `ASH_RAG_LSP_SESSION=0` → per-file one-shot (DX29)
- Rebuild injects `RepoRoot` as workspace root and closes the indexer when done

## Later sprints

- **DX32** hover + definition query helpers (internal) — done
- **DX33** bounded references + Hybrid/Knowledge wiring — done
- **DX34** console status panel — done
- **DX35** smoke + evidence — done
- **DX36** freeze + `make v2.8-signoff` — done

v2.8 is **frozen**; tag `v2.8.0` remains manual after human sign-off.

## Non-goals

Full IDE, multi-language beyond Go/TS/JS, Chroma/Milvus, marketplace, Active-Active.
