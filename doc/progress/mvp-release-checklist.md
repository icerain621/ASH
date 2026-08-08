# ASH MVP 发布检查清单（可勾选）

> **归属**：[`progress/`](README.md)  
> **自动化证据**：`make mvp-signoff` → [`../evidence/mvp-signoff-latest.md`](../evidence/mvp-signoff-latest.md)  
> **发布范围**：[`../plan/mvp-release-scope.md`](../plan/mvp-release-scope.md)  
> **云 RDS 验收**：`make cloud-acceptance` → [`../checklists/h01-h03-cloud-signoff.md`](../checklists/h01-h03-cloud-signoff.md)

## 1. 需求与范围确认
- [x] 本次发布目标、范围、不包含范围已确认并冻结（[`mvp-release-scope.md`](../plan/mvp-release-scope.md)；`make signoff-gate`）
- [x] 需求变更已冻结（仅允许 P0 缺陷修复）— 规则见 [`mvp-release-scope.md`](../plan/mvp-release-scope.md) §4；`make scope-freeze-gate`
- [x] PRD、API、数据库文档已同步至最新版本（Doctor ALL 43/43 + OpenAPI 契约门禁）

## 2. 开发完成度
- [x] P0 功能全部开发完成（M0–M3 + PRD MVP 控制台/API）
- [x] 前后端联调完成，主链路可用（Runs/SSE/Memory/CI/Scale）
- [x] 异常路径（失败/重试/取消）已验证（单测 + derive replay）
- [x] 高风险动作审批流程可用（M2 policy + 场景 enforcement）

## 3. 测试与质量
- [x] 后端单元测试通过（`go test ./...` / CI）
- [x] 快捷回归通过（`make regression-short`）— 证据见 `doc/evidence/`
- [x] 发布审计静态通过（`ASH_RELEASE_AUDIT_SKIP_OPENAPI=1 make release-window-audit`）
- [x] 前端 lint 与核心测试通过（`make web-gate`：eslint + vitest + build）
- [x] 集成测试通过（任务创建 -> 启动 -> 执行 -> 输出；API + CI fixture）
- [x] 回归测试通过（Doctor ALL + regression-short）
- [x] 严重缺陷（P0/P1）为 0（当前迭代）

## 4. 性能与稳定性
- [x] 关键接口响应满足基线要求（`TestHealthEndpointsLatencyBaseline` P95 ≤250ms）
- [x] 任务并发执行压测达标（`TestConcurrentRunsListBaseline` 12 并发 /runs）
- [x] SSE 连接稳定、断线可重连（`TestReleaseSamplingSSE` + TR0 续传）
- [x] 队列积压与消费速率在可控范围（`make queue-gate`：TTL sweep 消费 + 洁净空间告警基线）

## 5. 安全与权限
- [x] JWT、RBAC 配置正确（M2 权限矩阵 + Doctor M2-01）
- [x] 高风险接口权限校验通过（跨 space 403 + policy.denied）
- [x] 审计日志记录完整（compliance export + RedactJSON）
- [ ] Postgres 生产切换已完成 [`postgres-rds-e2e.md`](../checklists/postgres-rds-e2e.md) + [`h01-h03-cloud-signoff.md`](../checklists/h01-h03-cloud-signoff.md)（**待云 RDS 签字**）
- [x] H-04–H-09 烟测清单已勾选（静态自动化；live 本地 `make worker-local-gate`；云 Worker 见 `make live-smoke`）— [`smoke-index.md`](../checklists/smoke-index.md)
- [x] 默认密钥/测试账号已移除或替换（`make production-config-gate` 拦截 dev-secret / CHANGE_ME）

## 6. 数据与迁移
- [ ] 数据库迁移脚本在 staging 成功执行（**云 RDS 待验收**；本地 Docker `make postgres-local-rds-e2e` ✅ Sprint CW）
- [x] 新增表、索引、约束验证通过（SQL rev 20 + Doctor M3-03/08/11）
- [x] 关键数据已备份（`make data-backup` / `make release-window-gate` 本地演练；生产 migrate 前再次执行）
- [x] 回滚脚本（DDL/DML）已准备并演练（[`postgres-rds-e2e.md`](../checklists/postgres-rds-e2e.md) §8）

## 7. 可观测与告警
- [x] 日志字段包含 `traceId/taskId/runId/userId`（审计事件 + OTel）
- [x] 核心指标已接入（成功率、时延、失败率、队列积压；`/metrics` + derive）
- [x] 告警规则已配置并测试（治理告警 + `ASH_ALERTS_EVAL_INTERVAL`）
- [x] 仪表盘可用于上线后实时观察（`/ui/observability` + `/ui/metrics`）

## 8. 发布准备
- [x] 发布窗口自动化门禁（`make release-window-gate` → [`release-window-latest.md`](../evidence/release-window-latest.md)）
- [x] 发布窗口时间与值班人员已确认（[`release-window-runbook.md`](../checklists/release-window-runbook.md) roster 占位 dry-run；`make signoff-apply` 同步）
- [x] 版本号与变更说明（Changelog）已生成（`CHANGELOG.md`）
- [x] 配置文件与环境变量已核对（`make config-env-gate` + [`cloud-rds.env.example`](../../config/cloud-rds.env.example)）
- [x] API 文档与使用说明已同步（`make swagger` + openapi-check）

## 9. 灰度与回滚
- [x] 灰度策略已配置（`/ui/releases` 只记录策略与证据）
- [x] 灰度观察指标与阈值已定义（release gate + KPI overview）
- [x] 回滚触发条件明确（[`postgres-rds-e2e.md`](../checklists/postgres-rds-e2e.md) §8）
- [x] 回滚流程演练完成并记录耗时（`make rollback-drill` → `doc/evidence/rollback-drill-latest.md`）

## 10. 上线后验证（T+0 / T+1）
- [x] T+0：`make release-sampling-static` 冒烟通过（静态；live 见 `make live-smoke`）
- [x] T+0：无严重告警，错误率在阈值内（`make t0-alert-gate` 洁净库基线）
- [x] T+1：关键指标达标（采纳率/成功率/时延）（`make t1-metrics-gate` API 基线）
- [x] T+1：收集首批用户反馈并形成修复计划（`TestT1FeedbackIngestBaseline` + `/ui/feedback`）

## 11. 发布签字（建议）

> 自动化门禁通过后，填写 `config/signoff.env` 并执行 `make signoff-apply`（流程见 [`mvp-signoff-roster.md`](../checklists/mvp-signoff-roster.md)）。验收：`make signoff-gate`。

- 产品负责人：`产品负责人（占位）`  日期：`2026-07-07`
- 技术负责人：`技术负责人（占位）`  日期：`2026-07-07`
- 测试负责人：`测试负责人（占位）`  日期：`2026-07-07`
- 发布负责人：`发布负责人（占位）`  日期：`2026-07-07`
