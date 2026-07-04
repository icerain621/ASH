# 附录 C：Memory Schema（SQLite）+ 迁移与兼容（v0.1）

> 本附录定义 ASH 记忆系统的元数据存储（SQLite）、候选/评审流、去重/冲突关系、迁移与向后兼容策略。
>
> 冻结等级：**M0 冻结**（candidate→review→merge + schemaVersion + evidence 强制）。edges/dedupe/conflicts 的深度治理为 P1+。

## 1. 目标与原则
- **分层**：L0/L1/L2/L3（临时→项目→团队→证据库）
- **可治理**：candidate→review→merge，所有长期层写入必须可追溯
- **可兼容**：schemaVersion + migration；旧记录可读，新写入走新版本
- **可观测**：candidate/review/query/hit_used/ttl/migrate 事件齐全

## 2. 表结构（DDL）
```sql
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS memory_records (
  id TEXT PRIMARY KEY,
  layer TEXT NOT NULL CHECK(layer IN ('L0','L1','L2','L3')),
  status TEXT NOT NULL CHECK(status IN ('candidate','approved','rejected','deprecated')),
  schema_version INTEGER NOT NULL,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  scope_repo TEXT,
  scope_team TEXT,
  tags_json TEXT NOT NULL DEFAULT '[]',
  ttl_days INTEGER,
  sensitivity TEXT NOT NULL DEFAULT 'normal' CHECK(sensitivity IN ('normal','restricted','secret')),
  dedupe_key TEXT,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_memory_layer_status ON memory_records(layer, status);
CREATE INDEX IF NOT EXISTS idx_memory_repo ON memory_records(scope_repo);
CREATE INDEX IF NOT EXISTS idx_memory_dedupe ON memory_records(dedupe_key);

CREATE TABLE IF NOT EXISTS memory_evidence (
  id TEXT PRIMARY KEY,
  memory_id TEXT NOT NULL REFERENCES memory_records(id) ON DELETE CASCADE,
  kind TEXT NOT NULL CHECK(kind IN ('file','pr','ci','url')),
  ref TEXT NOT NULL,
  digest TEXT,
  meta_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_evidence_memory ON memory_evidence(memory_id);

CREATE TABLE IF NOT EXISTS memory_reviews (
  id TEXT PRIMARY KEY,
  memory_id TEXT NOT NULL REFERENCES memory_records(id) ON DELETE CASCADE,
  decision TEXT NOT NULL CHECK(decision IN ('approve','reject','deprecate')),
  reviewer_id TEXT,
  reason TEXT NOT NULL,
  policy_profile TEXT NOT NULL,
  created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_reviews_memory ON memory_reviews(memory_id);

CREATE TABLE IF NOT EXISTS memory_edges (
  id TEXT PRIMARY KEY,
  from_memory_id TEXT NOT NULL REFERENCES memory_records(id) ON DELETE CASCADE,
  to_memory_id TEXT NOT NULL REFERENCES memory_records(id) ON DELETE CASCADE,
  rel TEXT NOT NULL CHECK(rel IN ('supersedes','duplicates','conflicts','depends_on')),
  meta_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_edges_from ON memory_edges(from_memory_id);
CREATE INDEX IF NOT EXISTS idx_edges_to ON memory_edges(to_memory_id);

CREATE TABLE IF NOT EXISTS memory_migrations (
  id TEXT PRIMARY KEY,
  from_version INTEGER NOT NULL,
  to_version INTEGER NOT NULL,
  tool_version TEXT NOT NULL,
  applied_at INTEGER NOT NULL,
  summary TEXT NOT NULL,
  meta_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS audit_log (
  id TEXT PRIMARY KEY,
  trace_id TEXT,
  run_id TEXT,
  actor_id TEXT,
  event_type TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_trace ON audit_log(trace_id);
CREATE INDEX IF NOT EXISTS idx_audit_run ON audit_log(run_id);
```

## 3. 写入与评审流程（状态机）
### 3.1 Candidate 写入（M0 必须）
- 插入 `memory_records(status='candidate')`
- 插入 `memory_evidence`（**L1+ 必须至少 1 条**；否则只能进入 L0）
- 写 `audit_log(event_type='memory.candidate_created')`
- emit `memory.candidate_created`

### 3.2 Review（M0 必须）
- 插入 `memory_reviews(decision, reason, policy_profile)`
- 更新 `memory_records.status`
- 写 `audit_log(event_type='memory.reviewed')`
- emit `memory.reviewed`

### 3.3 Merge（approved 的含义）
- `approved` 即进入长期层可被检索使用（仍需受 TTL/过期策略影响）
- 对于 `secret` 级别：默认不进入外发观测插件，仅保留审计索引

## 4. TTL/过期与降权
- `ttl_days` 到期后不强删，标记为 `deprecated` 或在检索侧降权。
- emit：
  - `memory.ttl_expired`（检测到到期）
  - `memory.deprecated`（人工或规则废弃）

**TODO（负责人：知识管理员）**：~~L1/L2 默认 TTL 与复核周期~~ → **已实现**（默认 90/365d；`ASH_MEMORY_TTL_REVIEW_DAYS` 复核窗口；`ttl-sweep` 到期弃用）。  
**验收方式**：到期前 7 天进入复核队列；过期后 sweep 弃用且不可检索；TR1-06 回归通过。

## 5. 去重、冲突与替代（P1+ 深化）
- `dedupe_key`：确定性哈希（例如 scope+title+归一化 body 的 hash）
- `memory_edges`：
  - `duplicates`：重复
  - `supersedes`：新记录替代旧记录（旧记录降权/废弃）
  - `conflicts`：冲突（检索时提示并要求 human）
  - `depends_on`：依赖（用于组合性知识）

## 6. 迁移与向后兼容
- `schema_version`：每条 record 自带版本。
- `memory_migrations`：记录迁移批次（from→to、工具版本、摘要）。**Postgres RLS**：全局审计表（无 `space_id`），见 `store.PostgresRLSGlobalTables()`，不纳入 `000013` 租户策略（Doctor **M3-10**）。
- `memory_evidence` / `memory_reviews`：无 `space_id`，通过 `memory_id` 关联 `memory_records`。**Postgres RLS**：SQL rev **19** `ash_rls_memory_visible`（`PostgresRLSMemoryScopedTables()`）；集成测试 `TestPostgresRLSSpaceIsolationOnMemoryChildren`（`make postgres-rls-e2e`）。
- **兼容策略**
  - 旧记录：保持只读可检索（compat layer 映射到当前内部表示）
  - 新写入：强制使用当前 schema_version

**TODO（负责人：后端）**：~~定义 v1→v2 的示例迁移~~ → **已实现**（`internal/memory/migrate.go` v1→v2：L1/L2 默认 TTL；`CurrentSchemaVersion=2`）。  
**验收方式**：迁移后旧记录可读、检索语义不变；TR3-01 回归通过。

