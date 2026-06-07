DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS memory_records;
DROP TABLE IF EXISTS run_steps;

UPDATE schema_meta
SET value = '3', updated_at = NOW()
WHERE key = 'schema_version';
