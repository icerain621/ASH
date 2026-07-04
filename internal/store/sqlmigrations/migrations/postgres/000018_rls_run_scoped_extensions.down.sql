DROP POLICY IF EXISTS ash_space_model_usage ON model_usage;
ALTER TABLE model_usage DISABLE ROW LEVEL SECURITY;

UPDATE schema_meta
SET value = '17', updated_at = NOW()
WHERE key = 'schema_version';
