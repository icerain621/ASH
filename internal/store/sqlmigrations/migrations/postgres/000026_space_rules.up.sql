-- Space rules (v2 Sprint DM) + tenant RLS.
CREATE TABLE IF NOT EXISTS space_rules (
    id           TEXT PRIMARY KEY,
    space_id     TEXT NOT NULL UNIQUE,
    version      INTEGER NOT NULL DEFAULT 1,
    body_yaml    TEXT NOT NULL,
    source       TEXT NOT NULL DEFAULT 'api',
    file_path    TEXT,
    updated_by   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_space_rules_space
    ON space_rules (space_id);

DO $rls$
DECLARE
    tbl text := 'space_rules';
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
        GRANT SELECT, INSERT, UPDATE, DELETE ON space_rules TO ash_app;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ash_rls_tester') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON space_rules TO ash_rls_tester;
    END IF;
END
$$;
