INSERT INTO schema_meta (key, value, updated_at)
VALUES ('sql_migrations', 'golang-migrate/v1', NOW())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at;

UPDATE schema_meta
SET value = '2', updated_at = NOW()
WHERE key = 'schema_version';
