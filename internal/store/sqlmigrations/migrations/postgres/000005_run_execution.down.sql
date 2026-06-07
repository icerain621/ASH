DROP TABLE IF EXISTS checkpoints;
DROP TABLE IF EXISTS artifact_index;
DROP TABLE IF EXISTS agent_tasks;
DROP TABLE IF EXISTS tool_calls;

UPDATE schema_meta
SET value = '4', updated_at = NOW()
WHERE key = 'schema_version';
