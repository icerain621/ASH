CREATE TABLE IF NOT EXISTS tool_calls (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL,
    trace_id VARCHAR(64),
    step_id VARCHAR(128),
    tool VARCHAR(128) NOT NULL,
    risk VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    args_digest VARCHAR(128),
    output_json TEXT NOT NULL DEFAULT '{}',
    error TEXT,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    attempt INTEGER NOT NULL DEFAULT 1,
    timeout_ms BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_tool_calls_run_id ON tool_calls (run_id);
CREATE INDEX IF NOT EXISTS idx_tool_calls_trace_id ON tool_calls (trace_id);
CREATE INDEX IF NOT EXISTS idx_tool_calls_step_id ON tool_calls (step_id);
CREATE INDEX IF NOT EXISTS idx_tool_calls_tool ON tool_calls (tool);
CREATE INDEX IF NOT EXISTS idx_tool_calls_status ON tool_calls (status);

CREATE TABLE IF NOT EXISTS agent_tasks (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL,
    trace_id VARCHAR(64),
    step_id VARCHAR(128),
    adapter VARCHAR(64) NOT NULL,
    agent_id VARCHAR(128),
    session_id VARCHAR(128),
    action_id VARCHAR(128),
    exec_go_task_id VARCHAR(128),
    status VARCHAR(32) NOT NULL,
    prompt_digest VARCHAR(128),
    stdout_summary TEXT,
    stderr_summary TEXT,
    exit_code INTEGER,
    error_code VARCHAR(64),
    error_message TEXT,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    timeout_ms BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_agent_tasks_run_id ON agent_tasks (run_id);
CREATE INDEX IF NOT EXISTS idx_agent_tasks_trace_id ON agent_tasks (trace_id);
CREATE INDEX IF NOT EXISTS idx_agent_tasks_step_id ON agent_tasks (step_id);
CREATE INDEX IF NOT EXISTS idx_agent_tasks_action_id ON agent_tasks (action_id);
CREATE INDEX IF NOT EXISTS idx_agent_tasks_exec_go_task_id ON agent_tasks (exec_go_task_id);
CREATE INDEX IF NOT EXISTS idx_agent_tasks_status ON agent_tasks (status);

CREATE TABLE IF NOT EXISTS artifact_index (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL,
    step_id VARCHAR(128),
    type VARCHAR(64) NOT NULL,
    name VARCHAR(256),
    uri VARCHAR(1024) NOT NULL,
    store_key VARCHAR(1024),
    digest VARCHAR(128) NOT NULL,
    content_type VARCHAR(128),
    size_bytes BIGINT NOT NULL DEFAULT 0,
    event_range VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_artifact_index_run_id ON artifact_index (run_id);
CREATE INDEX IF NOT EXISTS idx_artifact_index_step_id ON artifact_index (step_id);
CREATE INDEX IF NOT EXISTS idx_artifact_index_type ON artifact_index (type);
CREATE INDEX IF NOT EXISTS idx_artifact_index_digest ON artifact_index (digest);

CREATE TABLE IF NOT EXISTS checkpoints (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL,
    step_id VARCHAR(128) NOT NULL,
    snapshot_digest VARCHAR(128) NOT NULL,
    uri VARCHAR(1024),
    store_key VARCHAR(1024),
    content_type VARCHAR(128),
    size_bytes BIGINT NOT NULL DEFAULT 0,
    strategy VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_checkpoints_run_id ON checkpoints (run_id);
CREATE INDEX IF NOT EXISTS idx_checkpoints_step_id ON checkpoints (step_id);

UPDATE schema_meta
SET value = '5', updated_at = NOW()
WHERE key = 'schema_version';
