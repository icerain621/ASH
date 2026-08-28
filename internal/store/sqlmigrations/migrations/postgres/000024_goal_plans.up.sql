-- Goal plan drafts (v2 Sprint DJ) + tenant RLS.
CREATE TABLE IF NOT EXISTS goal_plans (
    id                TEXT PRIMARY KEY,
    space_id          TEXT NOT NULL,
    goal              TEXT NOT NULL,
    scenario_name     TEXT NOT NULL,
    scenario_version  TEXT NOT NULL,
    policy_profile    TEXT NOT NULL DEFAULT 'default',
    route_reason      TEXT,
    inputs_json       TEXT NOT NULL DEFAULT '{}',
    steps_json        TEXT NOT NULL DEFAULT '[]',
    status            TEXT NOT NULL,
    run_id            TEXT,
    created_by        TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_goal_plans_space_status
    ON goal_plans (space_id, status);

DO $rls$
DECLARE
    tbl text := 'goal_plans';
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
        GRANT SELECT, INSERT, UPDATE, DELETE ON goal_plans TO ash_app;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ash_rls_tester') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON goal_plans TO ash_rls_tester;
    END IF;
END
$$;
