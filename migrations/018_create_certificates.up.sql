-- Migration: 018_create_certificates
-- Creates the certificates table for the Certificates bounded context.
-- Requirements: 18.1

CREATE TABLE IF NOT EXISTS certificates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id      UUID NOT NULL REFERENCES users(id),
    course_id       UUID NOT NULL REFERENCES courses(id),
    verification_id VARCHAR(64) UNIQUE NOT NULL,
    student_name    VARCHAR(100) NOT NULL,
    course_title    VARCHAR(255) NOT NULL,
    instructor_name VARCHAR(100) NOT NULL,
    completion_date DATE NOT NULL,
    pdf_rustfs_key  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_certificates_student_course UNIQUE (student_id, course_id)
);

-- Index for fast lookup by verification_id (public verification endpoint)
CREATE INDEX IF NOT EXISTS idx_certificates_verification_id ON certificates (verification_id);

-- Index for student certificate listing
CREATE INDEX IF NOT EXISTS idx_certificates_student_id ON certificates (student_id);
