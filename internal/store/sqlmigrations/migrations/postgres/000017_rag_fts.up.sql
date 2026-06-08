-- Postgres full-text search for rag_chunks (Sprint K).
ALTER TABLE rag_chunks
    ADD COLUMN IF NOT EXISTS search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector(
            'simple',
            coalesce(path, '') || ' ' || coalesce(symbol, '') || ' ' || coalesce(text, '')
        )
    ) STORED;

CREATE INDEX IF NOT EXISTS idx_rag_chunks_search_vector ON rag_chunks USING GIN (search_vector);
