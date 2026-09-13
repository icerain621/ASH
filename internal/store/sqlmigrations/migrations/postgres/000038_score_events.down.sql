ALTER TABLE improve_proposals DROP COLUMN IF EXISTS score_event_id;
ALTER TABLE improve_proposals DROP COLUMN IF EXISTS source;

DROP INDEX IF EXISTS idx_score_events_run_id;
DROP INDEX IF EXISTS idx_score_events_target;
DROP INDEX IF EXISTS idx_score_events_space_id;
DROP TABLE IF EXISTS score_events;

UPDATE schema_meta
SET value = '37', updated_at = NOW()
WHERE key = 'schema_version';
