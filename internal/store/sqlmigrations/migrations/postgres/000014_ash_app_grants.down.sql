REVOKE EXECUTE ON FUNCTION ash_rls_run_visible(text) FROM ash_rls_tester;
REVOKE EXECUTE ON FUNCTION ash_rls_space_visible(text) FROM ash_rls_tester;
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM ash_rls_tester;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM ash_rls_tester;
REVOKE USAGE ON SCHEMA public FROM ash_rls_tester;

REVOKE EXECUTE ON FUNCTION ash_rls_run_visible(text) FROM ash_app;
REVOKE EXECUTE ON FUNCTION ash_rls_space_visible(text) FROM ash_app;
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM ash_app;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM ash_app;
REVOKE USAGE ON SCHEMA public FROM ash_app;

UPDATE schema_meta
SET value = '13', updated_at = NOW()
WHERE key = 'schema_version';
