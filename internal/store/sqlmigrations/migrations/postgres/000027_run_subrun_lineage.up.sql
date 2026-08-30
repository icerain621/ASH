-- Run sub-run lineage columns (v2 Sprint DO). No new RLS table.
ALTER TABLE runs ADD COLUMN IF NOT EXISTS parent_run_id TEXT;
ALTER TABLE runs ADD COLUMN IF NOT EXISTS root_run_id TEXT;
ALTER TABLE runs ADD COLUMN IF NOT EXISTS depth INTEGER NOT NULL DEFAULT 0;
ALTER TABLE runs ADD COLUMN IF NOT EXISTS tool_allowlist_json TEXT;

CREATE INDEX IF NOT EXISTS idx_runs_parent_run_id ON runs (parent_run_id);
CREATE INDEX IF NOT EXISTS idx_runs_root_run_id ON runs (root_run_id);
