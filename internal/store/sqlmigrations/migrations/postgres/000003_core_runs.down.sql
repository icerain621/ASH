DROP TABLE IF EXISTS run_events;
DROP TABLE IF EXISTS runs;

UPDATE schema_meta
SET value = '2', updated_at = NOW()
WHERE key = 'schema_version';
