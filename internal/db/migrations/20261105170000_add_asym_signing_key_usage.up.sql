-- Tag every signing key with what it signs, so a later key usage (audit checkpoint keys) can share this table.
-- Every existing key signs JWTs. The signing key manager only queries 'jws' keys, so instances running this
-- version keep ignoring checkpoint keys while a release that adds them rolls out. IF NOT EXISTS keeps this
-- compatible with the audit log migration, which adds the same column.
ALTER TABLE asym_signing_keys ADD COLUMN IF NOT EXISTS key_usage VARCHAR NOT NULL DEFAULT 'jws';
