ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS drift_detection_enabled BOOLEAN;

ALTER TABLE workspaces
    ADD COLUMN IF NOT EXISTS drift_detection_enabled BOOLEAN;

ALTER TABLE runs
    ADD COLUMN IF NOT EXISTS is_assessment_run BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS workspace_assessments (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    workspace_id UUID NOT NULL,
    has_drift BOOLEAN NOT NULL,
    requires_notification BOOLEAN NOT NULL,
    completed_run_id UUID,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    CONSTRAINT fk_workspace_id FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    CONSTRAINT fk_completed_run_id FOREIGN KEY(completed_run_id) REFERENCES runs(id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX index_workspace_assessments_on_workspace_id ON workspace_assessments(workspace_id);
CREATE INDEX index_workspace_assessments_on_completed_at ON workspace_assessments(completed_at);
