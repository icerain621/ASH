-- Dev / e2e Postgres roles for ASH tenant RLS (idempotent).
-- ash_app: production-style app role (NOBYPASSRLS); use with ASH_POSTGRES_RLS_FORCE=1.
-- ash_rls_tester: integration tests that SET LOCAL ROLE to verify isolation.

DO $$ BEGIN
  CREATE ROLE ash_app LOGIN PASSWORD 'ash_app' NOINHERIT NOBYPASSRLS;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

GRANT USAGE ON SCHEMA public TO ash_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO ash_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO ash_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO ash_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO ash_app;

DO $$ BEGIN
  CREATE ROLE ash_rls_tester LOGIN PASSWORD 'ash_rls_tester' NOINHERIT NOBYPASSRLS;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

GRANT USAGE ON SCHEMA public TO ash_rls_tester;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO ash_rls_tester;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO ash_rls_tester;
GRANT ash_rls_tester TO ash;
