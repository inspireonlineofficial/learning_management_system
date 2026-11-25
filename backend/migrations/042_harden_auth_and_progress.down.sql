ALTER TABLE lesson_progress DROP COLUMN IF EXISTS started_at;
DROP INDEX IF EXISTS idx_users_email_lower;
