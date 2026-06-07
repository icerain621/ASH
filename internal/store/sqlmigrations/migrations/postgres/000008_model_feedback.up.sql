CREATE TABLE IF NOT EXISTS model_usage (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64),
    step_id VARCHAR(128),
    provider VARCHAR(64) NOT NULL,
    model VARCHAR(128) NOT NULL,
    input_tokens BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    cost_micros BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_model_usage_run_id ON model_usage (run_id);
CREATE INDEX IF NOT EXISTS idx_model_usage_step_id ON model_usage (step_id);
CREATE INDEX IF NOT EXISTS idx_model_usage_status ON model_usage (status);

CREATE TABLE IF NOT EXISTS quality_metrics (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64),
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    name VARCHAR(128) NOT NULL,
    value DOUBLE PRECISION NOT NULL DEFAULT 0,
    unit VARCHAR(32),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_quality_metrics_run_id ON quality_metrics (run_id);
CREATE INDEX IF NOT EXISTS idx_quality_metrics_space_id ON quality_metrics (space_id);
CREATE INDEX IF NOT EXISTS idx_quality_metrics_name ON quality_metrics (name);

CREATE TABLE IF NOT EXISTS mcp_tools (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    name VARCHAR(128) NOT NULL,
    server VARCHAR(256) NOT NULL,
    schema_json TEXT NOT NULL DEFAULT '{}',
    risk VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mcp_tools_space_id ON mcp_tools (space_id);
CREATE INDEX IF NOT EXISTS idx_mcp_tools_name ON mcp_tools (name);
CREATE INDEX IF NOT EXISTS idx_mcp_tools_status ON mcp_tools (status);

CREATE TABLE IF NOT EXISTS feedback (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    target_type VARCHAR(64) NOT NULL,
    target_id VARCHAR(128) NOT NULL,
    rating INTEGER NOT NULL DEFAULT 0,
    category VARCHAR(64) NOT NULL DEFAULT 'general',
    status VARCHAR(32) NOT NULL DEFAULT 'open',
    severity VARCHAR(32) NOT NULL DEFAULT 'info',
    source VARCHAR(64) NOT NULL DEFAULT 'ui',
    comment TEXT,
    actor_id VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_feedback_space_id ON feedback (space_id);
CREATE INDEX IF NOT EXISTS idx_feedback_target_type ON feedback (target_type);
CREATE INDEX IF NOT EXISTS idx_feedback_target_id ON feedback (target_id);
CREATE INDEX IF NOT EXISTS idx_feedback_category ON feedback (category);
CREATE INDEX IF NOT EXISTS idx_feedback_status ON feedback (status);
CREATE INDEX IF NOT EXISTS idx_feedback_severity ON feedback (severity);
CREATE INDEX IF NOT EXISTS idx_feedback_source ON feedback (source);

UPDATE schema_meta
SET value = '8', updated_at = NOW()
WHERE key = 'schema_version';
