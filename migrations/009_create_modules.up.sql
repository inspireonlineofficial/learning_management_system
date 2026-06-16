-- Create modules table
CREATE TABLE IF NOT EXISTS modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    position INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Create index for modules
CREATE UNIQUE INDEX idx_modules_course_position_unique ON modules(course_id, position) WHERE deleted_at IS NULL;
CREATE INDEX idx_modules_course_id ON modules(course_id) WHERE deleted_at IS NULL;
