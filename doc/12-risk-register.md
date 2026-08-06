# ASH 项目风险台账（Risk Register）

## 1. 使用说明
- 本台账用于项目周会、里程碑评审、上线前评估。
- 风险等级定义：
  - 高（H）：可能导致里程碑延期或上线失败
  - 中（M）：影响部分交付质量或效率
  - 低（L）：可通过日常管理消化
- 状态定义：`Open`（未关闭）、`Mitigating`（处理中）、`Closed`（已关闭）

## 2. 风险清单（MVP 阶段）

| ID | 风险项 | 类型 | 等级 | 触发条件 | 预警指标 | 应对策略 | 责任角色 | 状态 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| R-01 | 需求频繁变更导致排期失控 | 管理 | H | 迭代中新增/变更 P0 > 2 次 | 迭代内变更单数量 | 需求冻结；新增需求进入下迭代；变更走评审 | 产品负责人 | Open |
| R-02 | Agent 结果不稳定影响信任 | 产品/算法 | H | 同场景输出波动明显 | 任务成功率下降、低分反馈上升 | 增加 Reviewer 校验；模板化 prompt；失败案例回流 | 算法负责人 | Open |
| R-03 | CI 诊断准确率不足 | 技术 | H | 根因定位偏差导致修复无效 | 诊断采纳率低、重复失败率高 | 高频模式已扩（cancel/OOM/frontend，Sprint BQ）；BJ 用真实 GitHub log 校准采纳率 | 后端负责人 | Mitigating |
| R-04 | Memory 记忆污染 | 产品/数据 | M | 错误信息被高置信写入 | 命中后负反馈升高 | TTL/sweep 已有；低分 feedback 衰减 + 检索置信度门槛（Sprint BR）；高置信额外确认待产品 | 产品+算法 | Mitigating |
| R-05 | 任务状态机异常流转 | 技术 | H | 出现非法状态跳转 | 状态异常告警、人工修复增加 | 状态表 + mid-loop Cancel + Replay/Approve/Resume 门禁（BT–BW） | 后端负责人 | Mitigating |
| R-06 | 队列积压导致任务延迟 | 稳定性 | M | 峰值任务超出 worker 消费能力 | 队列积压时长、消费速率下降 | TTL queue-gate；Scale inflight；告警 `run_inflight_count` + Observability 治理面板（Sprint BY/CA/CE）；分级扩容仍待运维 | 后端/运维 | Mitigating |
| R-07 | SSE 不稳定影响体验 | 前端/稳定性 | M | 日志流频繁中断 | SSE 断连率、前端报错率 | 自动重连 + Last-Event-ID + timeline 轮询回退（Sprint BM/BN）；生产观察断连率 | 前端负责人 | Mitigating |
| R-08 | 权限策略缺陷导致越权 | 安全 | H | 非授权用户访问敏感接口 | 401/403 异常分布、审计告警 | RBAC + `TestCrossSpaceAPIRegression`（控制面 + secret/memory/CI/audit/plugin/access，Sprint BM/CC/CE）；CI diagnosis 跨 space 403；发布窗口抽测 | 后端负责人 | Mitigating |
| R-09 | 关键依赖服务不稳定 | 外部依赖 | M | Repo/CI API 波动或限流 | 第三方错误率上升 | GitHub 429/5xx 退避重试 + 连续失败熔断；API `CI_PROVIDER_UNAVAILABLE`（Sprint BX） | 平台负责人 | Mitigating |
| R-10 | 文档与实现不一致 | 协作 | M | 接口变更未同步文档 | 联调阻塞次数 | OpenAPI 控制面 409 / Scale backlog 字段已同步（Sprint BY）；变更继续走 openapi-check | 全体模块负责人 | Mitigating |
| R-11 | 人员并行不足 | 资源 | M | 单点人员阻塞关键链路 | PR 积压、任务延期 | 明确备份人；关键模块双人覆盖 | 项目经理 | Open |
| R-12 | 上线回滚预案不完整 | 发布 | H | 发布异常后无法快速回退 | 回滚演练失败/超时 | 发布窗口强制 `rollback-drill` + SLA；备份 sha256/`integrity_check`；`EvaluateGate` 纳入 drill（`ASH_REQUIRE_ROLLBACK_DRILL` 可硬阻断）；触发准则写入证据（Sprint BZ） | 发布负责人 | Mitigating |

## 3. Top 5 高优先风险（当前关注）
1. R-01 需求变更失控
2. R-02 Agent 结果稳定性
3. R-03 CI 诊断准确率
4. R-05 状态机异常流转
5. R-12 回滚预案不足

## 4. 风险评审节奏建议
- 周会：更新风险状态与新增风险
- 里程碑评审：重点检查高风险项缓解进度
- 上线前：逐条确认 H 级风险是否关闭或可接受

## 5. 风险升级机制
- 当 H 级风险连续 3 天无缓解动作 -> 自动升级到项目负责人
- 当任一风险触发上线阻断条件 -> 进入发布冻结
- 需冻结发布的典型条件：
  - 严重安全风险未修复
  - 回滚方案不可执行
  - 核心链路成功率低于预设阈值
