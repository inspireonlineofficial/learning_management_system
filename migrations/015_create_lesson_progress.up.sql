-- Create lesson_progress table
CREATE TABLE IF NOT EXISTS lesson_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enrollment_id UUID NOT NULL REFERENCES enrollments(id) ON DELETE CASCADE,
    lesson_id UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    position_seconds INT DEFAULT 0,
    watched_percent NUMERIC(5, 2) DEFAULT 0,
    completed BOOLEAN DEFAULT false,
    completed_at TIMESTAMPTZ,
    last_watched_at TIMESTAMPTZ,
    CONSTRAINT unique_enrollment_lesson UNIQUE (enrollment_id, lesson_id)
);

-- Create indexes for lesson_progress
CREATE INDEX idx_lesson_progress_enrollment_id ON lesson_progress(enrollment_id);
CREATE INDEX idx_lesson_progress_lesson_id ON lesson_progress(lesson_id);
CREATE INDEX idx_lesson_progress_completed ON lesson_progress(completed);
CREATE INDEX idx_lesson_progress_last_watched ON lesson_progress(last_watched_at);
