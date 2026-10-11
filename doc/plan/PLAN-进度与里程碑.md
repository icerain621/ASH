# ASH 项目计划与进度 / Plan & Progress

> **状态 / Status**：现行排期真相源（2026-10-10）  
> **代码水位 / Watermark**：`main` · **v6.7 已冻结**；**v7 P0–P23 验收测试已绿**（签字可跳过）  
> **Tag**：`v0.1.0-mvp` 已打；`v2.*`–`v7.*` **均未自动打 tag**（人工 + 签字）  
> **Doctor / Schema（v7）**：ALL **56** · QUALITY **7** · BOOT **5** · SQL rev **40** · RLS **58** · 见 [`v7.7-release-scope.md`](v7.7-release-scope.md)  
> **归属**：[`plan/`](README.md) · 短待办 [`TODO.md`](TODO.md) · [`v6.x-program.md`](v6.x-program.md) · [`v7.x-program.md`](v7.x-program.md)  
> **发布勾选**：[`../progress/`](../progress/README.md) · [`../checklists/`](../checklists/smoke-index.md)

---

## 0. 一句话结论 / Verdict

**功能代际已冻到 v6.7**（Harness 回显沙箱目录）。冻结后主线上又落地了一批 **不升代** 的薄交付：会话推理等级投影、助手 `provider · model · effort` 账本字段、Composer/壳 APG·DSH 无障碍（P02）、控制台空态与测试门禁。  
**v6.7 is frozen.** Post-freeze work on `main` stays **version-neutral**: reasoning-effort projection, assistant caption fields, Composer/shell APG·DSH a11y (P02), console empty states, and test gates.

下一功能代 **v7** 的 P0–P10 验收测试已绿（最新 [`v7.10-release-scope.md`](v7.10-release-scope.md)），待真人签字。不开 **v6.8**。GA 与 v6 人工 tag 仍并行。  
**v7** P0–P10 acceptance tests are green (latest [`v7.10-release-scope.md`](v7.10-release-scope.md)); human sign-off is still open. There is no v6.8. GA and optional v6 tags stay in parallel.

---

## 1. 代际快照 / Generation snapshot

| 代际 | 主题 | 状态 |
|------|------|------|
| MVP → v1 | 门禁 / Doctor / 控制台 | ✅ 自动化达标；云切流与真人签字 ⏸ |
| v2.x–v3.2 | Harness / RAG / Quest / OIDC 网关 | ✅ 各代已冻结 |
| v4.0–v4.3 | Auth / 配额审计 / 插件 / 私有目录 | ✅ 已冻结 |
| v5.0 | 薄交互 + 管控 + 厚评审 | ✅ 已冻结（GV06） |
| v6.0 | Hooks / 会话语义 / 吸收收口 | ✅ 已冻结（EW130） |
| v6.3–v6.6 | RPC · Notifier · Ingress · Webhook | ✅ 已冻结 |
| **v6.7** | Harness `sandboxBackends` 回显 | ✅ **已冻结**（VX71–VX72） |
| 冻结后（不升代） | 推理等级 · P02 投影/无障碍 · 空态 · 测试 | ✅ 已合入 `main`（见 §3） |
| **v7** | 组件化 Agent × 可插拔记忆 × 三镜头观测 | ✅ **P0–P10 测试绿**；待签字 |

详情：[`v6.7-release-scope.md`](v6.7-release-scope.md) · [`v6.x-program.md`](v6.x-program.md)。

---

## 2. 当前主轨 / Active tracks

| 轨 | 内容 | 状态 |
|----|------|------|
| **A · 功能代** | v6.7 冻结。**v7 P0–P20 测试绿**（签字可跳过；`make v7-signoff`；不开 v6.8） | P20 已绿；签字⏸ |
| **B · P02 薄交互** | UI 只发意图、只渲投影；Composer/壳对齐 DSH/APG | **已收口**（含工作区键盘 / Timeline 滤镜组） |
| **C · GA / 上线** | 云 RDS、真人签字、可选 tag | G3 本地 dry-run ✅；G2/G4 ⏸ |
| **D · 质量门禁** | Vitest coverage · Playwright · 后端包覆盖补强 | ✅ 已合入；持续维护 |

---

## 3. 冻结后已交付（2026-09-23 → 2026-10-08） / Post-freeze delivered

不升代、不改 v6.7 Out。事实以 `CHANGELOG.md` `[Unreleased]` 与下列提交为准。  
Version-neutral. Facts live in CHANGELOG and commits below.

| 主题 | 摘要 | 代表提交 |
|------|------|----------|
| 文档分类 | 需求/设计/计划入口；叙事归档；agent 协作文档 | `da29289d` |
| 签字锚点 | ALL 63 / SQL 32 / RLS 56；本地云验收证据 | `153b23ee` |
| 测试 | Vitest coverage 门禁 · Playwright E2E；多包覆盖补强 | `15a7de96` 等 |
| 控制台 | 各业务页空态 / 错误 / 路由别名 | `a919fd34` |
| 推理等级 | 会话文档字段 → Chat/Quest/ACP/ExecGo 投影 | `cfd5261d` |
| 账本投影 | `assistant.delta/message` 带 `llmModel` + `providerKind`；气泡 caption | `d604285f` |
| P02 无障碍 | Composer 菜单/斜杠/门控；tablist；列分隔键盘调宽；`aria-current` 等 | `7b474964` |

---

## 4. 下一步（可执行） / Next steps

### 4.1 产品未决议前（默认）

1. **G3**：本地 `make rollback-drill` + `make release-window-gate` 已绿（2026-10-11）；生产切流仍待 G2。  
2. **GA 准备**：填 `cloud-rds.env` → `make cloud-acceptance`；四人真人签字；需要时再人工打 tag（本波不创建版本号）。  
3. **v7 签字**：`make v7-signoff-prep` → 四人填 `doc/evidence/v7-signatures-latest.md`（见 TODO `V7-SIGN`）。  
4. **质量**：`make regression-short && make web-gate`；保持 Doctor 套件计数与卡片一致。

### 4.2 下一功能代 v7（P0–P23 测试绿；签字可跳过）

- 程序与全部功能点：[`v7.x-program.md`](v7.x-program.md)。范围见 `v7.0`…`v7.23-release-scope.md`。设计总表：[`../design/ASH-Pi组件重构-改动清单.md`](../design/ASH-Pi组件重构-改动清单.md)。  
- 门禁：`make v7-signoff`（P0–P13）；真人签字 / 可选 tag 见 TODO `V7-SIGN`（本波跳过）。  
- 目录缺口下一刀候选：目录对照后薄收口 / 观测缝等（须先写 release-scope；禁止自动发明）。

### 4.3 明确不做

- 不开 v6.8 承载 v7；不把 v7 回填进已冻结的 v6.7 正文。  
- Hermes 全 IM / Cordis / YOLO / 公网计费技能市场 / Agentic RL / A2A。  
- 把冻结后 P02 工作回填进 v6.7 Out 或改写已冻结 scope 正文。

---

## 5. 模块进度（对照） / Module progress

| 模块 | 代码 | 备注 |
|------|------|------|
| Worker / CLI / OpenAPI | ✅ | `openapi-check` |
| Session + Intent + Steer/Queue | ✅ | 含 reasoningEffort / provider 投影 |
| Agent Chat 三栏壳 | ✅ | DSH 对标 + APG 键盘/ARIA 本轮加深 |
| Composer 席位（模型/审批/Plan/+） | ✅ | 互斥菜单；effort PATCH |
| Hooks / Notifier / Ingress | ✅ | v6.0–v6.6 |
| Harness + sandboxBackends 回显 | ✅ | v6.7 |
| Postgres SQL + RLS（本地） | ✅ | 云切流 ⏸ |
| 向量主路径 / 公网市场 / Active-Active | ❌ | 范围外 |

---

## 6. 门禁速查 / Gates

```bash
make openapi-check
make regression-short && make web-gate
make v6.7-signoff          # 打 v6.7.0 前
make v7-signoff            # 打 v7.0.0 前（不自动 tag）
# 云（需 cloud-rds.env）
bash scripts/source-cloud-rds-env.sh && make cloud-acceptance
```

---

## 7. 文档缺口 / Doc gaps

| 入口 | 状态 |
|------|------|
| 本文件 `PLAN-进度与里程碑.md` | ✅ 已重建（2026-10-08） |
| [`TODO.md`](TODO.md) | ✅ 同步冻结后项 |
| [`SRS-需求规格.md`](../design/SRS-需求规格.md) / [`SDD-总体设计.md`](../design/SDD-总体设计.md) | ⚠️ 索引有链、正文待建（叙事稿在 `archive/narrative/`） |
| [`GA-上线推进计划.md`](GA-上线推进计划.md) | ✅ 薄入口（指向清单） |

---

## 8. 修订记录 / Revisions

| 日期 | 说明 |
|------|------|
| 2026-10-09 | v7.4 P4 落地：记忆巩固提案、BOOT 套件、review/passk、三层记忆计数 |
| 2026-10-09 | v7.3 P3 落地：薄 feature_delivery 模板链、旧 kind 弃用事件、rag.query、无场景 tpl.react |
| 2026-10-09 | v7.2 P2 落地：抽取式压缩、技能披露、循环 token 配额、session.follow_up |
| 2026-10-09 | 下一功能代定为 v7（方向归档）；不开 v6.8；开工须先写 v7.0 scope |
| 2026-10-08 | 自 archive 叙事稿重建为现行 PLAN；对齐 v6.7 冻结 + 冻结后 P02/推理等级/测试水位（`7b474964`） |
| （历史） | 旧长文见 [`../archive/narrative/PLAN-进度与里程碑.md`](../archive/narrative/PLAN-进度与里程碑.md) |
