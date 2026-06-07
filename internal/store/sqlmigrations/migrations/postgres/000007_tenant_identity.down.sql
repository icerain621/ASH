DROP TABLE IF EXISTS audit_policies;
DROP TABLE IF EXISTS resource_scopes;
DROP TABLE IF EXISTS members;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS spaces;
DROP TABLE IF EXISTS orgs;
DROP TABLE IF EXISTS users;

UPDATE schema_meta
SET value = '6', updated_at = NOW()
WHERE key = 'schema_version';
