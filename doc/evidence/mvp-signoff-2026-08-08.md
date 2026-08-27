# MVP 发布签字证据（2026-08-08）

- 生成时间（UTC）：2026-08-08T16:34:10Z
- Git：ff35d5a @ main
- 自动化步骤通过：**18**
- 证据目录：`/c/Go_Work/src/ash/.ash/evidence/mvp-signoff-20260808T162431Z`（本地，未入库）
- 门禁命令：`make mvp-signoff`

## 自动化验收

| 门禁 | 状态 |
|------|------|
| regression-short | ✅ |
| web-gate (lint+test+build) | ❌ |
| production-config-gate | ✅ |
| config-env-gate | ✅ |
| queue-gate | ✅ |
| t0-alert-gate | ✅ |
| rollback-drill | ✅ |
| release-sampling-static (H-09) | ✅ |
| Doctor ALL 43/43 | ✅ |
| release-window-audit (H-08) | ✅ |
| H-01 云 RDS | ✅ 本地 Docker dry-run（postgres-local-rds-e2e） |
| H-02/H-03 ash_app | ✅ 本地 Docker |

## MVP 清单映射

| MVP § | 项 | 自动化证据 |
|-------|-----|------------|
| 3 | 快捷回归 | regression-short |
| 3 | 前端 lint/测试/构建 | web-gate |
| 3 | 发布审计静态 | release-window-audit |
| 5 | H-04~H-09 烟测 | release-sampling-static + regression-short |
| 5 | Postgres 生产切换 | H-01~H-03 云验收 |
| 10 | T+0 冒烟 | release-sampling-static |

## 待人工签字

见 [`mvp-signoff-roster.md`](../checklists/mvp-signoff-roster.md)（`make signoff-apply` / `make signoff-gate`）。云 RDS：[`h01-h03-cloud-signoff.md`](../checklists/h01-h03-cloud-signoff.md)。

签字记录：[`mvp-signatures-latest.md`](../evidence/mvp-signatures-latest.md) ✅

| 角色 | 姓名 | 日期 |
|------|------|------|
| 产品负责人 | | |
| 技术负责人 | | |
| 测试负责人 | | |
| 发布负责人 | | |
