CREATE TABLE IF NOT EXISTS repo_connections (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    provider VARCHAR(32) NOT NULL,
    owner VARCHAR(128) NOT NULL,
    repo VARCHAR(128) NOT NULL,
    default_branch VARCHAR(128) NOT NULL DEFAULT 'main',
    secret_id VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    last_sync_at TIMESTAMPTZ,
    created_by VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_repo_connections_space_id ON repo_connections (space_id);
CREATE INDEX IF NOT EXISTS idx_repo_connections_provider ON repo_connections (provider);
CREATE INDEX IF NOT EXISTS idx_repo_connections_owner ON repo_connections (owner);
CREATE INDEX IF NOT EXISTS idx_repo_connections_repo ON repo_connections (repo);
CREATE INDEX IF NOT EXISTS idx_repo_connections_secret_id ON repo_connections (secret_id);
CREATE INDEX IF NOT EXISTS idx_repo_connections_status ON repo_connections (status);

CREATE TABLE IF NOT EXISTS ci_runs (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    connection_id VARCHAR(64) NOT NULL,
    provider_run_id VARCHAR(128) NOT NULL,
    workflow VARCHAR(256),
    status VARCHAR(32) NOT NULL,
    conclusion VARCHAR(32),
    attempt INTEGER NOT NULL DEFAULT 1,
    commit_sha VARCHAR(64),
    branch VARCHAR(256),
    run_url VARCHAR(1024),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ci_runs_space_id ON ci_runs (space_id);
CREATE INDEX IF NOT EXISTS idx_ci_runs_connection_id ON ci_runs (connection_id);
CREATE INDEX IF NOT EXISTS idx_ci_runs_provider_run_id ON ci_runs (provider_run_id);
CREATE INDEX IF NOT EXISTS idx_ci_runs_workflow ON ci_runs (workflow);
CREATE INDEX IF NOT EXISTS idx_ci_runs_status ON ci_runs (status);
CREATE INDEX IF NOT EXISTS idx_ci_runs_conclusion ON ci_runs (conclusion);
CREATE INDEX IF NOT EXISTS idx_ci_runs_attempt ON ci_runs (attempt);
CREATE INDEX IF NOT EXISTS idx_ci_runs_commit_sha ON ci_runs (commit_sha);
CREATE INDEX IF NOT EXISTS idx_ci_runs_branch ON ci_runs (branch);
CREATE INDEX IF NOT EXISTS idx_ci_runs_started_at ON ci_runs (started_at);
CREATE INDEX IF NOT EXISTS idx_ci_runs_completed_at ON ci_runs (completed_at);

CREATE TABLE IF NOT EXISTS ci_jobs (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    connection_id VARCHAR(64) NOT NULL,
    ci_run_id VARCHAR(64) NOT NULL,
    provider_job_id VARCHAR(128) NOT NULL,
    name VARCHAR(256),
    status VARCHAR(32) NOT NULL,
    conclusion VARCHAR(32),
    attempt INTEGER NOT NULL DEFAULT 1,
    log_digest VARCHAR(128),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ci_jobs_space_id ON ci_jobs (space_id);
CREATE INDEX IF NOT EXISTS idx_ci_jobs_connection_id ON ci_jobs (connection_id);
CREATE INDEX IF NOT EXISTS idx_ci_jobs_ci_run_id ON ci_jobs (ci_run_id);
CREATE INDEX IF NOT EXISTS idx_ci_jobs_provider_job_id ON ci_jobs (provider_job_id);
CREATE INDEX IF NOT EXISTS idx_ci_jobs_name ON ci_jobs (name);
CREATE INDEX IF NOT EXISTS idx_ci_jobs_status ON ci_jobs (status);
CREATE INDEX IF NOT EXISTS idx_ci_jobs_conclusion ON ci_jobs (conclusion);
CREATE INDEX IF NOT EXISTS idx_ci_jobs_attempt ON ci_jobs (attempt);
CREATE INDEX IF NOT EXISTS idx_ci_jobs_log_digest ON ci_jobs (log_digest);
CREATE INDEX IF NOT EXISTS idx_ci_jobs_started_at ON ci_jobs (started_at);
CREATE INDEX IF NOT EXISTS idx_ci_jobs_completed_at ON ci_jobs (completed_at);

CREATE TABLE IF NOT EXISTS ci_diagnoses (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    connection_id VARCHAR(64),
    ci_run_id VARCHAR(64),
    ci_job_id VARCHAR(64),
    status VARCHAR(32) NOT NULL,
    root_cause VARCHAR(128) NOT NULL,
    fix_suggestions_json TEXT NOT NULL DEFAULT '[]',
    evidence_refs_json TEXT NOT NULL DEFAULT '[]',
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    adopted BOOLEAN NOT NULL DEFAULT FALSE,
    decision_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    decision_reason TEXT,
    decided_by VARCHAR(128),
    decided_at TIMESTAMPTZ,
    log_digest VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ci_diagnoses_space_id ON ci_diagnoses (space_id);
CREATE INDEX IF NOT EXISTS idx_ci_diagnoses_connection_id ON ci_diagnoses (connection_id);
CREATE INDEX IF NOT EXISTS idx_ci_diagnoses_ci_run_id ON ci_diagnoses (ci_run_id);
CREATE INDEX IF NOT EXISTS idx_ci_diagnoses_ci_job_id ON ci_diagnoses (ci_job_id);
CREATE INDEX IF NOT EXISTS idx_ci_diagnoses_status ON ci_diagnoses (status);
CREATE INDEX IF NOT EXISTS idx_ci_diagnoses_root_cause ON ci_diagnoses (root_cause);
CREATE INDEX IF NOT EXISTS idx_ci_diagnoses_adopted ON ci_diagnoses (adopted);
CREATE INDEX IF NOT EXISTS idx_ci_diagnoses_decision_status ON ci_diagnoses (decision_status);
CREATE INDEX IF NOT EXISTS idx_ci_diagnoses_decided_at ON ci_diagnoses (decided_at);
CREATE INDEX IF NOT EXISTS idx_ci_diagnoses_log_digest ON ci_diagnoses (log_digest);

UPDATE schema_meta
SET value = '9', updated_at = NOW()
WHERE key = 'schema_version';
