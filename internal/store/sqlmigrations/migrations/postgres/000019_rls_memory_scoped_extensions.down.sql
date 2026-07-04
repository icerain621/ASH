DROP POLICY IF EXISTS ash_space_memory_evidence ON memory_evidence;
ALTER TABLE memory_evidence DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS ash_space_memory_reviews ON memory_reviews;
ALTER TABLE memory_reviews DISABLE ROW LEVEL SECURITY;
DROP FUNCTION IF EXISTS ash_rls_memory_visible(text);

UPDATE schema_meta
SET value = '18', updated_at = NOW()
WHERE key = 'schema_version';
