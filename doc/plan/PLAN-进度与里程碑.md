# ASH 项目计划与进度（Plan）v0.3

> **状态**：现行排期真相源（2026-08-28）  
> **当前版本**：**v1**（`v0.1.0-mvp` → `v1.0.0`）  
> **下一版本**：**v2**（见 [`v2-dual-core-evolution-plan.md`](v2-dual-core-evolution-plan.md)）  
> **代码锚点**：Doctor ALL **53/53** · M3 **11/11** · M4 **6/6** · M5 **4/4** · TR3 **10/10** · SQL rev **27** · RLS **46**（v2 DH–DV 收口；tag 待人工）  
> **归属**：[`plan/`](README.md)  
> **关联**：短待办 [`TODO.md`](TODO.md) · 范围 [`mvp-release-scope.md`](mvp-release-scope.md) · 风险 [`risk-register.md`](risk-register.md) · 设计 [`../design/`](../design/README.md)

---

## 0. 一句话结论

**MVP 功能与自动化门禁已完成**；当前主线不是新功能，而是 **生产环境验收（云 RDS / 真实 CI·ExecGo）→ 正式发布签字 → 上线观察**。设计上 M0–M3 主体已闭环，未完成项集中在「生产切流 + 合规/生态增强」。

---

## 1. 里程碑总览（设计完成度）

| 里程碑 | 设计目标 | 设计状态 | 代码状态 | 说明 |
|--------|----------|----------|----------|------|
| **M0** Feature 闭环 + 回放 + 记忆评审 + SSE + Prometheus + Doctor TR0 | ✅ 设计完成 | ✅ 已实现 | Runs/DSL/ToolBus/Artifacts/Memory/SSE/Doctor TR0 |
| **M1** 三场景 + 引用门禁 + 记忆治理 + OTel/质量 + 自我迭代 + TR1/TR2 子集 | ✅ 设计完成 | ✅ 已实现 | hotfix/security_patch、citations、edges、improve、TR1/TR2 |
| **M2** 组织权限/合规/多租户隔离 | ✅ 设计完成（基础） | ✅ 基础已实现 | `authz`、resource-scopes、policy.denied、合规导出 |
| **M3** 规模化 / Postgres / RLS / Scale | ✅ 设计完成 | ✅ 本地已实现 | SQL **26**、RLS **46**；**云 RDS 切流未完成** |
| **PRD MVP 后四项** CI 诊断 / KPI / 反馈告警 / 发布治理 | ✅ 设计完成 | ✅ 已实现 | fixture 可验；**真实 GitHub/ExecGo 待生产验证** |
| **P3 生态** 网关 / 技能市场 / 知识图谱 / 向量库 | ⚠️ 仅展望设计 | ❌ 未实现 | 范围冻结外 backlog |

### 设计已完成 vs 未完成（聚焦）

**设计已完成（可直接按附录/OpenAPI 实现，代码已落地）**

- 事件协议、Rules DSL、Artifacts digest、Memory schema v2、Obs 指标 derive、Doctor TR0–TR3/M2/M3
- OpenAPI `/api/v1` 契约、错误码表、Postgres 迁移与 RLS 策略目录
- MVP 发布范围、检查清单、烟测与签字 runbook

**设计未完成 / 仍为草案（阻塞或影响后续迭代）**

| 项 | 优先级 | 文档位置 | 影响 |
|----|--------|----------|------|
| ~~商业落地：谁付费/谁决策/组织样板~~ | P2 | PRD §3 / ORG 文档 | **已关闭**（Sprint CO；`org-templates`） |
| ~~数据分级与保留期~~ | P1（合规） | 附录 J / PRD §8 | **已关闭**（Sprint CF） |
| ~~插件打包/签名策略~~ | P2 | ARCH §3.2 / 附录 H §6 | **已关闭**（CH 骨架 + CR 轮换 SOP / `plugin-sign-smoke`） |
| 向量库 / 符号索引路线 | P3 | ARCH §5–6 | RAG 增强，FTS 已够用 |
| ~~Artifacts 跨平台默认路径与权限~~ | P2 | 附录 F | **已关闭**（Sprint CG） |
| ~~Canonical JSON 实现说明（产物 digest）~~ | P2 | 附录 F | **已关闭**（Sprint CG） |
| ~~RACI（Epic owner）~~ | P2 | 本文件 §7 | **已关闭**（发布阶段 RACI + Epic 归属） |
| 强隔离沙箱 / IdP 联邦 / 技能市场 | P3 | PRD Out | 下一产品世代 |

> HLD「GORM 迁移策略」、ARCH「CLI/Web 待实现」、附录 I「run 索引表」等历史 TODO：**已由代码落地**，见下文进度表。

---

## 2. 代码完成度快照

| 域 | 状态 | 证据 |
|----|------|------|
| Worker + CLI + 控制台 12 页 | ✅ | `cmd/worker`、`cmd/cli`、`frontend/` vitest smoke |
| Runs 状态机 / Resume/Replay/Cancel/Approve | ✅ | Sprint BT–BW |
| SSE 重连 + Last-Event-ID + 轮询回退 | ✅ | Sprint BM/BN |
| Memory TTL / 治理 / feedback 衰减 | ✅ | Sprint AL/BR |
| Postgres SQL + RLS（本地 Docker） | ✅ | `postgres-*-e2e`、`postgres-app-gate` |
| OpenAPI / Doctor / 发布门禁 | ✅ | `openapi-check`、`mvp-signoff`、`release-window-gate` |
| 云 RDS 生产切换 | ⚠️ | 本地 `cloud-acceptance` ✅；需 `config/cloud-rds.env` |
| 真实 GitHub CI / ExecGo live | ⏸ | fixture/static 已覆盖；待 token / E2E |
| 正式四人签字（非占位） | ⚠️ | dry-run 占位已过门禁；需真人回填 |
| 向量检索 / 技能市场 / 多区域 | ❌ | 范围外 |

---

## 3. 优先级梯度计划（自现在起）

### P0 — 发布闭环（环境 + 签字，1–2 周窗口）

| ID | 任务 | 可纯代码？ | 验收 |
|----|------|------------|------|
| P0-1 | 配置真实 `cloud-rds.env`，跑通 `make cloud-acceptance` | 否 | H-01~H-03 云签字勾选 |
| P0-2 | 切换日：`data-backup` → migrate → Worker `ASH_DATABASE_APP_URL` + RLS → `t0-alert-gate` | 否 | runbook §4 + 证据归档 |
| P0-3 | 产品/技术/测试/发布 **真人签字**（替换占位） | 否 | `signoff.env` + `make signoff-gate` |
| P0-4 | 远端同步：推送 `main` / 确认 CI 绿（若尚未） | 部分 | Actions 全绿 |
| P0-5 | 切换日再跑一次 `rollback-drill` 并签字 | 否 | R-12 可关闭或可接受 |

### P1 — 生产可信度（上线后 1–2 周，可与 P0 并行准备）

| ID | 任务 | 验收 |
|----|------|------|
| P1-1 | 真实 GitHub token：H-04/H-05 live | 诊断 rootCause 与真实 log 一致 |
| P1-2 | `ASH_EXECGO_E2E=1` ExecGo live（H-06） | Doctor M3-05 live |
| P1-3 | T+1 生产 KPI vs `t1-metrics-gate` 偏差 &lt;5% | KPI §9 |
| P1-4 | 数据分级与保留期草案落地（审计/事件/记忆） | ✅ 附录 J + data-policy / retention apply（Sprint CF） |
| P1-5 | 发布窗口抽测跨 space + RLS e2e（R-08） | ✅ API 门禁 Sprint CP；云 RLS live 随 H-01 |

### P2 — 硬化与产品化（发布后迭代）

| ID | 任务 | 备注 |
|----|------|------|
| P2-1 | Artifacts 路径策略 + canonical JSON 说明写入实现文档 | ✅ Sprint CG |
| P2-2 | 插件 ABI 签名/打包策略 + 生产 gRPC 暴露策略 | ✅ Sprint CH |
| P2-3 | Hotfix / Security Patch 场景模板深化 | ✅ Sprint CH |
| P2-4 | SSE 浏览器级 E2E；Agent 稳定性度量（R-02） | ✅ SSE E2E（CM）+ KPI-11（CN） |
| P2-5 | 商业组织样板与付费/决策模型 | ✅ Sprint CO |

### P3 — 下一世代（明确不做进 v0.1）

- 向量库（Chroma/Qdrant）与符号索引 Hybrid RAG  
- 完整容器沙箱 / 外部 IdP 联邦 / 计费  
- OpenClaw 式多端网关、Skill Marketplace、知识图谱深度  

---

## 4. 建议 Sprint 序列（接续 CE 之后）

| Sprint | 焦点 | 依赖 |
|--------|------|------|
| **CF** | P1-4 数据分级与保留期（纯代码） | ✅ 已完成 |
| **CG** | P2-1 Artifacts 路径 + canonical JSON（纯代码） | ✅ 已完成 |
| **CH** | P2-2/P2-3 插件签名 + Hotfix/Security 深化 | ✅ 已完成 |
| **CM** | P2-4 SSE 浏览器级 E2E（Playwright） | ✅ 已完成 |
| **CN** | P2-4 R-02 场景稳定率（KPI-11） | ✅ 已完成 |
| **CO** | P2-5 商业组织样板（PRD §3） | ✅ 已完成 |
| **CP** | P1-5 R-08 跨 space 发布窗口抽测门禁 | ✅ 已完成（云 RLS live 另轨） |
| **CQ** | KPI-08 SSE 稳定率口径硬化（R-07） | ✅ 已完成 |
| **CR** | 插件签名轮换 SOP + CLI/烟测 | ✅ 已完成 |
| **CS** | 危险操作目录 + human-step UX | ✅ 已完成 |
| **CT** | data-policy 导入环修复 + RACI 闭环 | ✅ 已完成 |
| **CU** | TR0 固定探测语料 + SpacePage/web-gate | ✅ 已完成 |
| **CV** | Docker 本地 Postgres 门禁复验 | ✅ 已完成 |
| **CW** | 本地 sqlite→Postgres E2E（M3-04） | ✅ 已完成 |
| **CX** | doctor static 全扫 + UI/文档门禁对齐 | ✅ 已完成 |
| **CY** | mvp-signoff 证据诚实性 + 清单死链 | ✅ 已完成 |
| **CI–CJ** | P0-1/P0-2 云 RDS 切流 + P0-3/P0-5 真人签字 | RDS + `cloud-rds.env` |
| **DH–DV** | **v2 双核心演进**（见 [`v2-dual-core-evolution-plan.md`](v2-dual-core-evolution-plan.md)） | v1 P0 或本地门禁稳定 |
| **CK–CL** | P1-1/P1-2 真实 CI / ExecGo + P1-3 观察 | 外部 token / 上线后 |

历史 Sprint AY–CE 明细与已完成清单见 [`TODO.md`](TODO.md)「已完成归档」节（仅保留摘要，细节以 CHANGELOG 为准）。

---

## 5. 模块进度表（实现对照设计）

| 模块 | 设计 | 代码 | 备注 |
|------|------|------|------|
| Worker / health / metrics | ✅ | ✅ | Gin |
| Runs + 控制面 + 状态机 | ✅ | ✅ | R-05 已 Mitigating |
| SSE 事件流 | ✅ | ✅ | 含轮询回退 |
| Rules DSL + 三场景 | ✅ | ✅ | JSON Schema 校验 |
| ToolBus / Policy / MCP | ✅ | ✅ | ExecGo 桥接需 live |
| Artifacts 四件套 | ✅ | ✅ | 附录 F §3.3 + canonical.go（CG） |
| Memory 评审/治理/TTL | ✅ | ✅ | schema v2 |
| RAG FTS（SQLite/Postgres） | ✅ | ✅ | 向量库未做（设计为 P1+/P3） |
| Doctor TR0–TR3 / M2 / M3 | ✅ | ✅ | 43 cases |
| 合规 / 权限 / Scale | ✅ | ✅ | |
| CI / KPI / Feedback / Releases | ✅ | ✅ | 生产外部依赖待验 |
| OTel 骨架 | ✅ | ✅ | 完整外发策略按组织开关 |
| Proto 插件 ABI | ✅ signature 字段 | ✅ 签名+SOP | 外部实机打包可选 |
| 向量库 / 技能市场 | 展望 | ❌ | P3 |

---

## 6. 风险与门禁（执行指引）

Top 风险行动见 [`risk-register.md`](risk-register.md)。发布命令：

```bash
# 日常
make regression-short && make web-gate

# 发布前本地
make local-readiness-gate

# 云切换
cp config/cloud-rds.env.example config/cloud-rds.env
# 填入真实 RDS 后：
bash scripts/source-cloud-rds-env.sh
make cloud-acceptance
ASH_RELEASE_WINDOW_LIVE=1 make release-window-gate
make mvp-signoff
```

---

## 7. 团队假设与 RACI（精简）

- **默认产能**：后端/平台 2 + 前端 1 + 全栈/测试 1（可按实际缩放）  
- **Epic owner（MVP 域）**

| Epic | Owner (A) | 主要执行 (R) |
|------|-----------|--------------|
| Control Plane / Runs / SSE | 技术负责人 | 后端 |
| Memory / RAG / 合规保留 | 技术负责人 | 后端 |
| ToolBus / Policy / 插件 ABI | 平台负责人 | 后端 |
| Doctor / 发布门禁 / OpenAPI | 测试/平台 | 全栈 |
| 控制台 / 组织样板 | 产品 + 前端 | 前端 |
| 云切流 / 签字 / 切换日 | 发布负责人 | 运维 + 技术 |

- **RACI（发布阶段）**

| 事项 | R | A | C | I |
|------|---|---|---|---|
| 云 RDS 切流 | 后端/运维 | 技术负责人 | 发布 | 产品 |
| 真人签字 | 发布 | 产品 | 技术/测试 | 全员 |
| 真实 CI/ExecGo | 后端 | 技术 | 算法 | 产品 |
| 范围解冻（P2+） | 产品 | 产品 | 技术 | 全员 |

---

## 8. 文档与本计划的关系

- 本文件取代旧版「仅 M0 Sprint 1–3」叙述及已归档的 10 周人天计划。  
- 实现细节规范仍以 [`../appendices/`](../appendices/README.md) + OpenAPI 为准；进度只在本文件与 `TODO.md` 更新。

---

## 9. v2 路线图（摘要）

| 版本 | 目标 Tag | 里程碑 | 关键 Sprint |
|------|----------|--------|-------------|
| v1.0 | `v1.0.0` | MVP 发布 | CI–CJ |
| v2.0-alpha | `v2.0.0-alpha` | M4 | DH、DX、DY、DJ–DL |
| v2.0-beta | `v2.0.0-beta` | M5 | DO–DZ、DX2、DQ |
| v2.0 | `v2.0.0` | M6 | DS–DV |

完整排期、DH/DX 验收与 KPI → [`v2-dual-core-evolution-plan.md`](v2-dual-core-evolution-plan.md)。  
Harness/沙盒设计 → [`../design/HLD-Harness与沙盒.md`](../design/HLD-Harness与沙盒.md)。  
Sprint DH 实现勾选 → [`sprint-dh-harness-implementation.md`](sprint-dh-harness-implementation.md)（✅ 骨架已合入代码）。
