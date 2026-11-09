-- Documentation processing status: 'not_started', 'pending', 'completed', or 'failed'.
ALTER TABLE terraform_provider_versions
    ADD COLUMN IF NOT EXISTS docs_processing_status VARCHAR NOT NULL DEFAULT 'not_started';

-- Error message set when documentation processing fails.
ALTER TABLE terraform_provider_versions
    ADD COLUMN IF NOT EXISTS docs_processing_error_message VARCHAR;

-- One row per documentation file of a provider version; file content is stored in object storage.
CREATE TABLE IF NOT EXISTS terraform_provider_doc_items (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    provider_version_id UUID NOT NULL,
    filename VARCHAR NOT NULL,
    category VARCHAR NOT NULL,
    subcategory VARCHAR,
    CONSTRAINT fk_provider_version_id FOREIGN KEY(provider_version_id) REFERENCES terraform_provider_versions(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS index_terraform_provider_doc_items_on_version_category_filename ON terraform_provider_doc_items(provider_version_id, category, filename);
