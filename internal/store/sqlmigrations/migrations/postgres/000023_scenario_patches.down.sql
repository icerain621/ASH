DROP TABLE IF EXISTS scenario_patch_drafts;

UPDATE schema_meta
SET value = '22', updated_at = NOW()
WHERE key = 'schema_version';
