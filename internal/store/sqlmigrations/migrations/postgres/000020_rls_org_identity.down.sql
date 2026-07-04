DROP POLICY IF EXISTS ash_space_users ON users;
ALTER TABLE users DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS ash_space_members ON members;
ALTER TABLE members DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS ash_space_roles ON roles;
ALTER TABLE roles DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS ash_space_orgs ON orgs;
ALTER TABLE orgs DISABLE ROW LEVEL SECURITY;
DROP FUNCTION IF EXISTS ash_rls_user_visible(text);
DROP FUNCTION IF EXISTS ash_rls_member_visible(text, text);
DROP FUNCTION IF EXISTS ash_rls_role_visible(text);
DROP FUNCTION IF EXISTS ash_rls_org_visible(text);

UPDATE schema_meta
SET value = '19', updated_at = NOW()
WHERE key = 'schema_version';
