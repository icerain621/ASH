ALTER TABLE spaces DROP CONSTRAINT IF EXISTS chk_spaces_kind;
DROP INDEX IF EXISTS idx_spaces_kind;
ALTER TABLE spaces DROP COLUMN IF EXISTS kind;

UPDATE schema_meta
SET value = '34', updated_at = NOW()
WHERE key = 'schema_version';
