-- Scenario patch drafts (v2 Sprint DZ) + tenant RLS.
CREATE TABLE IF NOT EXISTS scenario_patch_drafts (
    id              TEXT PRIMARY KEY,
    space_id        TEXT NOT NULL,
    scenario_name   TEXT NOT NULL,
    from_version    TEXT,
    to_version      TEXT,
    title           TEXT NOT NULL,
    diff_text       TEXT NOT NULL,
    status          TEXT NOT NULL,
    created_by      TEXT,
    decided_by      TEXT,
    decision_note   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    decided_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_scenario_patch_space_status
    ON scenario_patch_drafts (space_id, status);

DO $rls$
DECLARE
    tbl text := 'scenario_patch_drafts';
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
        GRANT SELECT, INSERT, UPDATE, DELETE ON scenario_patch_drafts TO ash_app;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ash_rls_tester') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON scenario_patch_drafts TO ash_rls_tester;
    END IF;
END $$;

UPDATE schema_meta
SET value = '23', updated_at = NOW()
WHERE key = 'schema_version';
