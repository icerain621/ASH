# Sprint DX27 — Sandbox e2e 证据（v2.7）

> 状态：**完成**（2026-09-06）  
> 范围：[`v2.7-release-scope.md`](v2.7-release-scope.md) · 清单 [`../checklists/sandbox-landlock-e2e.md`](../checklists/sandbox-landlock-e2e.md)

## 交付

| # | 项 | 状态 |
|---|-----|------|
| DX27-1 | Linux `TestE2E*`：repo 可读 / 外路径 deny / `SYS_MOUNT` seccomp | ✅ |
| DX27-2 | `ASH_SANDBOX_E2E=1` → `sandbox-smoke` 证据写入 `doc/evidence/` | ✅ |
| DX27-3 | `sandbox.Executor` 扩展注释（**无 E2B 客户端**） | ✅ |
| DX27-4 | checklist + sprint / CHANGELOG / TODO | ✅ |

## 验收

```bash
ASH_SKIP_SANDBOX=1 make sandbox-smoke
# Linux CI:
ASH_SANDBOX_E2E=1 ASH_SKIP_SANDBOX=1 make sandbox-smoke
```

## 备注

- 非 Linux：e2e **Skip**，证据标注 skipped；门禁仍绿。
- 夹具放 `$HOME/.ash-sandbox-e2e/`（避开 `/tmp` Landlock 白名单）。
- **无新表** / Doctor 水位不变（M4-SBX-04/05 已有）。
