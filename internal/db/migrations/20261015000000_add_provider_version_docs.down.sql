DROP TABLE IF EXISTS terraform_provider_doc_items;

ALTER TABLE terraform_provider_versions DROP COLUMN IF EXISTS docs_processing_error_message;
ALTER TABLE terraform_provider_versions DROP COLUMN IF EXISTS docs_processing_status;
