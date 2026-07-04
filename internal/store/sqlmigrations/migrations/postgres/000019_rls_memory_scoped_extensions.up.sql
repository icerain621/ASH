-- Memory child tables filtered via memory_records.space_id.
CREATE OR REPLACE FUNCTION ash_rls_memory_visible(p_memory_id text) RETURNS boolean
    LANGUAGE sql STABLE
AS $$
    SELECT current_setting('app.ash_rls_bypass', true) = 'on'
        OR EXISTS (
            SELECT 1 FROM memory_records m
            WHERE m.id = p_memory_id
              AND m.space_id = current_setting('app.ash_space_id', true)
        );
$$;

GRANT EXECUTE ON FUNCTION ash_rls_memory_visible(text) TO ash_app;
GRANT EXECUTE ON FUNCTION ash_rls_memory_visible(text) TO ash_rls_tester;

DO $rls$
DECLARE
    tbl text;
BEGIN
    FOREACH tbl IN ARRAY ARRAY['memory_evidence', 'memory_reviews']
    LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', tbl);
        EXECUTE format('DROP POLICY IF EXISTS %I ON %I', 'ash_space_' || tbl, tbl);
        EXECUTE format(
            'CREATE POLICY %I ON %I USING (ash_rls_memory_visible(memory_id)) WITH CHECK (ash_rls_memory_visible(memory_id))',
            'ash_space_' || tbl,
            tbl
        );
    END LOOP;
END
$rls$;

UPDATE schema_meta
SET value = '19', updated_at = NOW()
WHERE key = 'schema_version';
