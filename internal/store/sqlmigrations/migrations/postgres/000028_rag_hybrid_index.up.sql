-- RAG Hybrid path + symbol index (Sprint DX9).
CREATE TABLE IF NOT EXISTS rag_path_entries (
    id          TEXT PRIMARY KEY,
    space_id    TEXT NOT NULL DEFAULT 'local',
    repo_root   TEXT NOT NULL,
    path        TEXT NOT NULL,
    basename    TEXT NOT NULL,
    digest      TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (space_id, repo_root, path)
);
CREATE INDEX IF NOT EXISTS idx_rag_path_entries_basename ON rag_path_entries (basename);
CREATE INDEX IF NOT EXISTS idx_rag_path_entries_space ON rag_path_entries (space_id);

CREATE TABLE IF NOT EXISTS rag_symbols (
    id          TEXT PRIMARY KEY,
    space_id    TEXT NOT NULL DEFAULT 'local',
    repo_root   TEXT NOT NULL,
    path        TEXT NOT NULL,
    name        TEXT NOT NULL,
    kind        TEXT NOT NULL DEFAULT 'unknown',
    line        INTEGER NOT NULL DEFAULT 1,
    digest      TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_rag_symbols_name ON rag_symbols (space_id, repo_root, name);
CREATE INDEX IF NOT EXISTS idx_rag_symbols_path ON rag_symbols (space_id, repo_root, path);

DO $rls$
DECLARE
    tbl text := 'rag_path_entries';
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

DO $rls$
DECLARE
    tbl text := 'rag_symbols';
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
        GRANT SELECT, INSERT, UPDATE, DELETE ON rag_path_entries TO ash_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON rag_symbols TO ash_app;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ash_rls_tester') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON rag_path_entries TO ash_rls_tester;
        GRANT SELECT, INSERT, UPDATE, DELETE ON rag_symbols TO ash_rls_tester;
    END IF;
END
$$;
