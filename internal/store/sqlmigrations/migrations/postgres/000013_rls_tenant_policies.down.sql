DO $rls$
DECLARE
    tbl text;
BEGIN
    FOREACH tbl IN ARRAY ARRAY[
        'runs', 'memory_records', 'memory_edges', 'rag_documents', 'rag_chunks',
        'quality_metrics', 'mcp_tools', 'feedback', 'repo_connections', 'ci_runs', 'ci_jobs',
        'ci_diagnoses', 'alert_rules', 'alert_events', 'alert_silences', 'release_records',
        'release_checklist_items', 'release_gate_results', 'rollback_drills', 'secret_records',
        'audit_log', 'approval_requests', 'resource_scopes', 'audit_exports', 'audit_policies',
        'plugin_registry', 'improve_proposals', 'spaces',
        'run_steps', 'tool_calls', 'agent_tasks', 'artifact_index', 'checkpoints', 'run_events'
    ]
    LOOP
        EXECUTE format('DROP POLICY IF EXISTS %I ON %I', 'ash_space_' || tbl, tbl);
        EXECUTE format('ALTER TABLE %I DISABLE ROW LEVEL SECURITY', tbl);
    END LOOP;
END
$rls$;

DROP FUNCTION IF EXISTS ash_rls_run_visible(text);
DROP FUNCTION IF EXISTS ash_rls_space_visible(text);

UPDATE schema_meta
SET value = '12', updated_at = NOW()
WHERE key = 'schema_version';
