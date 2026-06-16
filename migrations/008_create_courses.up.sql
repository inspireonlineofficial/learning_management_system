-- Create courses table
CREATE TABLE IF NOT EXISTS courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    short_description VARCHAR(500),
    description TEXT,
    subject VARCHAR(100),
    level VARCHAR(20) CHECK (level IN ('beginner', 'intermediate', 'advanced')),
    price_type VARCHAR(10) NOT NULL CHECK (price_type IN ('free', 'paid')),
    price NUMERIC(10, 2) DEFAULT 0,
    currency CHAR(3) DEFAULT 'BDT',
    prerequisites TEXT,
    thumbnail_url TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'pending', 'published', 'rejected')),
    rating_average NUMERIC(3, 2) DEFAULT 0,
    rating_count INT DEFAULT 0,
    total_enrolled INT DEFAULT 0,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Create indexes for courses
CREATE INDEX idx_courses_status_deleted ON courses(status, deleted_at);
CREATE INDEX idx_courses_teacher_id ON courses(teacher_id);
CREATE INDEX idx_courses_slug ON courses(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_courses_published_at ON courses(published_at) WHERE status = 'published' AND deleted_at IS NULL;
