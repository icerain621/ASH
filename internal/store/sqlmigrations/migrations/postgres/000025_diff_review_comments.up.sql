-- Diff review comments (v2 Sprint DK) + tenant RLS.
CREATE TABLE IF NOT EXISTS diff_review_comments (
    id          TEXT PRIMARY KEY,
    space_id    TEXT NOT NULL,
    run_id      TEXT NOT NULL,
    file_path   TEXT NOT NULL,
    line_index  INTEGER NOT NULL,
    side        TEXT NOT NULL DEFAULT 'new',
    body        TEXT NOT NULL,
    created_by  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_diff_comments_run
    ON diff_review_comments (run_id, file_path);

CREATE INDEX IF NOT EXISTS idx_diff_comments_space
    ON diff_review_comments (space_id);

DO $rls$
DECLARE
    tbl text := 'diff_review_comments';
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
        GRANT SELECT, INSERT, UPDATE, DELETE ON diff_review_comments TO ash_app;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ash_rls_tester') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON diff_review_comments TO ash_rls_tester;
    END IF;
END
$$;
