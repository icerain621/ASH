DROP INDEX IF EXISTS idx_feedback_run_id;
ALTER TABLE feedback DROP COLUMN IF EXISTS run_id;

UPDATE schema_meta
SET value = '21', updated_at = NOW()
WHERE key = 'schema_version';
