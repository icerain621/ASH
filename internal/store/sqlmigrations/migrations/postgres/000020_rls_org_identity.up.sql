-- Org identity tables filtered via app.ash_org_id (and members also respect app.ash_space_id).

CREATE OR REPLACE FUNCTION ash_rls_org_visible(p_org_id text) RETURNS boolean
    LANGUAGE sql STABLE
AS $$
    SELECT current_setting('app.ash_rls_bypass', true) = 'on'
        OR (
            NULLIF(current_setting('app.ash_org_id', true), '') IS NOT NULL
            AND p_org_id = current_setting('app.ash_org_id', true)
        );
$$;

CREATE OR REPLACE FUNCTION ash_rls_role_visible(p_org_id text) RETURNS boolean
    LANGUAGE sql STABLE
AS $$
    SELECT current_setting('app.ash_rls_bypass', true) = 'on'
        OR p_org_id IS NULL
        OR p_org_id = ''
        OR (
            NULLIF(current_setting('app.ash_org_id', true), '') IS NOT NULL
            AND p_org_id = current_setting('app.ash_org_id', true)
        );
$$;

CREATE OR REPLACE FUNCTION ash_rls_member_visible(p_org_id text, p_space_id text) RETURNS boolean
    LANGUAGE sql STABLE
AS $$
    SELECT current_setting('app.ash_rls_bypass', true) = 'on'
        OR (
            NULLIF(current_setting('app.ash_space_id', true), '') IS NOT NULL
            AND NULLIF(p_space_id, '') IS NOT NULL
            AND p_space_id = current_setting('app.ash_space_id', true)
        )
        OR (
            NULLIF(current_setting('app.ash_org_id', true), '') IS NOT NULL
            AND p_org_id = current_setting('app.ash_org_id', true)
            AND (p_space_id IS NULL OR p_space_id = '')
        );
$$;

CREATE OR REPLACE FUNCTION ash_rls_user_visible(p_user_id text) RETURNS boolean
    LANGUAGE sql STABLE
AS $$
    SELECT current_setting('app.ash_rls_bypass', true) = 'on'
        OR EXISTS (
            SELECT 1 FROM members m
            WHERE m.user_id = p_user_id
              AND m.org_id = current_setting('app.ash_org_id', true)
              AND m.status = 'active'
        );
$$;

GRANT EXECUTE ON FUNCTION ash_rls_org_visible(text) TO ash_app;
GRANT EXECUTE ON FUNCTION ash_rls_org_visible(text) TO ash_rls_tester;
GRANT EXECUTE ON FUNCTION ash_rls_role_visible(text) TO ash_app;
GRANT EXECUTE ON FUNCTION ash_rls_role_visible(text) TO ash_rls_tester;
GRANT EXECUTE ON FUNCTION ash_rls_member_visible(text, text) TO ash_app;
GRANT EXECUTE ON FUNCTION ash_rls_member_visible(text, text) TO ash_rls_tester;
GRANT EXECUTE ON FUNCTION ash_rls_user_visible(text) TO ash_app;
GRANT EXECUTE ON FUNCTION ash_rls_user_visible(text) TO ash_rls_tester;

DO $rls$
DECLARE
    rec record;
BEGIN
    FOR rec IN
        SELECT *
        FROM (
            VALUES
                ('orgs', 'ash_rls_org_visible(id)'),
                ('roles', 'ash_rls_role_visible(org_id)'),
                ('members', 'ash_rls_member_visible(org_id, space_id)'),
                ('users', 'ash_rls_user_visible(id)')
        ) AS t(table_name, policy_expr)
    LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', rec.table_name);
        EXECUTE format('DROP POLICY IF EXISTS %I ON %I', 'ash_space_' || rec.table_name, rec.table_name);
        EXECUTE format(
            'CREATE POLICY %I ON %I USING (%s) WITH CHECK (%s)',
            'ash_space_' || rec.table_name,
            rec.table_name,
            rec.policy_expr,
            rec.policy_expr
        );
    END LOOP;
END
$rls$;

UPDATE schema_meta
SET value = '20', updated_at = NOW()
WHERE key = 'schema_version';
