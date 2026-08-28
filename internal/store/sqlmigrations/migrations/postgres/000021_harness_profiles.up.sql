-- Harness Profile versions (v2 Sprint DH) + tenant RLS.
CREATE TABLE IF NOT EXISTS harness_profile_versions (
    id              TEXT PRIMARY KEY,
    space_id        TEXT NOT NULL,
    name            TEXT NOT NULL,
    version         INTEGER NOT NULL,
    status          TEXT NOT NULL,
    spec_json       TEXT NOT NULL,
    parent_version  INTEGER,
    created_by      TEXT,
    promoted_by     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    promoted_at     TIMESTAMPTZ,
    UNIQUE (space_id, name, version)
);

CREATE INDEX IF NOT EXISTS idx_harness_profile_space_name_status
    ON harness_profile_versions (space_id, name, status);

DO $rls$
DECLARE
    tbl text := 'harness_profile_versions';
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
        GRANT SELECT, INSERT, UPDATE, DELETE ON harness_profile_versions TO ash_app;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ash_rls_tester') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON harness_profile_versions TO ash_rls_tester;
    END IF;
END $$;

UPDATE schema_meta
SET value = '21', updated_at = NOW()
WHERE key = 'schema_version';
