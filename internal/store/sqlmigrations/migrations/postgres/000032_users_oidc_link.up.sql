-- DX56: durable OIDC issuer/subject link on users (no new table).
ALTER TABLE users ADD COLUMN IF NOT EXISTS oidc_issuer VARCHAR(512);
ALTER TABLE users ADD COLUMN IF NOT EXISTS oidc_subject VARCHAR(256);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_users_oidc_issuer_subject
  ON users (oidc_issuer, oidc_subject)
  WHERE oidc_subject IS NOT NULL AND oidc_subject <> '';
