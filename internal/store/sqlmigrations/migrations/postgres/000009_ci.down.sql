DROP TABLE IF EXISTS ci_diagnoses;
DROP TABLE IF EXISTS ci_jobs;
DROP TABLE IF EXISTS ci_runs;
DROP TABLE IF EXISTS repo_connections;

UPDATE schema_meta
SET value = '8', updated_at = NOW()
WHERE key = 'schema_version';
