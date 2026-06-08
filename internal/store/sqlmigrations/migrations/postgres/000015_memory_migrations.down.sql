DROP TABLE IF EXISTS memory_migrations;

UPDATE schema_meta
SET value = '14', updated_at = NOW()
WHERE key = 'schema_version';
