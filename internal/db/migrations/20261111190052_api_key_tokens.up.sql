-- SCIM tokens: replace the JWT nonce with a hash of an opaque API key. Existing tokens are
-- JWTs that can no longer be verified, so they're removed and must be regenerated.
DELETE FROM scim_tokens;

DROP INDEX IF EXISTS index_scim_tokens_on_nonce;
ALTER TABLE scim_tokens DROP COLUMN IF EXISTS nonce;

ALTER TABLE scim_tokens ADD COLUMN IF NOT EXISTS token_hash VARCHAR NOT NULL;
ALTER TABLE scim_tokens ADD COLUMN IF NOT EXISTS idp_issuer_url VARCHAR NOT NULL;
ALTER TABLE scim_tokens ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP;
CREATE UNIQUE INDEX IF NOT EXISTS index_scim_tokens_on_token_hash ON scim_tokens(token_hash);

-- VCS provider links: new links store a hash of an opaque API key. Legacy links keep their
-- JWT nonce until the webhook token is reset.
ALTER TABLE workspace_vcs_provider_links ADD COLUMN IF NOT EXISTS token_hash VARCHAR;
ALTER TABLE workspace_vcs_provider_links ALTER COLUMN token_nonce DROP NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS index_workspace_vcs_provider_links_on_token_hash ON workspace_vcs_provider_links(token_hash);
