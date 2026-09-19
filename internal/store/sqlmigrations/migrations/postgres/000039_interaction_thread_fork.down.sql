DROP INDEX IF EXISTS idx_interaction_threads_parent_thread_id;
DROP INDEX IF EXISTS uniq_interaction_threads_one_main;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_interaction_threads_run_kind
    ON interaction_threads (run_id, kind);

ALTER TABLE interaction_threads DROP COLUMN IF EXISTS parent_thread_id;

UPDATE schema_meta
SET value = '38', updated_at = NOW()
WHERE key = 'schema_version';
