-- Feedback run_id for evolution-plane uniqueness (Sprint DY).
ALTER TABLE feedback ADD COLUMN IF NOT EXISTS run_id TEXT;
CREATE INDEX IF NOT EXISTS idx_feedback_run_id ON feedback (run_id);

UPDATE schema_meta
SET value = '22', updated_at = NOW()
WHERE key = 'schema_version';
