DROP TABLE IF EXISTS rag_chunks;
DROP TABLE IF EXISTS rag_documents;
DROP TABLE IF EXISTS memory_edges;
DROP TABLE IF EXISTS memory_reviews;
DROP TABLE IF EXISTS memory_evidence;

UPDATE schema_meta
SET value = '5', updated_at = NOW()
WHERE key = 'schema_version';
