-- Add homepage_config JSONB column to system_settings so public frontend
-- sections (course tracks, bookshop categories, featured copy, outcomes,
-- info panels, footer, bookshop subjects) are configurable by admins.
ALTER TABLE system_settings
    ADD COLUMN IF NOT EXISTS homepage_config JSONB NOT NULL DEFAULT '{}'::jsonb;
