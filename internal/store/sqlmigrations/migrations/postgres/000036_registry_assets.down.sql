DROP INDEX IF EXISTS idx_memory_assets_kind;
DROP INDEX IF EXISTS idx_memory_assets_status;
DROP INDEX IF EXISTS idx_memory_assets_space_id;
DROP TABLE IF EXISTS memory_assets;

DROP INDEX IF EXISTS idx_agent_assets_kind;
DROP INDEX IF EXISTS idx_agent_assets_status;
DROP INDEX IF EXISTS idx_agent_assets_space_id;
DROP TABLE IF EXISTS agent_assets;

UPDATE schema_meta
SET value = '35', updated_at = NOW()
WHERE key = 'schema_version';
