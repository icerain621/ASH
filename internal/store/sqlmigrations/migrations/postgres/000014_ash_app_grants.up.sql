-- ash_app / ash_rls_tester roles and DML grants (production worker + integration tests).

DO $$ BEGIN
    CREATE ROLE ash_app LOGIN PASSWORD 'ash_app' NOINHERIT NOBYPASSRLS;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

GRANT USAGE ON SCHEMA public TO ash_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO ash_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO ash_app;
GRANT EXECUTE ON FUNCTION ash_rls_space_visible(text) TO ash_app;
GRANT EXECUTE ON FUNCTION ash_rls_run_visible(text) TO ash_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO ash_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO ash_app;

DO $$ BEGIN
    CREATE ROLE ash_rls_tester LOGIN PASSWORD 'ash_rls_tester' NOINHERIT NOBYPASSRLS;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

GRANT USAGE ON SCHEMA public TO ash_rls_tester;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO ash_rls_tester;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO ash_rls_tester;
GRANT EXECUTE ON FUNCTION ash_rls_space_visible(text) TO ash_rls_tester;
GRANT EXECUTE ON FUNCTION ash_rls_run_visible(text) TO ash_rls_tester;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ash') THEN
        GRANT ash_rls_tester TO ash;
    END IF;
END $$;

UPDATE schema_meta
SET value = '14', updated_at = NOW()
WHERE key = 'schema_version';
