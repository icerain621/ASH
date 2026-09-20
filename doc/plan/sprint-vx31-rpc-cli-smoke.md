# Sprint VX31 — RPC CLI 烟测 / session rpc smoke

> **前置 / Prerequisite：** v6.0 已冻结；[`v6.3-release-scope.md`](v6.3-release-scope.md)  
> **状态 / Status：** ✅  
> **做法 / Approach：** 固化已有 `ash session rpc`（`internal/session.ServeRPC`）：最小 `session.start` → EOF 烟测 + `make session-rpc-smoke`。无新表。  
> Harden existing `ash session rpc`: minimal session.start→EOF smoke + `make session-rpc-smoke`. No new table.

## 任务板 / Board

| ID | 任务 / Task | 状态 |
|----|-------------|------|
| VX31-1 | CLI/包测：空闲 `session.start` 回 `session.started` | ✅ |
| VX31-2 | `make session-rpc-smoke` 写 evidence | ✅ |
| VX31-3 | CHANGELOG 一句 | ✅ |

## 验收 / Verify

```bash
go test ./internal/session/ -count=1 -run TestServeRPCSessionStartIdle
make session-rpc-smoke
# optional: ASH_SESSION_RPC_CLI=1 make session-rpc-smoke
```
