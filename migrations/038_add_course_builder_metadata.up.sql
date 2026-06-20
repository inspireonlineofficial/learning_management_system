ALTER TABLE courses
    ADD COLUMN IF NOT EXISTS visibility VARCHAR(20) NOT NULL DEFAULT 'public',
    ADD COLUMN IF NOT EXISTS learning_outcomes TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS requirements TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS target_audience TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS estimated_duration_minutes INT NOT NULL DEFAULT 0;

ALTER TABLE courses
    DROP CONSTRAINT IF EXISTS courses_visibility_check;

ALTER TABLE courses
    ADD CONSTRAINT courses_visibility_check
    CHECK (visibility IN ('public', 'unlisted', 'private'));
