DROP INDEX IF EXISTS index_announcements_time_range;
DROP TABLE IF EXISTS announcements;
ALTER TABLE maintenance_mode ADD COLUMN IF NOT EXISTS message TEXT NOT NULL;
