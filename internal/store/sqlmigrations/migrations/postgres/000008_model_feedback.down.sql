DROP TABLE IF EXISTS feedback;
DROP TABLE IF EXISTS mcp_tools;
DROP TABLE IF EXISTS quality_metrics;
DROP TABLE IF EXISTS model_usage;

UPDATE schema_meta
SET value = '7', updated_at = NOW()
WHERE key = 'schema_version';
