ALTER TABLE courses
    DROP CONSTRAINT IF EXISTS courses_visibility_check;

ALTER TABLE courses
    DROP COLUMN IF EXISTS estimated_duration_minutes,
    DROP COLUMN IF EXISTS target_audience,
    DROP COLUMN IF EXISTS requirements,
    DROP COLUMN IF EXISTS learning_outcomes,
    DROP COLUMN IF EXISTS visibility;
