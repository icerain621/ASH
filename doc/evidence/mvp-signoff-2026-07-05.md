# MVP 发布签字证据（2026-07-05）

- 生成时间（UTC）：2026-07-05T14:09:41Z
- Git：a56fc1b @ main
- 自动化步骤通过：**16**
- 证据目录：`/c/Go_Work/src/ash/.ash/evidence/mvp-signoff-20260705T135953Z`（本地，未入库）
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
| H-01 云 RDS | ⏸ 需 ASH_DATABASE_URL |
| H-02/H-03 ash_app | ⏸ 需 Docker 或云 RDS |

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

见 [`11-mvp-release-checklist.md`](../progress/mvp-release-checklist.md) §11 与 [`h01-h03-cloud-signoff.md`](../checklists/h01-h03-cloud-signoff.md)。

| 角色 | 姓名 | 日期 |
|------|------|------|
| 产品负责人 | 产品负责人（占位） | 2026-07-06 |
| 技术负责人 | 技术负责人（占位） | 2026-07-06 |
| 测试负责人 | 测试负责人（占位） | 2026-07-06 |
| 发布负责人 | 发布负责人（占位） | 2026-07-06 |
