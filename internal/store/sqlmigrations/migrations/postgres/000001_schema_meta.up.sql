CREATE TABLE IF NOT EXISTS schema_meta (
    key VARCHAR(64) PRIMARY KEY,
    value VARCHAR(256) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO schema_meta (key, value, updated_at)
VALUES ('schema_version', '1', NOW())
ON CONFLICT (key) DO NOTHING;
