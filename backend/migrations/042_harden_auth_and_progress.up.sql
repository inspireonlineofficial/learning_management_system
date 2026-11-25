-- Case-insensitive email lookups (login, registration duplicate check,
-- password reset) now use LOWER(email); index the expression so these stay
-- index-backed.
CREATE INDEX IF NOT EXISTS idx_users_email_lower ON users (LOWER(email)) WHERE deleted_at IS NULL;

-- Server-side watch-time plausibility: record when a student first started a
-- lesson so a video lesson cannot be marked complete seconds after opening it.
-- Rows that existed before this migration are grandfathered (epoch) so no
-- student loses the ability to finish a lesson they were already watching.
ALTER TABLE lesson_progress ADD COLUMN IF NOT EXISTS started_at TIMESTAMPTZ;
UPDATE lesson_progress SET started_at = TIMESTAMPTZ '1970-01-01 00:00:00+00' WHERE started_at IS NULL;
ALTER TABLE lesson_progress ALTER COLUMN started_at SET DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE lesson_progress ALTER COLUMN started_at SET NOT NULL;
