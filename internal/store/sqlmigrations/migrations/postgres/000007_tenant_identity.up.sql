CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(64) PRIMARY KEY,
    email VARCHAR(256),
    display_name VARCHAR(256),
    password_hash VARCHAR(256),
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_status ON users (status);

CREATE TABLE IF NOT EXISTS orgs (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(256) NOT NULL,
    slug VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_orgs_slug ON orgs (slug);

CREATE TABLE IF NOT EXISTS spaces (
    id VARCHAR(64) PRIMARY KEY,
    org_id VARCHAR(64) NOT NULL,
    name VARCHAR(256) NOT NULL,
    slug VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_spaces_org_id ON spaces (org_id);
CREATE INDEX IF NOT EXISTS idx_spaces_slug ON spaces (slug);

CREATE TABLE IF NOT EXISTS roles (
    id VARCHAR(64) PRIMARY KEY,
    org_id VARCHAR(64),
    name VARCHAR(128) NOT NULL,
    permissions TEXT NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_roles_org_id ON roles (org_id);
CREATE INDEX IF NOT EXISTS idx_roles_name ON roles (name);

CREATE TABLE IF NOT EXISTS members (
    id VARCHAR(64) PRIMARY KEY,
    org_id VARCHAR(64) NOT NULL,
    space_id VARCHAR(64),
    user_id VARCHAR(64) NOT NULL,
    role_id VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_members_org_id ON members (org_id);
CREATE INDEX IF NOT EXISTS idx_members_space_id ON members (space_id);
CREATE INDEX IF NOT EXISTS idx_members_user_id ON members (user_id);
CREATE INDEX IF NOT EXISTS idx_members_role_id ON members (role_id);
CREATE INDEX IF NOT EXISTS idx_members_status ON members (status);

CREATE TABLE IF NOT EXISTS resource_scopes (
    id VARCHAR(64) PRIMARY KEY,
    space_id VARCHAR(64) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    resource_id VARCHAR(128) NOT NULL,
    policy_json TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_resource_scopes_space_id ON resource_scopes (space_id);
CREATE INDEX IF NOT EXISTS idx_resource_scopes_resource_type ON resource_scopes (resource_type);
CREATE INDEX IF NOT EXISTS idx_resource_scopes_resource_id ON resource_scopes (resource_id);

CREATE TABLE IF NOT EXISTS audit_policies (
    space_id VARCHAR(64) PRIMARY KEY,
    retention_days INTEGER NOT NULL DEFAULT 365,
    redact_payload BOOLEAN NOT NULL DEFAULT FALSE,
    locked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

UPDATE schema_meta
SET value = '7', updated_at = NOW()
WHERE key = 'schema_version';
