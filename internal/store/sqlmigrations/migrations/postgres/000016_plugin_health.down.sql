ALTER TABLE plugin_registry
    DROP COLUMN IF EXISTS last_export_at,
    DROP COLUMN IF EXISTS export_errors,
    DROP COLUMN IF EXISTS drop_count;

UPDATE schema_meta
SET value = '15', updated_at = NOW()
WHERE key = 'schema_version';
