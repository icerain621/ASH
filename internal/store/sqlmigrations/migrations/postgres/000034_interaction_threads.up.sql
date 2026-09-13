-- GV04: interaction_threads for Session→Thread observability index.

CREATE TABLE IF NOT EXISTS interaction_threads (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    session_id VARCHAR(64) NOT NULL DEFAULT '',
    run_id VARCHAR(64) NOT NULL,
    kind VARCHAR(32) NOT NULL DEFAULT 'main',
    status VARCHAR(32) NOT NULL DEFAULT 'open',
    digest VARCHAR(128) NOT NULL DEFAULT '',
    head_seq BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_interaction_threads_run_kind
    ON interaction_threads (run_id, kind);
CREATE INDEX IF NOT EXISTS idx_interaction_threads_space_id ON interaction_threads (space_id);
CREATE INDEX IF NOT EXISTS idx_interaction_threads_session_id ON interaction_threads (session_id);
CREATE INDEX IF NOT EXISTS idx_interaction_threads_status ON interaction_threads (status);

DO $rls$
DECLARE
    tbl text := 'interaction_threads';
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
        GRANT SELECT, INSERT, UPDATE, DELETE ON interaction_threads TO ash_app;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ash_rls_tester') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON interaction_threads TO ash_rls_tester;
    END IF;
END
$$;

UPDATE schema_meta
SET value = '34', updated_at = NOW()
WHERE key = 'schema_version';
