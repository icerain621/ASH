ALTER TABLE plugin_registry
    ADD COLUMN IF NOT EXISTS last_export_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS export_errors BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS drop_count BIGINT NOT NULL DEFAULT 0;

UPDATE schema_meta
SET value = '16', updated_at = NOW()
WHERE key = 'schema_version';
