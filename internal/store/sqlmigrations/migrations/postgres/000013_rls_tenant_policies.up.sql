-- Tenant RLS policies (replaces runtime ApplyPostgresRLSPolicies DDL when ASH_POSTGRES_RLS=1).
-- Session vars: app.ash_space_id, app.ash_rls_bypass (see internal/store/rls.go).

CREATE OR REPLACE FUNCTION ash_rls_space_visible(col text) RETURNS boolean
    LANGUAGE sql STABLE
AS $$
    SELECT current_setting('app.ash_rls_bypass', true) = 'on'
        OR (
            NULLIF(current_setting('app.ash_space_id', true), '') IS NOT NULL
            AND col = current_setting('app.ash_space_id', true)
        );
$$;

CREATE OR REPLACE FUNCTION ash_rls_run_visible(p_run_id text) RETURNS boolean
    LANGUAGE sql STABLE
AS $$
    SELECT current_setting('app.ash_rls_bypass', true) = 'on'
        OR EXISTS (
            SELECT 1 FROM runs r
            WHERE r.id = p_run_id
              AND r.space_id = current_setting('app.ash_space_id', true)
        );
$$;

-- space_id (or id for spaces) scoped tables
DO $rls$
DECLARE
    rec record;
BEGIN
    FOR rec IN
        SELECT *
        FROM (
            VALUES
                ('runs', 'space_id'),
                ('memory_records', 'space_id'),
                ('memory_edges', 'space_id'),
                ('rag_documents', 'space_id'),
                ('rag_chunks', 'space_id'),
                ('quality_metrics', 'space_id'),
                ('mcp_tools', 'space_id'),
                ('feedback', 'space_id'),
                ('repo_connections', 'space_id'),
                ('ci_runs', 'space_id'),
                ('ci_jobs', 'space_id'),
                ('ci_diagnoses', 'space_id'),
                ('alert_rules', 'space_id'),
                ('alert_events', 'space_id'),
                ('alert_silences', 'space_id'),
                ('release_records', 'space_id'),
                ('release_checklist_items', 'space_id'),
                ('release_gate_results', 'space_id'),
                ('rollback_drills', 'space_id'),
                ('secret_records', 'space_id'),
                ('audit_log', 'space_id'),
                ('approval_requests', 'space_id'),
                ('resource_scopes', 'space_id'),
                ('audit_exports', 'space_id'),
                ('audit_policies', 'space_id'),
                ('plugin_registry', 'space_id'),
                ('improve_proposals', 'space_id'),
                ('spaces', 'id')
        ) AS t(table_name, space_column)
    LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', rec.table_name);
        EXECUTE format('DROP POLICY IF EXISTS %I ON %I', 'ash_space_' || rec.table_name, rec.table_name);
        EXECUTE format(
            'CREATE POLICY %I ON %I USING (ash_rls_space_visible(%I::text)) WITH CHECK (ash_rls_space_visible(%I::text))',
            'ash_space_' || rec.table_name,
            rec.table_name,
            rec.space_column,
            rec.space_column
        );
    END LOOP;
END
$rls$;

-- run_id scoped child tables
DO $rls$
DECLARE
    tbl text;
BEGIN
    FOREACH tbl IN ARRAY ARRAY[
        'run_steps', 'tool_calls', 'agent_tasks', 'artifact_index', 'checkpoints', 'run_events'
    ]
    LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', tbl);
        EXECUTE format('DROP POLICY IF EXISTS %I ON %I', 'ash_space_' || tbl, tbl);
        EXECUTE format(
            'CREATE POLICY %I ON %I USING (ash_rls_run_visible(run_id)) WITH CHECK (ash_rls_run_visible(run_id))',
            'ash_space_' || tbl,
            tbl
        );
    END LOOP;
END
$rls$;

UPDATE schema_meta
SET value = '13', updated_at = NOW()
WHERE key = 'schema_version';
