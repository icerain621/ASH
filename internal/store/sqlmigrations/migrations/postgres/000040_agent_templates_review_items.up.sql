-- Agent template versions + in-process review queue persistence (v7 G01) + tenant RLS.
CREATE TABLE IF NOT EXISTS agent_templates (
    id              TEXT PRIMARY KEY,
    space_id        TEXT NOT NULL,
    template_id     TEXT NOT NULL,
    version         TEXT NOT NULL,
    status          TEXT NOT NULL,
    manifest_json   TEXT NOT NULL,
    created_by      TEXT,
    approved_by     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approved_at     TIMESTAMPTZ,
    UNIQUE (space_id, template_id, version)
);

CREATE INDEX IF NOT EXISTS idx_agent_templates_space_status
    ON agent_templates (space_id, status);

CREATE TABLE IF NOT EXISTS review_items (
    id              TEXT PRIMARY KEY,
    space_id        TEXT NOT NULL,
    kind            TEXT NOT NULL,
    target_id       TEXT NOT NULL,
    status          TEXT NOT NULL,
    reason          TEXT,
    decided_by      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    decided_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_review_items_space_status
    ON review_items (space_id, status);
CREATE INDEX IF NOT EXISTS idx_review_items_kind_target
    ON review_items (kind, target_id);

DO $rls$
DECLARE
    tbl text;
BEGIN
    FOREACH tbl IN ARRAY ARRAY['agent_templates', 'review_items']
    LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', tbl);
        EXECUTE format('DROP POLICY IF EXISTS %I ON %I', 'ash_space_' || tbl, tbl);
        EXECUTE format(
            'CREATE POLICY %I ON %I USING (ash_rls_space_visible(space_id)) WITH CHECK (ash_rls_space_visible(space_id))',
            'ash_space_' || tbl,
            tbl
        );
    END LOOP;
END
$rls$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ash_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON agent_templates TO ash_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON review_items TO ash_app;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ash_rls_tester') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON agent_templates TO ash_rls_tester;
        GRANT SELECT, INSERT, UPDATE, DELETE ON review_items TO ash_rls_tester;
    END IF;
END $$;

UPDATE schema_meta
SET value = '40', updated_at = NOW()
WHERE key = 'schema_version';
