# ASH PRD（需求文档）v0.1

> 目标：用“交付闭环 + 可审计记忆 + 可回放”的方式，辅助人类完成软件交付与知识自学习。
>
> 文档状态：v0.2（对照 MVP `v0.1.0-mvp`，2026-08-08）。未决项以 **TODO** 标注；实现进度见 [`../plan/PLAN-进度与里程碑.md`](../plan/PLAN-进度与里程碑.md)。  
> 归属：[`design/`](README.md)

## 1. 背景与问题
- **现状痛点**：软件交付涉及跨角色协作（需求/设计/实现/测试/发布/运维），信息分散、上下文易丢、重复劳动多；AI 参与后常出现“输出不可追溯、不可复现、不可审计、容易越权”的落地障碍。
- **机会**：将 AI 从“对话式建议”升级为“**交付驱动的机器人**”：能产出可合并的变更、测试证据、发布包，并沉淀分层记忆，持续迭代。

## 2. 产品定位
- **定位一句话**：ASH 是一个辅助人类 coding 开发的机器人，支持 Web UI + CLI，按软件开发流程拆分角色协作，支持规则场景编排，并具备分层记忆评审与永久存储能力。
- **核心原则**
  - **交付闭环优先**：每次运行（Run）必须形成可验证交付物（Artifacts）。
  - **安全与审计优先**：工具调用最小权限、可阻断、可追责。
  - **可回放**：所有运行都能回放以复现问题与进行对照实验。
  - **可插拔**：LLM provider / MCP / skills / RAG / 向量库 / 可观测性插件均可替换。

## 3. 目标用户与角色（Persona）
- **开发者（Coder）**：希望加速实现与排错，但不希望破坏代码质量与流程。
- **技术负责人/架构师（Architect）**：希望方案可控、可审计、有清晰边界。
- **评审者（Reviewer）**：需要证据引用、门禁与一致的评审输出。
- **测试（QA）**：需要测试策略与失败归因能力。
- **发布（Shipper）**：需要变更说明、回滚方案、发布门禁。
- **SRE/运维（SRE）**：需要可观测性、稳定性、故障演练与降级策略。
- **安全（Security）**：需要越权防护、secret 管理、审计与合规控制。
- **知识管理员（Librarian）**：需要记忆候选评审、冲突治理、过期策略。

**已关闭（Sprint CO）**：谁付费/谁决策/谁审批与组织落地样板见 [`ORG-组织样板与商业落地.md`](ORG-组织样板与商业落地.md)；API `GET/POST /api/v1/org-templates…`。

## 4. 核心场景（Use Cases）
### 4.1 场景 A：Feature Delivery（P0）
- **输入**：需求/issue、repoRoot、约束（时限/质量门禁/依赖限制）
- **输出**：`diff`、`test_report`、`release_notes`、`rollback_plan`、（可选）`adr`
- **人机协作**：在门禁点（策略/危险操作/发版）要求 human step。

### 4.2 场景 B：Hotfix（P1）
- **输入**：故障描述、日志/监控、最近变更证据
- **输出**：最小修复 diff + 定向验证报告 + 回滚方案 + 发布窗建议

### 4.3 场景 C：Security Patch / Dependency Upgrade（P1）
- **输入**：漏洞证据/扫描报告、锁文件、兼容性文档
- **输出**：升级计划与风险矩阵 + 变更 + 回归证据 + 安全公告草稿

## 5. 需求范围（In / Out）
### 5.1 In Scope（v0.1→v1.0）
- Web UI：运行列表/详情、事件流、记忆评审、规则管理（最小）
- CLI：`ash run/replay/doctor`（最小）
- WorkflowGraph：按角色/步骤执行，支持 checkpoint 与 resume
- Rules/Scenario DSL：步骤/门禁/hooks/超时/必需产物
- ToolBus：git/apply_patch/test（最小工具集）
- ModelRouter：至少 2 个 provider（可降级）
- RAG：repo 级检索 + 引用规范（最小）
- Memory：分层、candidate→review→merge、SQLite 永久存储、schemaVersion
- Observability：事件流 + Prometheus 指标（插件化）

### 5.2 Out of Scope（M0 不做）
- 多租户/组织级权限中心（P2）
- OpenClaw 风格多端网关/设备身份（P3）
- Skill Marketplace（P3）
- 完整沙箱/容器隔离（P2 演进）

## 6. 功能需求（按优先级）
### 6.1 P0（MVP 必须）
- **交互**：CLI + Web（事件流 + artifacts 展示）
- **编排**：Feature scenario（PM→Architect→Coder→Reviewer→QA→Shipper）
- **工具**：git / apply_patch / test.run + policy deny
- **记忆**：L0/L1 candidate + 评审 + 合并 + evidence 强制
- **可观测**：run/step/tool/model/rag/policy/memory 事件；Prometheus 基础指标；`ash doctor(TR0)`

### 6.2 P1（重要）
- Hotfix/SecPatch scenario
- 引用缺失门禁（阻断/降级）
- 记忆去重/冲突/替代（edges）
- OTel 插件 + Waterfall
- 自我迭代 1.0（提案→回放对照→灰度→回滚）

### 6.3 P2（产品化/规模化）
- 多租户/团队隔离、组织级审批与合规
- 更强安全隔离（容器/沙箱）、secret 管理与扫描
- 存储演进（SQLite→Postgres）

### 6.4 P3（可选生态）
- 多端网关（role/scope/设备身份）
- 技能市场
- 知识图谱深度增强

## 7. 非功能需求（NFR）
- **安全**：最小权限、危险工具默认 deny、审计、脱敏、注入防护
- **可靠性**：checkpoint 恢复；外部依赖不可用时降级（向量库/CI/MCP）
- **性能**：核心路径可观测（P95 延迟、token/成本）
- **可回放**：run_events + artifacts digest + memory version 锚定

## 8. 数据与合规
**已实现（2026-08-08）**：分级表 + 默认保留期 + 样例脱敏 + 审计导出流程见 [`../appendices/J-数据分级与保留期.md`](../appendices/J-数据分级与保留期.md)；运行时 `GET /api/v1/data-policy` 与 events/artifacts retention apply。

## 9. 成功指标（Metrics / SLO）
- 交付成功率、CI 一次通过率、回滚演练成功率
- token/成本/任务、RAG 引用缺失率、工具失败率、恢复成功率
- 记忆评审 SLA、拒绝率、缺证据率、hit→used 比例

## 10. 验收标准（DoD）
- **M0**：TR0-01/02/03 全绿（由 `ash doctor` 产出报告）
- **M1**：TR1/TR2 子集全绿；自我迭代提案可跑通对照与灰度

