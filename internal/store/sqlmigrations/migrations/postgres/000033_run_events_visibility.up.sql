-- GV01: event envelope visibility (model_visible|ui_only|audit).
-- Empty string preserves legacy rows; readers apply DefaultVisibility(type).

ALTER TABLE run_events
    ADD COLUMN IF NOT EXISTS visibility VARCHAR(32) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_run_events_visibility ON run_events (visibility);

UPDATE schema_meta
SET value = '33', updated_at = NOW()
WHERE key = 'schema_version';
