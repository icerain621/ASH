# R-08 跨 space / RLS 抽测清单

> 发布窗口越权抽测（P1-5）。自动化入口：`make r08-cross-space-gate`。

## 自动化覆盖

| 层 | 命令 / 测试 | 说明 |
|----|-------------|------|
| API 隔离 | `TestCrossSpaceAPIRegression` | 高风险读/写：runs 控制面、SSE、产物、审批、secret、memory、CI、audit、plugin、RAG、feedback、retention、space 成员/矩阵 |
| API 单点 | `TestGetRunRejectsCrossSpace` 等 | 早期回归用例 |
| H-09 | `TestReleaseSamplingH09CrossSpaceMemoryDenied` | Memory query 不泄漏他 space |
| RLS 静态 | `TestMigrationCatalog_RLSCoverage` 等 | 策略目录与 SQL 校验 |
| RLS live | `postgres-rls-e2e`（Docker 或 `ASH_DATABASE_URL`） | 可选；门禁在无环境时 SKIP |

```bash
make r08-cross-space-gate
# 强制跳过 Postgres live：
ASH_R08_SKIP_POSTGRES=1 make r08-cross-space-gate
```

## 发布窗口人工抽测（有环境时）

| # | 步骤 | 期望 |
|---|------|------|
| 1 | Space A token 访问 Space B run/stream/artifacts | `403 SPACE_ACCESS_DENIED` |
| 2 | Space A 创建 `spaceId=B` 的 secret / release / repo | `403` |
| 3 | Memory query 不得返回 B 的候选 | 空或仅 A |
| 4 |（Postgres）`ash_app` 会话不可见他 space 行 | 0 泄漏行 |

证据可附在 `doc/evidence/release-window-*.md`「切换检查」节。

## 状态

- **代码 + 门禁**：Sprint CP 关闭「发布窗口抽测」自动化缺口  
- **云 RDS live RLS**：仍随 H-01～H-03 云验收一并签字  
