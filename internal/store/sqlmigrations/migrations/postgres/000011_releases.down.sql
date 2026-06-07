DROP TABLE IF EXISTS rollback_drills;
DROP TABLE IF EXISTS release_gate_results;
DROP TABLE IF EXISTS release_checklist_items;
DROP TABLE IF EXISTS release_records;

UPDATE schema_meta
SET value = '10', updated_at = NOW()
WHERE key = 'schema_version';
