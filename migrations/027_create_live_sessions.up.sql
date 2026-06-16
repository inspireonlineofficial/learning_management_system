-- Live sessions table
-- Requirements: 16.1
CREATE TABLE IF NOT EXISTS live_sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id           UUID NOT NULL REFERENCES courses(id),
    teacher_id          UUID NOT NULL REFERENCES users(id),
    title               VARCHAR(255) NOT NULL,
    scheduled_at        TIMESTAMPTZ NOT NULL,
    duration_minutes    INT NOT NULL CHECK (duration_minutes > 0),
    status              VARCHAR(20) NOT NULL DEFAULT 'scheduled'
                            CHECK (status IN ('scheduled','live','ended','cancelled')),
    record_session      BOOLEAN NOT NULL DEFAULT false,
    attendee_count      INT NOT NULL DEFAULT 0,
    recording_rustfs_key TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_live_sessions_course_id ON live_sessions(course_id);
CREATE INDEX IF NOT EXISTS idx_live_sessions_teacher_id ON live_sessions(teacher_id);
CREATE INDEX IF NOT EXISTS idx_live_sessions_status ON live_sessions(status);

-- Attendance table
-- Requirements: 16.3, 16.7
CREATE TABLE IF NOT EXISTS attendance (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id       UUID NOT NULL REFERENCES live_sessions(id),
    student_id       UUID NOT NULL REFERENCES users(id),
    joined_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at          TIMESTAMPTZ,
    duration_minutes INT NOT NULL DEFAULT 0,
    UNIQUE(session_id, student_id)
);

CREATE INDEX IF NOT EXISTS idx_attendance_session_id ON attendance(session_id);
CREATE INDEX IF NOT EXISTS idx_attendance_student_id ON attendance(student_id);
