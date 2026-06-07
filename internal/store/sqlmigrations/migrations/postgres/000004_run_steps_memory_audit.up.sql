CREATE TABLE IF NOT EXISTS run_steps (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL,
    step_id VARCHAR(128) NOT NULL,
    step_order INTEGER NOT NULL,
    role VARCHAR(64),
    kind VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    error_code VARCHAR(64),
    error_message VARCHAR(1024),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_run_steps_run_order ON run_steps (run_id, step_order);
CREATE INDEX IF NOT EXISTS idx_run_steps_status ON run_steps (status);
CREATE INDEX IF NOT EXISTS idx_run_steps_started_at ON run_steps (started_at);
CREATE INDEX IF NOT EXISTS idx_run_steps_finished_at ON run_steps (finished_at);

CREATE TABLE IF NOT EXISTS memory_records (
    id VARCHAR(64) PRIMARY KEY,
    layer VARCHAR(8) NOT NULL,
    status VARCHAR(32) NOT NULL,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    schema_version INTEGER NOT NULL DEFAULT 1,
    title VARCHAR(512) NOT NULL,
    body TEXT NOT NULL,
    scope_repo VARCHAR(512),
    scope_team VARCHAR(128),
    tags_json TEXT NOT NULL DEFAULT '[]',
    ttl_days INTEGER,
    sensitivity VARCHAR(32) NOT NULL DEFAULT 'normal',
    dedupe_key VARCHAR(128),
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_memory_layer_status ON memory_records (layer, status);
CREATE INDEX IF NOT EXISTS idx_memory_records_space_id ON memory_records (space_id);
CREATE INDEX IF NOT EXISTS idx_memory_records_scope_repo ON memory_records (scope_repo);
CREATE INDEX IF NOT EXISTS idx_memory_records_dedupe_key ON memory_records (dedupe_key);

CREATE TABLE IF NOT EXISTS audit_log (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    trace_id VARCHAR(64),
    run_id VARCHAR(64),
    actor_id VARCHAR(128),
    event_type VARCHAR(128) NOT NULL,
    payload_json TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_space_id ON audit_log (space_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_trace_id ON audit_log (trace_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_run_id ON audit_log (run_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_event_type ON audit_log (event_type);

UPDATE schema_meta
SET value = '4', updated_at = NOW()
WHERE key = 'schema_version';
