ALTER TABLE runs DROP COLUMN IF EXISTS tool_allowlist_json;
ALTER TABLE runs DROP COLUMN IF EXISTS depth;
ALTER TABLE runs DROP COLUMN IF EXISTS root_run_id;
ALTER TABLE runs DROP COLUMN IF EXISTS parent_run_id;
DROP INDEX IF EXISTS idx_runs_parent_run_id;
DROP INDEX IF EXISTS idx_runs_root_run_id;
