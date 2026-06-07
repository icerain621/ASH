DELETE FROM schema_meta WHERE key = 'sql_migrations';

UPDATE schema_meta
SET value = '1', updated_at = NOW()
WHERE key = 'schema_version';
