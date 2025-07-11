CREATE TABLE IF NOT EXISTS announcements (
    id UUID PRIMARY KEY,
    version INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    message TEXT NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    created_by VARCHAR NOT NULL,
    type VARCHAR NOT NULL,
    dismissible BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX index_announcements_time_range ON announcements (start_time, end_time);

ALTER TABLE maintenance_mode
    DROP COLUMN IF EXISTS message;
