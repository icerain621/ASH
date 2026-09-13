DROP INDEX IF EXISTS idx_run_events_visibility;

ALTER TABLE run_events
    DROP COLUMN IF EXISTS visibility;

UPDATE schema_meta
SET value = '32', updated_at = NOW()
WHERE key = 'schema_version';
