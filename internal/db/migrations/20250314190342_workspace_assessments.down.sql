ALTER TABLE workspaces
    DROP COLUMN IF EXISTS drift_detection_enabled;

ALTER TABLE groups
    DROP COLUMN IF EXISTS drift_detection_enabled;

ALTER TABLE runs
    DROP COLUMN IF EXISTS is_assessment_run;

DROP TABLE IF EXISTS workspace_assessments;
