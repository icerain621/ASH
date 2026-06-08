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

UPDATE schema_meta
SET value = '15', updated_at = NOW()
WHERE key = 'schema_version';
