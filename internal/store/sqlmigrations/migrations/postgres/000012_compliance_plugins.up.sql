CREATE TABLE IF NOT EXISTS secret_records (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    name VARCHAR(128) NOT NULL,
    description VARCHAR(512),
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    scope_json TEXT NOT NULL DEFAULT '{}',
    value_ciphertext TEXT NOT NULL,
    value_digest VARCHAR(128) NOT NULL,
    created_by VARCHAR(128),
    updated_by VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_secret_space_name ON secret_records (space_id, name);
CREATE INDEX IF NOT EXISTS idx_secret_records_status ON secret_records (status);
CREATE INDEX IF NOT EXISTS idx_secret_records_value_digest ON secret_records (value_digest);

CREATE TABLE IF NOT EXISTS approval_requests (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    run_id VARCHAR(64) NOT NULL,
    trace_id VARCHAR(64),
    step_id VARCHAR(128) NOT NULL,
    gate VARCHAR(64) NOT NULL,
    risk VARCHAR(32),
    reason TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    requested_by VARCHAR(128),
    decided_by VARCHAR(128),
    decision_reason TEXT,
    evidence_json TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    decided_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_approval_requests_space_id ON approval_requests (space_id);
CREATE INDEX IF NOT EXISTS idx_approval_requests_run_id ON approval_requests (run_id);
CREATE INDEX IF NOT EXISTS idx_approval_requests_trace_id ON approval_requests (trace_id);
CREATE INDEX IF NOT EXISTS idx_approval_requests_step_id ON approval_requests (step_id);
CREATE INDEX IF NOT EXISTS idx_approval_requests_gate ON approval_requests (gate);
CREATE INDEX IF NOT EXISTS idx_approval_requests_status ON approval_requests (status);

CREATE TABLE IF NOT EXISTS audit_exports (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL,
    uri VARCHAR(1024),
    store_key VARCHAR(1024),
    digest VARCHAR(128),
    content_type VARCHAR(128),
    size_bytes BIGINT NOT NULL DEFAULT 0,
    requested_by VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_audit_exports_space_id ON audit_exports (space_id);
CREATE INDEX IF NOT EXISTS idx_audit_exports_status ON audit_exports (status);
CREATE INDEX IF NOT EXISTS idx_audit_exports_digest ON audit_exports (digest);

CREATE TABLE IF NOT EXISTS plugin_registry (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    name VARCHAR(128) NOT NULL,
    version VARCHAR(64) NOT NULL,
    protocol VARCHAR(32) NOT NULL DEFAULT 'grpc',
    abi VARCHAR(64) NOT NULL DEFAULT 'ash.plugin.v1',
    endpoint VARCHAR(512),
    capabilities TEXT NOT NULL DEFAULT '[]',
    compatible BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(32) NOT NULL,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_plugin_registry_space_id ON plugin_registry (space_id);
CREATE INDEX IF NOT EXISTS idx_plugin_registry_name ON plugin_registry (name);
CREATE INDEX IF NOT EXISTS idx_plugin_registry_protocol ON plugin_registry (protocol);
CREATE INDEX IF NOT EXISTS idx_plugin_registry_abi ON plugin_registry (abi);
CREATE INDEX IF NOT EXISTS idx_plugin_registry_status ON plugin_registry (status);

CREATE TABLE IF NOT EXISTS improve_proposals (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    title VARCHAR(256) NOT NULL,
    description TEXT,
    baseline_run_id VARCHAR(64),
    experiment_run_id VARCHAR(64),
    status VARCHAR(32) NOT NULL,
    change_summary TEXT,
    canary_percent INTEGER NOT NULL DEFAULT 0,
    compare_json TEXT NOT NULL DEFAULT '{}',
    actor_id VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_improve_proposals_space_id ON improve_proposals (space_id);
CREATE INDEX IF NOT EXISTS idx_improve_proposals_baseline_run_id ON improve_proposals (baseline_run_id);
CREATE INDEX IF NOT EXISTS idx_improve_proposals_experiment_run_id ON improve_proposals (experiment_run_id);
CREATE INDEX IF NOT EXISTS idx_improve_proposals_status ON improve_proposals (status);

UPDATE schema_meta
SET value = '12', updated_at = NOW()
WHERE key = 'schema_version';
