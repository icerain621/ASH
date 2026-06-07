CREATE TABLE IF NOT EXISTS memory_evidence (
    id VARCHAR(64) PRIMARY KEY,
    memory_id VARCHAR(64) NOT NULL,
    kind VARCHAR(16) NOT NULL,
    ref VARCHAR(1024) NOT NULL,
    digest VARCHAR(128),
    meta_json TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_memory_evidence_memory_id ON memory_evidence (memory_id);

CREATE TABLE IF NOT EXISTS memory_reviews (
    id VARCHAR(64) PRIMARY KEY,
    memory_id VARCHAR(64) NOT NULL,
    decision VARCHAR(32) NOT NULL,
    reviewer_id VARCHAR(128),
    reason TEXT NOT NULL,
    policy_profile VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_memory_reviews_memory_id ON memory_reviews (memory_id);

CREATE TABLE IF NOT EXISTS memory_edges (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    from_id VARCHAR(64) NOT NULL,
    to_id VARCHAR(64) NOT NULL,
    kind VARCHAR(32) NOT NULL,
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_memory_edges_space_id ON memory_edges (space_id);
CREATE INDEX IF NOT EXISTS idx_memory_edges_from_id ON memory_edges (from_id);
CREATE INDEX IF NOT EXISTS idx_memory_edges_to_id ON memory_edges (to_id);
CREATE INDEX IF NOT EXISTS idx_memory_edges_kind ON memory_edges (kind);

CREATE TABLE IF NOT EXISTS rag_documents (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    repo_root VARCHAR(512) NOT NULL,
    path VARCHAR(1024) NOT NULL,
    digest VARCHAR(128) NOT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rag_documents_space_id ON rag_documents (space_id);
CREATE INDEX IF NOT EXISTS idx_rag_documents_repo_root ON rag_documents (repo_root);
CREATE INDEX IF NOT EXISTS idx_rag_documents_path ON rag_documents (path);
CREATE INDEX IF NOT EXISTS idx_rag_documents_digest ON rag_documents (digest);

CREATE TABLE IF NOT EXISTS rag_chunks (
    id VARCHAR(64) PRIMARY KEY,
    document_id VARCHAR(64) NOT NULL,
    space_id VARCHAR(64) NOT NULL DEFAULT 'local',
    repo_root VARCHAR(512) NOT NULL,
    path VARCHAR(1024) NOT NULL,
    symbol VARCHAR(256),
    start_line INTEGER NOT NULL DEFAULT 0,
    end_line INTEGER NOT NULL DEFAULT 0,
    text TEXT NOT NULL,
    digest VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rag_chunks_document_id ON rag_chunks (document_id);
CREATE INDEX IF NOT EXISTS idx_rag_chunks_space_id ON rag_chunks (space_id);
CREATE INDEX IF NOT EXISTS idx_rag_chunks_repo_root ON rag_chunks (repo_root);
CREATE INDEX IF NOT EXISTS idx_rag_chunks_path ON rag_chunks (path);
CREATE INDEX IF NOT EXISTS idx_rag_chunks_symbol ON rag_chunks (symbol);
CREATE INDEX IF NOT EXISTS idx_rag_chunks_digest ON rag_chunks (digest);

UPDATE schema_meta
SET value = '6', updated_at = NOW()
WHERE key = 'schema_version';
