-- Links using an API key have no JWT nonce, so give them a random one. Their webhook
-- tokens stop working and must be reset.
DROP INDEX IF EXISTS index_workspace_vcs_provider_links_on_token_hash;
UPDATE workspace_vcs_provider_links SET token_nonce = gen_random_uuid() WHERE token_nonce IS NULL;
ALTER TABLE workspace_vcs_provider_links ALTER COLUMN token_nonce SET NOT NULL;
ALTER TABLE workspace_vcs_provider_links DROP COLUMN IF EXISTS token_hash;

-- API key tokens can't be converted back to JWTs, so they're removed.
DELETE FROM scim_tokens;

DROP INDEX IF EXISTS index_scim_tokens_on_token_hash;
ALTER TABLE scim_tokens DROP COLUMN IF EXISTS expires_at;
ALTER TABLE scim_tokens DROP COLUMN IF EXISTS idp_issuer_url;
ALTER TABLE scim_tokens DROP COLUMN IF EXISTS token_hash;

ALTER TABLE scim_tokens ADD COLUMN IF NOT EXISTS nonce UUID NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS index_scim_tokens_on_nonce ON scim_tokens(nonce);
