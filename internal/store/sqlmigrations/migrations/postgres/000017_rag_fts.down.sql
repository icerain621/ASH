DROP INDEX IF EXISTS idx_rag_chunks_search_vector;
ALTER TABLE rag_chunks DROP COLUMN IF EXISTS search_vector;
