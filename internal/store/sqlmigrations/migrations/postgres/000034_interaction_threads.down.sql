DROP INDEX IF EXISTS idx_interaction_threads_status;
DROP INDEX IF EXISTS idx_interaction_threads_session_id;
DROP INDEX IF EXISTS idx_interaction_threads_space_id;
DROP INDEX IF EXISTS uniq_interaction_threads_run_kind;
DROP TABLE IF EXISTS interaction_threads;

UPDATE schema_meta
SET value = '33', updated_at = NOW()
WHERE key = 'schema_version';
