CREATE TABLE IF NOT EXISTS release_records (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    version VARCHAR(128) NOT NULL,
    title VARCHAR(256) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    canary_strategy TEXT,
    gate_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    evidence_refs_json TEXT NOT NULL DEFAULT '[]',
    created_by VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_release_records_space_id ON release_records (space_id);
CREATE INDEX IF NOT EXISTS idx_release_records_version ON release_records (version);
CREATE INDEX IF NOT EXISTS idx_release_records_status ON release_records (status);
CREATE INDEX IF NOT EXISTS idx_release_records_gate_status ON release_records (gate_status);

CREATE TABLE IF NOT EXISTS release_checklist_items (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    release_id VARCHAR(64) NOT NULL,
    item_key VARCHAR(128) NOT NULL,
    label VARCHAR(512) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    evidence_ref VARCHAR(1024),
    updated_by VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_release_checklist_items_space_id ON release_checklist_items (space_id);
CREATE INDEX IF NOT EXISTS idx_release_checklist_items_release_id ON release_checklist_items (release_id);
CREATE INDEX IF NOT EXISTS idx_release_checklist_items_item_key ON release_checklist_items (item_key);
CREATE INDEX IF NOT EXISTS idx_release_checklist_items_status ON release_checklist_items (status);

CREATE TABLE IF NOT EXISTS release_gate_results (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    release_id VARCHAR(64) NOT NULL,
    gate_key VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL,
    message TEXT NOT NULL,
    evidence_refs_json TEXT NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_release_gate_results_space_id ON release_gate_results (space_id);
CREATE INDEX IF NOT EXISTS idx_release_gate_results_release_id ON release_gate_results (release_id);
CREATE INDEX IF NOT EXISTS idx_release_gate_results_gate_key ON release_gate_results (gate_key);
CREATE INDEX IF NOT EXISTS idx_release_gate_results_status ON release_gate_results (status);

CREATE TABLE IF NOT EXISTS rollback_drills (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    release_id VARCHAR(64) NOT NULL,
    scenario VARCHAR(256) NOT NULL,
    status VARCHAR(32) NOT NULL,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    evidence_refs_json TEXT NOT NULL DEFAULT '[]',
    notes TEXT,
    created_by VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rollback_drills_space_id ON rollback_drills (space_id);
CREATE INDEX IF NOT EXISTS idx_rollback_drills_release_id ON rollback_drills (release_id);
CREATE INDEX IF NOT EXISTS idx_rollback_drills_status ON rollback_drills (status);

UPDATE schema_meta
SET value = '11', updated_at = NOW()
WHERE key = 'schema_version';
