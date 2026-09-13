-- GV07: spaces.kind = user|team (no Team table; Space is the tenant unit).

ALTER TABLE spaces
    ADD COLUMN IF NOT EXISTS kind VARCHAR(16) NOT NULL DEFAULT 'team';

UPDATE spaces SET kind = 'team' WHERE kind IS NULL OR kind = '';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_spaces_kind'
    ) THEN
        ALTER TABLE spaces
            ADD CONSTRAINT chk_spaces_kind CHECK (kind IN ('user', 'team'));
    END IF;
END
$$;

CREATE INDEX IF NOT EXISTS idx_spaces_kind ON spaces (kind);

UPDATE schema_meta
SET value = '35', updated_at = NOW()
WHERE key = 'schema_version';
