CREATE TABLE IF NOT EXISTS runs (
    id VARCHAR(64) PRIMARY KEY,
    trace_id VARCHAR(64) NOT NULL,
    scenario_name VARCHAR(128) NOT NULL,
    scenario_version VARCHAR(64) NOT NULL,
    policy_profile VARCHAR(64) NOT NULL DEFAULT 'default',
    status VARCHAR(32) NOT NULL,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    actor_role VARCHAR(64) NOT NULL DEFAULT 'maintainer',
    inputs_digest VARCHAR(128),
    repo_root VARCHAR(512),
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    recovered BOOLEAN NOT NULL DEFAULT FALSE,
    error_code VARCHAR(64),
    error_message VARCHAR(1024),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_runs_trace_id ON runs (trace_id);
CREATE INDEX IF NOT EXISTS idx_runs_status ON runs (status);
CREATE INDEX IF NOT EXISTS idx_runs_space_id ON runs (space_id);
CREATE INDEX IF NOT EXISTS idx_runs_finished_at ON runs (finished_at);

CREATE TABLE IF NOT EXISTS run_events (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL,
    seq BIGINT NOT NULL,
    ts BIGINT NOT NULL,
    type VARCHAR(128) NOT NULL,
    severity VARCHAR(16) NOT NULL DEFAULT 'info',
    payload_json TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_run_events_run_id_seq ON run_events (run_id, seq);
CREATE INDEX IF NOT EXISTS idx_run_events_run_id ON run_events (run_id);
CREATE INDEX IF NOT EXISTS idx_run_events_type ON run_events (type);

UPDATE schema_meta
SET value = '3', updated_at = NOW()
WHERE key = 'schema_version';
