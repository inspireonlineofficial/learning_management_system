DROP TABLE IF EXISTS course_notes;

ALTER TABLE question_options
    DROP COLUMN IF EXISTS image_url,
    DROP COLUMN IF EXISTS content_type;

ALTER TABLE questions
    DROP COLUMN IF EXISTS is_required,
    DROP COLUMN IF EXISTS marks,
    DROP COLUMN IF EXISTS image_url,
    DROP COLUMN IF EXISTS content_type;

ALTER TABLE quizzes
    DROP COLUMN IF EXISTS is_published,
    DROP COLUMN IF EXISTS is_free,
    DROP COLUMN IF EXISTS description;

ALTER TABLE lessons
    DROP COLUMN IF EXISTS is_free,
    DROP COLUMN IF EXISTS description;

ALTER TABLE modules
    DROP COLUMN IF EXISTS is_published,
    DROP COLUMN IF EXISTS is_free,
    DROP COLUMN IF EXISTS description;
