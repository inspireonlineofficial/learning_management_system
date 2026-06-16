-- Create RBAC permissions table
CREATE TABLE rbac_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role VARCHAR(20) NOT NULL CHECK (role IN ('student', 'teacher', 'admin')),
    resource VARCHAR(100) NOT NULL,
    action VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(role, resource, action)
);

-- Insert default permissions for student role
INSERT INTO rbac_permissions (role, resource, action) VALUES
    ('student', 'courses', 'read'),
    ('student', 'enrollments', 'create'),
    ('student', 'enrollments', 'read'),
    ('student', 'forum_posts', 'create'),
    ('student', 'forum_posts', 'read'),
    ('student', 'quiz_attempts', 'create'),
    ('student', 'assignment_submissions', 'create');

-- Insert default permissions for teacher role (includes all student permissions)
INSERT INTO rbac_permissions (role, resource, action) VALUES
    ('teacher', 'courses', 'read'),
    ('teacher', 'enrollments', 'create'),
    ('teacher', 'enrollments', 'read'),
    ('teacher', 'forum_posts', 'create'),
    ('teacher', 'forum_posts', 'read'),
    ('teacher', 'quiz_attempts', 'create'),
    ('teacher', 'assignment_submissions', 'create'),
    ('teacher', 'courses', 'create'),
    ('teacher', 'courses', 'update_own'),
    ('teacher', 'quizzes', 'create'),
    ('teacher', 'quizzes', 'update_own'),
    ('teacher', 'quizzes', 'read_answers'),
    ('teacher', 'assignments', 'create'),
    ('teacher', 'submissions', 'grade'),
    ('teacher', 'live_sessions', 'create');

-- Admin has all permissions (enforced at application layer)
