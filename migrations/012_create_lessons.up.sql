-- Create lessons table
CREATE TABLE IF NOT EXISTS lessons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_id UUID NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('video', 'text', 'attachment')),
    video_id UUID REFERENCES videos(id) ON DELETE SET NULL,
    duration_seconds INT DEFAULT 0,
    is_free_preview BOOLEAN DEFAULT false,
    is_downloadable BOOLEAN DEFAULT false,
    position INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Create indexes for lessons
CREATE INDEX idx_lessons_chapter_id ON lessons(chapter_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_lessons_status ON lessons(status) WHERE deleted_at IS NULL;
