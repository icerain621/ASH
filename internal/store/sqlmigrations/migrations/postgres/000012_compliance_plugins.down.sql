DROP TABLE IF EXISTS improve_proposals;
DROP TABLE IF EXISTS plugin_registry;
DROP TABLE IF EXISTS audit_exports;
DROP TABLE IF EXISTS approval_requests;
DROP TABLE IF EXISTS secret_records;

UPDATE schema_meta
SET value = '11', updated_at = NOW()
WHERE key = 'schema_version';
