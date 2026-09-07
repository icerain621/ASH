# Sprint DX40 — remote-sandbox smoke + 证据（v2.9 · D5）

> **方案：** `make remote-sandbox-smoke`；无 key / 未开 live → skip pass；**不**新增 Doctor 用例（保持 ALL 57 / M4 10）  
> **Goal:** 清单 + `doc/evidence/remote-sandbox-smoke-latest.md`；sandbox-smoke 覆盖 remote 包  
> **状态：** ✅ 完成 · **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DX40-1 | `scripts/remote-sandbox-smoke.sh` + Makefile | ✅ |
| DX40-2 | 可选 `TestLiveE2B`（无 key skip） | ✅ |
| DX40-3 | checklist / evidence / smoke-index | ✅ |
| DX40-4 | sandbox-smoke 纳入 `./internal/sandbox/remote/` | ✅ |

## 验收

```bash
make remote-sandbox-smoke
# → doc/evidence/remote-sandbox-smoke-latest.md
```
