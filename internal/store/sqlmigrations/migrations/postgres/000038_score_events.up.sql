-- GV09: score_events for review rubric; ImproveProposal source linkage.

CREATE TABLE IF NOT EXISTS score_events (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    target_type VARCHAR(64) NOT NULL,
    target_id VARCHAR(128) NOT NULL,
    run_id VARCHAR(64) NOT NULL DEFAULT '',
    correctness INTEGER NOT NULL DEFAULT 0,
    safety INTEGER NOT NULL DEFAULT 0,
    citable INTEGER NOT NULL DEFAULT 0,
    efficiency INTEGER NOT NULL DEFAULT 0,
    composite DOUBLE PRECISION NOT NULL DEFAULT 0,
    actor_id VARCHAR(128) NOT NULL DEFAULT '',
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_score_events_space_id ON score_events (space_id);
CREATE INDEX IF NOT EXISTS idx_score_events_target ON score_events (target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_score_events_run_id ON score_events (run_id);

ALTER TABLE improve_proposals
    ADD COLUMN IF NOT EXISTS source VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS score_event_id VARCHAR(64) NOT NULL DEFAULT '';

DO $rls$
DECLARE
    tbl text := 'score_events';
BEGIN
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', tbl);
    EXECUTE format('DROP POLICY IF EXISTS %I ON %I', 'ash_space_' || tbl, tbl);
    EXECUTE format(
        'CREATE POLICY %I ON %I USING (ash_rls_space_visible(space_id)) WITH CHECK (ash_rls_space_visible(space_id))',
        'ash_space_' || tbl,
        tbl
    );
END
$rls$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ash_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON score_events TO ash_app;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ash_rls_tester') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON score_events TO ash_rls_tester;
    END IF;
END
$$;

UPDATE schema_meta
SET value = '38', updated_at = NOW()
WHERE key = 'schema_version';
