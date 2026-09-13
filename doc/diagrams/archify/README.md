# ASH Archify 架构图

基于 [archify](https://github.com/tt-a1i/archify) skill 生成的可交互 HTML 架构图。

## 产物

| 文件 | 类型 | 说明 |
|------|------|------|
| `ash-system.html` | architecture | ASH 单体系统组件与数据流 |
| `ash-v2-dual-core.html` | architecture | v2 双核心（Agent Core × Memory Core + 治理/演进平面） |
| `ash-package-map.html` | architecture | Go 包边界（对齐 HLD-双核心-v2 §2.3） |
| `ash-deploy-local.html` | architecture | **部署** Stage 0：本地单机 / SQLite |
| `ash-deploy-stage1.html` | architecture | **部署** Stage 1：Worker + Sandbox（v2 主路径） |
| `ash-deploy-stage2.html` | architecture | **部署** Stage 2：控制/数据/执行分离（P2 设计目标） |
| `ash-run-execution.html` | sequence | Run 执行时序（POST /runs → ToolBus → SSE） |
| `ash-sandbox-fallback.html` | sequence | Sandbox 路由降级（remote → landlock → docker → process） |
| `ash-run-lifecycle.html` | lifecycle | Run 状态机（对齐 `internal/runs/status.go`） |
| `ash-goal-plan-lifecycle.html` | lifecycle | GoalPlan 状态机（draft → started / rejected） |
| `ash-run-events.html` | dataflow | `run_events` 写入 → SSE 续传 → 回放/派生 |
| `ash-memory-lifecycle.html` | dataflow | Memory 候选 → 评审 → 检索 → hit_used |
| `ash-feature-delivery.html` | workflow | `feature_delivery` 场景步骤与门禁 |
| `ash-hotfix.html` | workflow | `hotfix` 场景（SRE 定界 → 人审发布） |
| `ash-security-patch.html` | workflow | `security_patch` 场景（评估 → 补丁 → 公告） |
| `ash-quest-goal.html` | workflow | Quest：from-goal → Plan → Approve → Run |
| `ash-v5-governance.html` | architecture | **v5** 双核管控：薄交互 + Registry + 厚评审 |
| `ash-v5-review-thick.html` | sequence | **v5** 厚评审时序（评分 → 入队 → 决定 → Improve） |
| `ash-v5-score-monitor.html` | dataflow | **v5** 评分与交互监控数据流 |
| `ash-v5-space-control.html` | workflow | **v5** 空间管控与测评流程 |
| `ash-v5-review-lifecycle.html` | lifecycle | **v5** 评审项状态机（含团队多签） |
| `*.json` | 规格 | 可编辑后重新渲染的 JSON 源 |

## 图集覆盖

```
系统层     ash-system / ash-v2-dual-core / ash-package-map
部署拓扑   ash-deploy-local → ash-deploy-stage1 → ash-deploy-stage2
运行时     ash-run-execution / ash-run-lifecycle / ash-run-events
沙箱       ash-sandbox-fallback
Quest      ash-quest-goal / ash-goal-plan-lifecycle
记忆治理   ash-memory-lifecycle
场景 DSL   ash-feature-delivery / ash-hotfix / ash-security-patch
v5 管控    ash-v5-governance / ash-v5-review-thick / ash-v5-score-monitor / ash-v5-space-control / ash-v5-review-lifecycle
```

部署图启用 `engineering_profile: deployment-ownership`（所有权 tag + 跨界机制命名），对齐 `doc/design/ARCH-架构与技术选型.md` §12。

v5 设计规格：[`docs/superpowers/specs/2026-09-13-v5-dual-core-governance-design.md`](../../../docs/superpowers/specs/2026-09-13-v5-dual-core-governance-design.md) · 程序：[`doc/plan/v5-governance-program.md`](../../plan/v5-governance-program.md)。

## 重新生成

```bash
ARCHIFY="$HOME/.agents/skills/archify"
DIAG="$(pwd)/doc/diagrams/archify"

deliver() {
  local type="$1" spec="$2" html="$3"
  node "$ARCHIFY/bin/archify.mjs" validate "$type" "$DIAG/$spec" --quality showcase
  node "$ARCHIFY/bin/archify.mjs" deliver "$type" "$DIAG/$spec" "$DIAG/$html" --quality showcase
  node "$ARCHIFY/bin/archify.mjs" visual-check "$DIAG/$html" --json
}

deliver architecture ash-system.architecture.json ash-system.html
deliver architecture ash-v2-dual-core.architecture.json ash-v2-dual-core.html
deliver architecture ash-package-map.architecture.json ash-package-map.html
deliver architecture ash-deploy-local.architecture.json ash-deploy-local.html
deliver architecture ash-deploy-stage1.architecture.json ash-deploy-stage1.html
deliver architecture ash-deploy-stage2.architecture.json ash-deploy-stage2.html
deliver sequence     ash-run-execution.sequence.json ash-run-execution.html
deliver sequence     ash-sandbox-fallback.sequence.json ash-sandbox-fallback.html
deliver lifecycle    ash-run.lifecycle.json ash-run-lifecycle.html
deliver lifecycle    ash-goal-plan.lifecycle.json ash-goal-plan-lifecycle.html
deliver dataflow     ash-run-events.dataflow.json ash-run-events.html
deliver dataflow     ash-memory-lifecycle.dataflow.json ash-memory-lifecycle.html
deliver workflow     ash-feature-delivery.workflow.json ash-feature-delivery.html
deliver workflow     ash-hotfix.workflow.json ash-hotfix.html
deliver workflow     ash-security-patch.workflow.json ash-security-patch.html
deliver workflow     ash-quest-goal.workflow.json ash-quest-goal.html
deliver architecture ash-v5-governance.architecture.json ash-v5-governance.html
deliver sequence     ash-v5-review-thick.sequence.json ash-v5-review-thick.html
deliver dataflow     ash-v5-score-monitor.dataflow.json ash-v5-score-monitor.html
deliver workflow     ash-v5-space-control.workflow.json ash-v5-space-control.html
deliver lifecycle    ash-v5-review.lifecycle.json ash-v5-review-lifecycle.html
```

本地预览（可选）：

```bash
node "$ARCHIFY/bin/archify.mjs" preview architecture "$DIAG/ash-system.architecture.json" /tmp/preview.html --quality showcase --open
```

## 浏览器打开

直接用浏览器打开任一 `*.html`。支持暗色/亮色切换、缩放、关系追踪与 PNG/SVG 导出。

## 相关：数据库 ER

持久化实体关系（Mermaid `erDiagram`）见 [`../er/README.md`](../er/README.md)：分域 6 张 + 全库总览巨图。
