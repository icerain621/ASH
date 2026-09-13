-- GV08: one SpacePolicyPack row per space (citation / multi-sign / SLA).

CREATE TABLE IF NOT EXISTS space_policy_packs (
    space_id VARCHAR(64) PRIMARY KEY,
    citation_mode VARCHAR(64) NOT NULL DEFAULT 'optional',
    multi_sign BOOLEAN NOT NULL DEFAULT FALSE,
    review_sla_hours INTEGER NOT NULL DEFAULT 72,
    body_json TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DO $rls$
DECLARE
    tbl text := 'space_policy_packs';
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
        GRANT SELECT, INSERT, UPDATE, DELETE ON space_policy_packs TO ash_app;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ash_rls_tester') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON space_policy_packs TO ash_rls_tester;
    END IF;
END
$$;

UPDATE schema_meta
SET value = '37', updated_at = NOW()
WHERE key = 'schema_version';
