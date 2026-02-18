ALTER TABLE service_accounts ADD COLUMN client_secret_hash VARCHAR;
ALTER TABLE service_accounts ADD COLUMN client_secret_expires_at TIMESTAMP;
ALTER TABLE service_accounts ADD COLUMN secret_expiration_email_sent_at TIMESTAMP;
