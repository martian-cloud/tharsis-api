CREATE TABLE namespace_favorites (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID NOT NULL,
    group_id UUID,
    workspace_id UUID,
    CONSTRAINT fk_user_id FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_group_id FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX index_namespace_favorites_on_user_id_group_id ON namespace_favorites(user_id, group_id) WHERE group_id IS NOT NULL;
CREATE UNIQUE INDEX index_namespace_favorites_on_user_id_workspace_id ON namespace_favorites(user_id, workspace_id) WHERE workspace_id IS NOT NULL;