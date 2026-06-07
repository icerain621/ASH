DROP TABLE IF EXISTS alert_silences;
DROP TABLE IF EXISTS alert_events;
DROP TABLE IF EXISTS alert_rules;

UPDATE schema_meta
SET value = '9', updated_at = NOW()
WHERE key = 'schema_version';
