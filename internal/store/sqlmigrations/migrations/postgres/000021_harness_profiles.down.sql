DROP POLICY IF EXISTS ash_space_harness_profile_versions ON harness_profile_versions;
DROP TABLE IF EXISTS harness_profile_versions;

UPDATE schema_meta
SET value = '20', updated_at = NOW()
WHERE key = 'schema_version';
