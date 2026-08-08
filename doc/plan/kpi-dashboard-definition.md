# ASH KPI 看板指标口径定义（BI 对接版）

> 归属：[`plan/`](README.md)

## 1. 文档目标
- 统一 ASH 平台核心指标口径，避免多版本解释冲突。
- 支持产品、研发、测试、管理层在同一口径下评估 MVP 成效。
- 作为数据埋点、BI 建模、运营复盘的基础协议文档。

## 2. 指标分层
- 业务价值层：衡量效率与质量改进
- 产品使用层：衡量功能使用和留存
- AI 能力层：衡量建议质量与自动化效果
- 工程稳定层：衡量性能、可用性、失败率

## 3. 指标定义总表

| 指标ID | 指标名称 | 定义 | 计算公式 | 统计周期 | 负责人 |
| --- | --- | --- | --- | --- | --- |
| KPI-01 | 任务成功率 | 完成且状态为成功的任务占比 | `completed_success / started_tasks` | 日/周 | 后端负责人 |
| KPI-02 | 平均任务时长 | 任务从启动到完成的平均耗时 | `sum(task_duration) / completed_tasks` | 日/周 | 后端负责人 |
| KPI-03 | AI 建议采纳率 | 被采纳建议占总建议数比例 | `accepted_suggestions / total_suggestions` | 周/月 | 产品负责人 |
| KPI-04 | CI 一次通过率 | 首次触发即通过的 CI 占比 | `first_pass_ci / total_ci_runs` | 日/周 | 测试负责人 |
| KPI-05 | CI 诊断采纳率 | 诊断建议被执行的比例 | `adopted_diagnosis / total_diagnosis` | 周/月 | 后端负责人 |
| KPI-06 | 低分反馈率 | 评分 <=2 的反馈占比 | `low_score_feedback / total_feedback` | 日/周 | 产品负责人 |
| KPI-07 | Memory 命中率 | 执行中命中记忆次数占检索次数比 | `memory_hits / memory_queries`（`run_events` 经 `runs.space_id` 过滤） | 日/周 | 算法负责人 |
| KPI-08 | SSE 稳定率 | SSE 正常会话占比 | `session_closed / (session_closed + session_failed)` | 日/周 | 前端负责人 |
| KPI-09 | API 错误率 | 非 2xx 请求占比 | `error_requests / total_requests` | 日/周 | 后端负责人 |
| KPI-10 | 队列积压时长 | 队列任务平均等待时长 | `sum(queue_wait_ms)/queue_task_count`（`run_steps` 经 `runs.space_id` 过滤） | 日/周 | 运维负责人 |
| KPI-11 | 场景稳定率 | 同场景可重复成功率达标占比（R-02） | `stable_scenarios / eligible_scenarios`（样本≥2 且成功率≥85%） | 日/周 | 算法/产品 |

breakdown：`scenarioStability` — 按 `scenario_name@scenario_version` 输出成功率 / 样本数 / 关联低分反馈（`target_type=run`）。

## 4. 关键指标详细口径

## 4.1 KPI-01 任务成功率
- 定义：状态最终为 `COMPLETED` 且执行无阻断错误的任务占比
- 分子：`tasks.status = COMPLETED`
- 分母：`tasks.status IN (RUNNING, NEED_APPROVAL, COMPLETED, FAILED, CANCELED)` 且已启动
- 排除条件：测试环境、压测任务、手工造数任务

## 4.2 KPI-03 AI 建议采纳率
- 定义：Copilot 或 Agent 输出建议中被用户明确采纳的比例
- 分子事件：`suggestion_accepted`
- 分母事件：`suggestion_presented`
- 口径说明：同一任务中重复展示相同建议按 1 次计数

## 4.3 KPI-04 CI 一次通过率
- 定义：首次 CI 运行结果为通过的比例
- 分子：`ci_runs.attempt_no = 1 AND status = success`
- 分母：`ci_runs.attempt_no = 1`
- 注意：仅统计主流程相关 CI，排除手工重跑调试流水线

## 4.4 KPI-07 Memory 命中率
- 定义：记忆检索请求中返回可用记忆且被任务引用的比例
- 分子事件：`memory_hit` 且 `used_in_step = true`
- 分母事件：`memory_query`
- 风险：高命中不等于高质量，需结合低分反馈率一起观察

## 4.5 KPI-08 SSE 稳定率
- 定义：终态 SSE 会话中正常关闭的占比（R-07）
- 成功：审计 `stream.session_closed`
- 失败：审计 `stream.session_failed`（握手失败、首刷失败、轮询中途失败）
- 分母：`closed + failed`（仅终态；仅有 `session_opened` 的在途会话不计）
- 采集：Worker `GET /runs/{id}/stream`；中途 `ListAfter` 失败会记 `session_failed` 且不再记 `closed`
- 空窗：无终态事件 → `unavailable` / 仅在途 → `empty`

## 4.6 KPI-11 场景稳定率（R-02）
- 定义：窗口内可重复场景中，成功率达到门槛的场景占比
- 分组：`runs.scenario_name` + `runs.scenario_version`
- 可重复门槛：同场景 `started ≥ 2`
- 稳定门槛：`finished / started ≥ 0.85`（`finished` = `status == finished`，与 KPI-01 一致）
- 分子：达到稳定门槛的场景数；分母：达到可重复门槛的场景数
- 空窗：无场景达到样本门槛时 KPI-11 = `empty`（不伪造 100%）
- 联动：breakdown `scenarioStability` + KPI-06 低分反馈（按 run 归属场景）
- 不含：产物 digest / 文本语义方差（后续算法轨）

## 5. 数据源映射

| 指标 | 主数据源 | 辅助数据源 |
| --- | --- | --- |
| 任务类（KPI-01/02） | `tasks`, `agent_runs`, `task_steps` | `tool_calls` |
| AI 采纳类（KPI-03/06） | `feedback_records` | 前端埋点事件 |
| CI 类（KPI-04/05） | `ci_events`（如后续建立）/ `tool_calls` | 外部 CI API |
| 记忆类（KPI-07） | `memories`, `memory_events` | `task_steps` |
| 稳定性类（KPI-08/09/10） | 网关日志、SSE 日志、队列监控 | OTel 指标 |
| 场景可重复（KPI-11 / R-02） | `runs`（scenario + status） | `feedback`（run 低分） |

## 6. 刷新频率与时区
- 实时指标（接口错误率、队列积压）：1 分钟刷新
- 运营指标（采纳率、低分反馈率）：1 小时刷新
- 管理指标（周报/月报）：每日凌晨批处理
- 时区统一：`Asia/Shanghai (UTC+8)`

## 7. 维度规范
- 必选维度：`date`, `projectId`, `spaceId`, `env`
- 推荐维度：`userRole`, `taskType`, `model`, `toolName`
- 维度缺失处理：统一使用 `UNKNOWN`，禁止空字符串

## 8. 埋点事件规范（建议）

| 事件名 | 触发时机 | 关键字段 |
| --- | --- | --- |
| `task_created` | 创建任务成功 | `taskId`, `projectId`, `mode`, `creatorId` |
| `task_started` | 任务进入执行 | `taskId`, `runId`, `riskLevel` |
| `agent_step_completed` | 步骤执行结束 | `runId`, `stepNo`, `stepType`, `status`, `durationMs` |
| `suggestion_presented` | 建议展示给用户 | `targetType`, `targetId`, `source` |
| `suggestion_accepted` | 用户采纳建议 | `targetType`, `targetId`, `userId` |
| `feedback_submitted` | 提交反馈 | `targetType`, `rating`, `category` |
| `memory_query` | 发起记忆检索 | `scope`, `topK`, `taskId` |
| `memory_hit` | 命中并使用记忆 | `memoryId`, `taskId`, `runId` |

## 9. 指标质量保障
- 指标上线前必须完成：
  - [ ] SQL 口径评审（BI / DBA 人工）
  - [x] 埋点字段完整性验证（`TestOverviewAggregatesKPIInputs` + `TestKPIOverviewSummaryCatalog`）
  - [x] 样本回放对账（`make kpi-reconcile-gate`：overview ↔ derive replay）
  - [ ] 与业务侧手工统计偏差 < 5%（生产 T+1 观察）

## 10. 指标预警阈值（MVP 建议）
- 任务成功率 < 85%：黄色预警，< 75% 红色预警
- AI 采纳率连续 3 天下降：黄色预警
- API 错误率 > 1%：黄色预警，> 3% 红色预警
- 队列平均等待 > 30s：黄色预警，> 60s 红色预警
- 低分反馈率 > 20%：黄色预警，> 30% 红色预警

## 11. 看板建议布局
- 第一屏（经营总览）：任务成功率、采纳率、CI 一次通过率、错误率
- 第二屏（AI 能力）：建议采纳漏斗、记忆命中率、低分反馈分布
- 第三屏（工程稳定）：接口 SLA、队列积压、SSE 稳定率
- 第四屏（分组对比）：按空间/项目/模型版本对比

## 12. 版本管理
- 当前版本：v1.0
- 生效日期：`2026-06-01`
- 变更原则：所有口径调整必须先评审，再版本化发布
