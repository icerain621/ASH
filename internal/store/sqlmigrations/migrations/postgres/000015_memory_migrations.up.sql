-- Global memory schema migration audit (no space_id). Intentionally excluded from tenant RLS (000013).
CREATE TABLE IF NOT EXISTS memory_migrations (
    id VARCHAR(64) PRIMARY KEY,
    from_version INTEGER NOT NULL,
    to_version INTEGER NOT NULL,
    tool_version VARCHAR(64) NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    summary TEXT NOT NULL,
    meta_json TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_memory_migrations_applied_at ON memory_migrations (applied_at DESC);

COMMENT ON TABLE memory_migrations IS 'Global memory schema migration audit; no space_id — excluded from ash tenant RLS policies.';

UPDATE schema_meta
SET value = '15', updated_at = NOW()
WHERE key = 'schema_version';
