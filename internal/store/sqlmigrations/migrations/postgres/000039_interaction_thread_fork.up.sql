-- W2 EW21: thread fork parent pointer. Drop (run_id, kind) unique so multiple forks share a run.

ALTER TABLE interaction_threads
    ADD COLUMN IF NOT EXISTS parent_thread_id VARCHAR(64) NOT NULL DEFAULT '';

DROP INDEX IF EXISTS uniq_interaction_threads_run_kind;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_interaction_threads_one_main
    ON interaction_threads (run_id) WHERE kind = 'main';

CREATE INDEX IF NOT EXISTS idx_interaction_threads_parent_thread_id
    ON interaction_threads (parent_thread_id);

UPDATE schema_meta
SET value = '39', updated_at = NOW()
WHERE key = 'schema_version';
