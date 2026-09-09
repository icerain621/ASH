DROP INDEX IF EXISTS uidx_users_oidc_issuer_subject;
ALTER TABLE users DROP COLUMN IF EXISTS oidc_subject;
ALTER TABLE users DROP COLUMN IF EXISTS oidc_issuer;
