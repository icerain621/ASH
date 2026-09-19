# Task EW18 Report — W1 v6.0 吸收签字

**Status:** ✅ Complete  
**Date:** 2026-09-19

## Summary

W1 板 EW11–EW18 按代码与门禁 honest 关闭：EW15–EW17 分别对应 `4e1d8e3`（ExecPolicy sandbox/Doctor）、`80fd034`（steer）、`daa02b2`（queue）；EW18 openapi-check、包测、IntentBar vitest 均绿。

## Code anchors (EW15–EW17)

| Sprint | Commit | Subject |
|--------|--------|---------|
| EW15 | `4e1d8e362170fd24b9ab076ac8188137de971534` | ExecPolicy 驱动沙箱地板与 Doctor 位 |
| EW16 | `80fd03477ce9fad2b488cb54cd1f3801a378dbec` | steer 运行中续写意图 |
| EW17 | `daa02b2784674e067f8c173e86e870758afcc0f5` | follow-up 队列意图与 Chat 投影 |

## EW18 checklist

| # | Item | Result |
|---|------|--------|
| EW18-1 | `make openapi-check` | ✅ OK（2026-09-19 复验） |
| EW18-1 | `go test ./internal/hooks/... ./internal/execpolicy/... ./internal/session/... -count=1` | ✅ pass |
| EW18-1 | `npx vitest run …/IntentBar.test.tsx` | ✅ 10/10 |
| EW18-2 | CHANGELOG Unreleased EW15–EW17 | ✅ 已有条目 |
| EW18-2 | `TODO.md` · `ash-feature-inventory.md` · sprint 板 | ✅ W1 行/板全 ✅ |
| EW18-3 | `sprint-ew-w1-v60-absorb.md` 全 ✅ | ✅ |

## Test output (EW18 复验)

```text
make openapi-check
openapi-check OK

go test ./internal/hooks/... ./internal/execpolicy/... ./internal/session/... -count=1
ok  	github.com/ash-repwiki/ash/internal/hooks	0.427s
ok  	github.com/ash-repwiki/ash/internal/execpolicy	0.451s
ok  	github.com/ash-repwiki/ash/internal/session	55.155s

cd frontend && npx vitest run src/modules/agent-session/components/IntentBar.test.tsx
Test Files  1 passed (1)
     Tests  10 passed (10)
```

## Commits

- `39b1442e4e0710dc1c0282f219cf2d1e224336e4` — `docs: W1 v6.0 吸收签字`（计划板 · inventory · TODO · session swagger）
- EW18 简报与「`docs: W1 v6.0 吸收签字`」同提交（`.superpowers/sdd/briefs/task-EW18-report.md`）

## Excluded from W1 signoff commit

- `doc/diagrams/archify/**`（未纳入）
- `doc/evidence/device-session-smoke-latest.md`（未纳入）
