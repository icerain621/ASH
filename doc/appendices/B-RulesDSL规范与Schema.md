# 附录 B：Rules / Scenario DSL 规范与 Schema（v0.1）

> 本附录定义 `ash.rules/v0.1`：用于描述场景模板（Scenario）、步骤（Steps）、门禁（Gates）、钩子（Hooks）与策略档（policyProfile）。
>
> 冻结等级：**M0 冻结**（v0.1）。语义变更必须通过 `scenarioVersion` 升级，不允许原地破坏兼容。

## 1. 设计目标
- 将交付流程“模板化”：Feature/Hotfix/SecPatch 可版本化复用。
- 将门禁“显式化”：阻断/降级/补救（remediation）可配置。
- 将安全“默认化”：危险工具默认 deny；通过 policyProfile 控制放行。
- 将回放“可行化”：DSL 与 run_events 共同构成可回放输入的一部分（版本锚定）。

## 2. 顶层结构（YAML）
```yaml
version: "ash.rules/v0.1"

scenario:
  name: "feature_delivery"
  scenarioVersion: "1.0.0"
  description: "Feature 从需求到交付的标准流程"
  policyProfile: "default"  # default|strict|hotfix|security
  checkpoint:
    strategy: "per_step"    # per_step|per_tool|manual
    retain: 50

  roles:
    PM:        { maxParallel: 1 }
    Architect: { maxParallel: 1 }
    Coder:     { maxParallel: 2 }
    Reviewer:  { maxParallel: 1 }
    QA:        { maxParallel: 1 }
    Shipper:   { maxParallel: 1 }

  inputs:
    required: [issueOrSpec, repoRoot]
    optional: [constraints, targetBranch]

  artifacts:
    required:
      - { type: "diff" }
      - { type: "test_report" }
      - { type: "release_notes" }
      - { type: "rollback_plan" }

  gates:
    - id: "gate.repo.clean"
      when: "before.step.Coder"
      blocking: true
      check:
        tool: "git.status"
        expect: { clean: true }
      onFail:
        message: "工作区不干净，请提交或 stash"
        remediation:
          - { tool: "git.diff" }
          - { tool: "git.stash" }

  steps:
    - id: "pm.clarify"
      role: "PM"
      kind: "llm"
      promptRef: "prompts/pm_clarify.md"

    - id: "code.implement"
      role: "Coder"
      kind: "tool_chain"
      chain:
        - { tool: "git.checkout", args: { newBranchFrom: "${inputs.targetBranch:-main}" } }
        - { tool: "apply_patch", args: { from: "${artifacts.design_adr}" } }
        - { tool: "test.run", args: { scope: "changed" } }

hooks:
  - id: "hook.pre_tool"
    on: "tool.called"
    policy: "enforce" # enforce|observe
    rules:
      - match: { tool: "shell.exec", risk: "danger" }
        action: { deny: true, reason: "危险命令需人工批准或提升策略档" }
```

## 3. 关键语义（评审冻结点）
### 3.1 版本与兼容
- **`version`** 固定为 `ash.rules/v0.1`（DSL 语法版本）。
- **`scenario.scenarioVersion`** 为场景模板版本：**语义变更必须升版本**。

### 3.2 Steps 执行器（kind）
- `llm`：模型输出（建议支持 `requireCitations` 约束）。
- `tool_chain`：工具链（tool 调用必须有 schema、超时、审计）。
- `human`：人工确认/审批（用于提升权限、发版放行等）。

### 3.3 Gates（门禁）
- `blocking=true`：不满足则 Run 阻断（产生 `policy.denied` 或 gate 事件）。
- `check` 支持两类：
  - `tool`：调用工具并断言输出
  - `artifact`：对产物断言（例如 test_report.pass）
- `onFail.remediation`：补救链路（可执行并记录审计）。

### 3.4 Hooks（钩子）
- `on` 必须来自允许的事件集合（见附录 A 事件类型表）。
- `policy=enforce`：可直接 deny 阻断；`observe`：只记录不阻断。

## 4. JSON Schema（实现要求）
> v0.1 必须提供 JSON Schema 文件并在运行时校验。此处列出“必须约束”。

### 4.1 必须约束
- `scenario.name + scenario.scenarioVersion` 组成不可变标识。
- `steps[].id` 全局唯一；`when`/`before.step.X` 必须引用存在的 step 或 role。
- `gates[].check` 必须二选一：`tool` 或 `artifact`。
- `hooks[].on` 必须是白名单事件（减少注入面）。
- `${...}` 模板变量必须在运行时解析失败时阻断（避免静默错误）。

**TODO（负责人：后端）**：生成并固化 `ash.rules/v0.1` 的完整 JSON Schema。  
**验收方式**：以 20 个非法 DSL 样例回归，错误定位准确。

## 5. 示例：Feature/Hotfix/SecPatch 的最小差异（指导）
- **Feature**：强调设计/评审/回归；产物四件套 + ADR 可选。
- **Hotfix**：policyProfile=strict；必须 rollback；danger 工具需 human step。
- **SecPatch**：必须包含 scan 证据引用与影响面分析；引入引用门禁。

