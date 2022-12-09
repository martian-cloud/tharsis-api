ALTER TABLE workspaces
    ADD COLUMN IF NOT EXISTS prevent_destroy_plan BOOLEAN DEFAULT false;
