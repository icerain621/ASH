-- RAG vector store point refs (Sprint DX17).
CREATE TABLE IF NOT EXISTS rag_vector_refs (
    id          TEXT PRIMARY KEY,
    space_id    TEXT NOT NULL DEFAULT 'local',
    repo_root   TEXT NOT NULL,
    chunk_id    TEXT NOT NULL,
    point_id    TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (space_id, repo_root, chunk_id)
);
CREATE INDEX IF NOT EXISTS idx_rag_vector_refs_space ON rag_vector_refs (space_id);
CREATE INDEX IF NOT EXISTS idx_rag_vector_refs_space_repo ON rag_vector_refs (space_id, repo_root);
CREATE INDEX IF NOT EXISTS idx_rag_vector_refs_point ON rag_vector_refs (point_id);

DO $rls$
DECLARE
    tbl text := 'rag_vector_refs';
BEGIN
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', tbl);
    EXECUTE format('DROP POLICY IF EXISTS %I ON %I', 'ash_space_' || tbl, tbl);
    EXECUTE format(
        'CREATE POLICY %I ON %I USING (ash_rls_space_visible(space_id)) WITH CHECK (ash_rls_space_visible(space_id))',
        'ash_space_' || tbl,
        tbl
    );
END
$rls$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ash_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON rag_vector_refs TO ash_app;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ash_rls_tester') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON rag_vector_refs TO ash_rls_tester;
    END IF;
END
$$;
