CREATE TABLE IF NOT EXISTS alert_rules (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    name VARCHAR(128) NOT NULL,
    metric VARCHAR(128) NOT NULL,
    condition VARCHAR(16) NOT NULL DEFAULT 'gt',
    threshold DOUBLE PRECISION NOT NULL DEFAULT 0,
    window_minutes INTEGER NOT NULL DEFAULT 60,
    severity VARCHAR(32) NOT NULL DEFAULT 'warn',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    description VARCHAR(512),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alert_rules_space_id ON alert_rules (space_id);
CREATE INDEX IF NOT EXISTS idx_alert_rules_name ON alert_rules (name);
CREATE INDEX IF NOT EXISTS idx_alert_rules_metric ON alert_rules (metric);
CREATE INDEX IF NOT EXISTS idx_alert_rules_severity ON alert_rules (severity);
CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled ON alert_rules (enabled);

CREATE TABLE IF NOT EXISTS alert_events (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    rule_id VARCHAR(64),
    rule_name VARCHAR(128),
    severity VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    target_type VARCHAR(64),
    target_id VARCHAR(128),
    fingerprint VARCHAR(128),
    message TEXT NOT NULL,
    evidence_refs_json TEXT NOT NULL DEFAULT '[]',
    triggered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alert_events_space_id ON alert_events (space_id);
CREATE INDEX IF NOT EXISTS idx_alert_events_rule_id ON alert_events (rule_id);
CREATE INDEX IF NOT EXISTS idx_alert_events_rule_name ON alert_events (rule_name);
CREATE INDEX IF NOT EXISTS idx_alert_events_severity ON alert_events (severity);
CREATE INDEX IF NOT EXISTS idx_alert_events_status ON alert_events (status);
CREATE INDEX IF NOT EXISTS idx_alert_events_target_type ON alert_events (target_type);
CREATE INDEX IF NOT EXISTS idx_alert_events_target_id ON alert_events (target_id);
CREATE INDEX IF NOT EXISTS idx_alert_events_fingerprint ON alert_events (fingerprint);
CREATE INDEX IF NOT EXISTS idx_alert_events_triggered_at ON alert_events (triggered_at);
CREATE INDEX IF NOT EXISTS idx_alert_events_resolved_at ON alert_events (resolved_at);

CREATE TABLE IF NOT EXISTS alert_silences (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    rule_id VARCHAR(64),
    reason TEXT NOT NULL,
    created_by VARCHAR(128),
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alert_silences_space_id ON alert_silences (space_id);
CREATE INDEX IF NOT EXISTS idx_alert_silences_rule_id ON alert_silences (rule_id);
CREATE INDEX IF NOT EXISTS idx_alert_silences_starts_at ON alert_silences (starts_at);
CREATE INDEX IF NOT EXISTS idx_alert_silences_ends_at ON alert_silences (ends_at);

UPDATE schema_meta
SET value = '10', updated_at = NOW()
WHERE key = 'schema_version';
