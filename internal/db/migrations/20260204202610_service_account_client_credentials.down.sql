ALTER TABLE service_accounts DROP COLUMN IF EXISTS secret_expiration_email_sent_at;
ALTER TABLE service_accounts DROP COLUMN IF EXISTS client_secret_expires_at;
ALTER TABLE service_accounts DROP COLUMN IF EXISTS client_secret_hash;
