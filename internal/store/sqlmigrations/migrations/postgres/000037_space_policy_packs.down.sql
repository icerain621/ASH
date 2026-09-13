DROP TABLE IF EXISTS space_policy_packs;

UPDATE schema_meta
SET value = '36', updated_at = NOW()
WHERE key = 'schema_version';
