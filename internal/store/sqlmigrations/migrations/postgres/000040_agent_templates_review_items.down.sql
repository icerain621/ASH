DROP POLICY IF EXISTS ash_space_review_items ON review_items;
DROP POLICY IF EXISTS ash_space_agent_templates ON agent_templates;
DROP TABLE IF EXISTS review_items;
DROP TABLE IF EXISTS agent_templates;

UPDATE schema_meta
SET value = '39', updated_at = NOW()
WHERE key = 'schema_version';
