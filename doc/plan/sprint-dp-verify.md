# Sprint DP：verify 步骤（方案 C）Implementation Plan

> **方案：** 已批准 **C** = DSL `kind: verify` + 执行/重试 + 失败自动 improve 草稿  
> **Goal:** 场景可声明验证步骤；失败可重试；仍失败则阻塞 Run 并创建 improve proposal（draft）。  
> **状态：** ✅ 完成（方案 C）· **无新表**

## 任务板

| ID | 任务 | 状态 |
|----|------|------|
| DP-1 | Schema/types/validate：`kind: verify` + `verify.checks` | ✅ |
| DP-2 | execute 执行 checks + step.retry | ✅ |
| DP-3 | 失败 → improve draft + 事件 | ✅ |
| DP-4 | 样例场景步骤 + 单测 | ✅ |
| DP-5 | docs / openapi 说明（DSL） | ✅ |

## DSL

```yaml
- id: qa.verify
  role: QA
  kind: verify
  verify:
    checks:
      - { tool: test.run, args: { scope: changed }, timeoutMs: 120000 }
    onFail: improve   # fail|improve（默认 fail；improve 仍 failRun）
  retry:
    maxAttempts: 2
    backoffMs: 200
```

## 行为

- 每个 check 走 ToolBus（尊重 sub-run allowlist / 危险工具策略）
- `retry.maxAttempts` 作用于整个 verify 步骤（一轮内所有 checks）
- 最终失败：`VERIFY_FAILED`；若 `onFail=improve` → 创建 improve draft + `improve.draft_created`

## 退出标准

- [x] feature_delivery 含 verify 步骤且 schema 绿
- [x] 重试与 improve 草稿单测绿
- [x] docs/CHANGELOG 更新
