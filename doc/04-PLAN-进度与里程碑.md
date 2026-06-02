# ASH 进度与里程碑（Plan）v0.1

> 文档状态：v0.1（可执行排期版）。以 M0/M1 为主，M2/M3 为展望。

## 1. 团队与产能假设
- **默认团队**：4 人（2 后端/平台 + 1 前端 + 1 全栈/测试）
- **有效产能**：约 100h/周（团队合计）
- **迭代节奏**：2 周/Sprint（≈200h/Sprint）

**TODO（负责人：项目经理）**：补全 RACI（每 Epic/Story 的 owner、reviewer、approver）。  
**验收方式**：Sprint 0 评审通过 RACI 表。

## 2. 里程碑定义
- **M0（P0）**：Feature 交付闭环 + 回放 + 记忆评审 + SSE 可观测 + Prometheus + `ash doctor(TR0)`
- **M1（P1）**：三场景模板齐全 + 引用门禁 + 记忆治理(去重/冲突) + OTel/质量插件 + 自我迭代 1.0 + `doctor(TR1/TR2 子集)`
- **M2（P2）**：组织级权限/合规/多租户 + 强隔离 + 存储演进
- **M3（P3）**：网关/技能市场/知识图谱增强

## 3. 关键路径（M0）
事件与回放 → DSL → workflow+checkpoint → toolbus → memory → obs+doctor

## 4. Sprint 计划（M0）
> 详细 Backlog 与验收矩阵见 `docs/appendices/`。

### Sprint 1（事件流 + DSL 可加载 + 最小 gate）
- 交付：SSE 续传；DSL 校验加载；git.status gate

### Sprint 2（能改代码 + 能跑测试 + checkpoint 初版）
- 交付：apply_patch/test_report；Feature 模板基本闭环；checkpoint per_step

### Sprint 3（记忆评审 + obs 插件 + doctor TR0）
- 交付：candidate→review→merge；Prometheus；doctor(TR0) 全绿；Web 最小闭环

## 5. 验收与门禁
- **TR0（M0）**：闭环/事件/回放
- **TR1/TR2（M1）**：provider 替换、schema 校验、checkpoint 恢复、注入红队、secret 扫描、MCP 隔离

## 6. 风险登记（摘要）
- DSL 语义膨胀、checkpoint 语义、apply_patch 可靠性、跨平台工具链、RAG 引用质量、记忆评审吞吐、门禁误杀、观测泄密风险

**TODO（负责人：项目经理）**：补全 Risk Register（触发阈值/回滚动作/owner）。  
**验收方式**：每 Sprint Review 更新一次并追踪趋势。

## 7. 实现进度（代码仓）
> 代码目录：`ash/`（Go Worker，Gin + GORM + SQLite）

| 模块 | 状态 | 说明 |
|---|---|---|
| Worker 骨架（Gin） | ✅ 已启动 | `ash/cmd/worker`，health/ready/metrics |
| GORM + SQLite 模型 | ✅ 已启动 | runs、run_events、memory/audit 表 AutoMigrate |
| Runs API | ✅ 已启动 | POST/GET runs，创建时写入 run.started/finished 事件 |
| Run 控制 (resume/replay) | ✅ M0 | `POST /runs/:id/resume`、`POST /runs/:id/replay`；`run.json` 持久化 |
| SSE 事件流 | ✅ 已启动 | `/api/v1/runs/:id/stream`，支持 Last-Event-ID |
| Rules DSL 引擎 | ✅ 已启动 | 解析/校验/Loader/Engine；`scenarios/feature_delivery.yaml` |
| Scenarios API | ✅ 已启动 | list/get/validate |
| Runs 绑定 DSL | ✅ 已启动 | 按 steps 发 step/tool/checkpoint 事件 |
| ToolBus / Policy | ✅ M0 | git.status/diff/checkout、apply_patch、test.run |
| Artifacts 四件套 | ✅ M0 | manifest + diff/test_report/release/rollback |
| ash doctor (TR0) | ✅ M0 | `go run ./cmd/cli doctor --suite TR0` 或 POST `/api/v1/doctor/run` |
| Memory Review API | ✅ M0 | candidates CRUD/review、query、hit-used、audit |
| Web UI | ✅ M0 React | Vite+React+TS；`make web-build` → `frontend/dist`；Worker `/ui/` |
| 目录结构 (monorepo) | ✅ 脚本就绪 | `make reorganize` 或 `scripts/reorganize.ps1` → `backend/` + `doc/design/` |
| Swagger (swag) | ✅ M0 已实现端点 | `/docs`、`/openapi.json`；`make swagger` 再生 |
| Proto/gRPC 插件 ABI | ⏳ 待做 | P1 |

本地验证：
```bash
make tidy && make web-build && make run
make test
make doctor
# Run control
curl -X POST http://localhost:8080/api/v1/runs/{runId}/resume
curl -X POST http://localhost:8080/api/v1/runs/{runId}/replay -H "Content-Type: application/json" -d "{\"mode\":\"exact\"}"
```
