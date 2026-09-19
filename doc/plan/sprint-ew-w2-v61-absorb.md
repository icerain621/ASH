# Sprint EW · W2 v6.1 吸收任务板

> 波次：**W2 · v6.1**（[`ash-feature-inventory.md`](ash-feature-inventory.md)）  
> 前置：**W0 ✅ · W1 ✅**  
> 原则：fail-closed；不引入 Cordis；中文 commit

| # | Sprint | 人时 | 依赖 | 状态 |
|---|--------|------|------|------|
| **EW21** | 会话线程 Fork（parentThreadId） | 40 | — | ✅ |
| **EW22** | Compare 投影到管控台 | 40 | EW21 | ✅ |
| **EW23** | Compact 事件进 Chat/Trajectory | 48 | — | ✅ |
| **EW24** | 记忆四视角字段回填 | 80 | — | ✅ |
| **EW25** | W2 签字 | 16 | EW22+EW23+EW24 | ✅ |

## EW21 — Fork

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| EW21-1 | `interaction.Fork` + `parentThreadId` | 同 run 可多次 fork；单测 | ✅ |
| EW21-2 | `POST /interactions/threads/{id}/fork` | handler | ✅ |
| EW21-3 | SQL `000039` parent 列；去掉 (run,kind) 唯一 | expectedVersion 39 | ✅ |

**非目标：** 分支独立事件流（fork 仍折同一 run 事件）；UI 树（EW22）。

## EW22 — Compare 投影

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| EW22-1 | 管控台比对可选同会话 fork | ThreadComparePanel vitest | ✅ |
| EW22-2 | Fork 按钮调用 `POST …/fork` | 客户端 `forkInteractionThread` | ✅ |

## EW23 — Compact 投影

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| EW23-1 | `harness.compaction` → Chat/Trajectory 节点 | vitest | ✅ |

## EW24 — 记忆视角

| # | 项 | 验收 | 状态 |
|---|-----|------|------|
| EW24-1 | 场景/Skill/Tools/项目分组 | 标签 + scopeRepo；无字段才提示 | ✅ |
| EW24-2 | 带 Run 新建时回填 `scenario:` 与空的 scopeRepo | `TestCreateCandidateFillsScenarioFromRun` | ✅ |

## EW25 — 签字

无新 HTTP 路径。CHANGELOG 与本板勾选即完成。
