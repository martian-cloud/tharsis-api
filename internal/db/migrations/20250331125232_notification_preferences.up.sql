CREATE TABLE IF NOT EXISTS notification_preferences (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID NOT NULL,
    namespace_id UUID,
    scope VARCHAR(255) NOT NULL,
    custom_events JSONB,
    CONSTRAINT fk_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_namespace_id FOREIGN KEY(namespace_id) REFERENCES namespaces(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX index_user_id_namespace_id ON notification_preferences(user_id, namespace_id) NULLS NOT DISTINCT;
