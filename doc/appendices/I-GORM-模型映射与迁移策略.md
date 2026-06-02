# 附录 I：GORM 模型映射与迁移策略（v0.1）

> 目的：将附录 A/C/F 的“表结构/Schema”落到 Go（Gin+GORM）的模型与迁移执行方式，保证 M0 可落地、P1/P2 可演进。
>
> 冻结等级：M0 冻结“表结构+最小迁移流程”；P1 引入严格迁移与回滚；P2 支持 Postgres。

## 1. 数据库与驱动选型
- **M0 默认**：SQLite
  - Windows 建议纯 Go sqlite 驱动（避免 CGO）
- **P2 演进**：Postgres

## 2. GORM 模型映射（建议）
> 说明：以下为“字段→模型”映射要点，避免实现时偏离 Schema。

### 2.1 `run_events`
- 表：`run_events(id, run_id, seq, ts, type, payload_json, created_at)`
- 关键约束：
  - `UNIQUE(run_id, seq)`
  - `INDEX(run_id, type)`
- 模型建议：
  - `PayloadJSON` 用 `[]byte` 或 `datatypes.JSON`

### 2.2 `memory_records` / `memory_evidence` / `memory_reviews` / `memory_edges` / `memory_migrations`
- 直接对应附录 C 的 DDL
- `tags_json/meta_json/payload_json` 建议统一使用 JSON 类型封装（SQLite 下为 TEXT）
- 枚举字段（layer/status/sensitivity/rel/decision）在应用层做强校验（同时 DB CHECK 约束保留）

### 2.3 `audit_log`
- 作为审计索引；生产建议外置 append-only（P2）
- M0 仍需保证：
  - 每次 tool.called/tool.result/memory.reviewed/run.failed 都写入 audit_log（可异步）

### 2.4 artifacts 与 run summary
- artifacts 文件落盘规范见附录 F
- `manifest.json` 与 `run.json` 的 schema 见 `docs/appendices/schemas/`
- DB 中建议只存：
  - run 的索引字段（runId/status/startedAt/finishedAt）
  - manifest 摘要（digest/路径）

**TODO（负责人：后端）**：明确“run 索引表”是否需要（推荐需要，用于 runs 列表）。  
**验收方式**：`GET /runs` 不应扫描文件系统全量目录。

## 3. 迁移策略（M0→P2）
### 3.1 M0（最小可用）
- 方式：
  - 启动时执行“幂等建表”迁移（AutoMigrate 或 SQL 文件）
  - 记录 `schema_version`（建议单独表 `schema_meta`）与 `memory_migrations`（记忆迁移批次）
- 风险：AutoMigrate 在复杂变更时不可控（列改名/回滚困难）

### 3.2 P1（严格迁移）
- 引入 `golang-migrate`（或等价）：
  - `migrations/0001_init.up.sql` / `.down.sql`
  - CI 检查：迁移可应用、可回滚（至少开发环境）
- 约束：所有 schema 变更必须以迁移文件形式提交，不允许运行时隐式变更

### 3.3 P2（多租户/高并发）
- Postgres：
  - 连接池参数与超时治理
  - 事务边界明确（run_events 写入批量）
  - 分区/归档（run_events 与 audit 可能增长快）

## 4. 数据一致性与性能建议
- `run_events` 写入：
  - 批量插入（buffer）降低 IO
  - `seq` 分配使用单行锁/原子计数（按 run_id）
- memory query：
  - 常用索引：`(layer,status)`、`scope_repo`、`dedupe_key`
- 审计：
  - 高吞吐时使用异步队列写入，失败发事件并保底落本地文件

## 5. 验收（必须通过）
- M0：TR0 全绿；`run_events` 不丢序；`GET /runs` 可用；memory review 链路可追溯
- P1：TR1/TR2 子集全绿；迁移演练通过；回滚策略明确

