ALTER TABLE modules
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS is_free BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS is_published BOOLEAN NOT NULL DEFAULT true;

ALTER TABLE lessons
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS is_free BOOLEAN NOT NULL DEFAULT true;

ALTER TABLE quizzes
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS is_free BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS is_published BOOLEAN NOT NULL DEFAULT true;

ALTER TABLE questions
    ADD COLUMN IF NOT EXISTS content_type VARCHAR(20) NOT NULL DEFAULT 'text'
        CHECK (content_type IN ('text', 'image', 'text_image')),
    ADD COLUMN IF NOT EXISTS image_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS marks NUMERIC(10, 2) NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS is_required BOOLEAN NOT NULL DEFAULT true;

ALTER TABLE question_options
    ADD COLUMN IF NOT EXISTS content_type VARCHAR(20) NOT NULL DEFAULT 'text'
        CHECK (content_type IN ('text', 'image', 'text_image')),
    ADD COLUMN IF NOT EXISTS image_url TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS course_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    module_id UUID REFERENCES modules(id) ON DELETE CASCADE,
    lesson_id UUID REFERENCES lessons(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    file_url TEXT NOT NULL DEFAULT '',
    is_free BOOLEAN NOT NULL DEFAULT true,
    is_published BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CHECK (module_id IS NOT NULL OR lesson_id IS NOT NULL OR course_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_course_notes_course_id
    ON course_notes(course_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_course_notes_module_id
    ON course_notes(module_id)
    WHERE module_id IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_course_notes_lesson_id
    ON course_notes(lesson_id)
    WHERE lesson_id IS NOT NULL AND deleted_at IS NULL;
