-- Extend run_id-scoped tenant RLS (model_usage filtered via runs.space_id).
DO $rls$
DECLARE
    tbl text;
BEGIN
    FOREACH tbl IN ARRAY ARRAY['model_usage']
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
SET value = '18', updated_at = NOW()
WHERE key = 'schema_version';
