-- GV07–08: AgentAsset / MemoryAsset registry tables with space RLS.

CREATE TABLE IF NOT EXISTS agent_assets (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    kind VARCHAR(64) NOT NULL,
    ref_id VARCHAR(128) NOT NULL DEFAULT '',
    name VARCHAR(256) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_assets_space_id ON agent_assets (space_id);
CREATE INDEX IF NOT EXISTS idx_agent_assets_status ON agent_assets (status);
CREATE INDEX IF NOT EXISTS idx_agent_assets_kind ON agent_assets (kind);

CREATE TABLE IF NOT EXISTS memory_assets (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    kind VARCHAR(64) NOT NULL,
    ref_id VARCHAR(128) NOT NULL DEFAULT '',
    name VARCHAR(256) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_memory_assets_space_id ON memory_assets (space_id);
CREATE INDEX IF NOT EXISTS idx_memory_assets_status ON memory_assets (status);
CREATE INDEX IF NOT EXISTS idx_memory_assets_kind ON memory_assets (kind);

DO $rls$
DECLARE
    tbl text;
BEGIN
    FOREACH tbl IN ARRAY ARRAY['agent_assets', 'memory_assets']
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
        GRANT SELECT, INSERT, UPDATE, DELETE ON agent_assets TO ash_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON memory_assets TO ash_app;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ash_rls_tester') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON agent_assets TO ash_rls_tester;
        GRANT SELECT, INSERT, UPDATE, DELETE ON memory_assets TO ash_rls_tester;
    END IF;
END
$$;

UPDATE schema_meta
SET value = '36', updated_at = NOW()
WHERE key = 'schema_version';
