-- Create quizzes table
CREATE TABLE IF NOT EXISTS quizzes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    lesson_id UUID REFERENCES lessons(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    time_limit_seconds INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 0,
    passing_score_percent NUMERIC(5, 2) NOT NULL DEFAULT 60.00,
    shuffle_questions BOOLEAN NOT NULL DEFAULT false,
    show_answers_after_submission BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for quizzes
CREATE INDEX idx_quizzes_course_id ON quizzes(course_id);
CREATE INDEX idx_quizzes_lesson_id ON quizzes(lesson_id);

-- Create questions table
CREATE TABLE IF NOT EXISTS questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_id UUID NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('single', 'multiple', 'true_false')),
    position INT NOT NULL,
    explanation TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for questions
CREATE INDEX idx_questions_quiz_id ON questions(quiz_id);
CREATE INDEX idx_questions_position ON questions(quiz_id, position);

-- Create question_options table
CREATE TABLE IF NOT EXISTS question_options (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    is_correct BOOLEAN NOT NULL DEFAULT false,
    position INT NOT NULL
);

-- Create indexes for question_options
CREATE INDEX idx_question_options_question_id ON question_options(question_id);
CREATE INDEX idx_question_options_position ON question_options(question_id, position);

-- Create quiz_attempts table
CREATE TABLE IF NOT EXISTS quiz_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_id UUID NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    submitted_at TIMESTAMPTZ,
    score_percent NUMERIC(5, 2),
    passed BOOLEAN,
    time_taken_seconds INT,
    points_awarded INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'submitted', 'auto_submitted'))
);

-- Create indexes for quiz_attempts
CREATE INDEX idx_quiz_attempts_quiz_id ON quiz_attempts(quiz_id);
CREATE INDEX idx_quiz_attempts_student_id ON quiz_attempts(student_id);
CREATE INDEX idx_quiz_attempts_status ON quiz_attempts(status);
CREATE INDEX idx_quiz_attempts_student_quiz ON quiz_attempts(student_id, quiz_id);

-- Create assignments table
CREATE TABLE IF NOT EXISTS assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    due_at TIMESTAMPTZ NOT NULL,
    submission_type VARCHAR(10) NOT NULL CHECK (submission_type IN ('file', 'text', 'both')),
    max_file_size_mb INT NOT NULL DEFAULT 50,
    allow_late_submission BOOLEAN NOT NULL DEFAULT false,
    total_marks NUMERIC(10, 2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for assignments
CREATE INDEX idx_assignments_course_id ON assignments(course_id);
CREATE INDEX idx_assignments_due_at ON assignments(due_at);

-- Create assignment_submissions table
CREATE TABLE IF NOT EXISTS assignment_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id UUID NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(30) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'graded', 'revision_requested')),
    text_content TEXT,
    submitted_at TIMESTAMPTZ,
    is_late BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_student_assignment UNIQUE (student_id, assignment_id)
);

-- Create indexes for assignment_submissions
CREATE INDEX idx_assignment_submissions_assignment_id ON assignment_submissions(assignment_id);
CREATE INDEX idx_assignment_submissions_student_id ON assignment_submissions(student_id);
CREATE INDEX idx_assignment_submissions_status ON assignment_submissions(status);

-- Create submission_files table
CREATE TABLE IF NOT EXISTS submission_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id UUID NOT NULL REFERENCES assignment_submissions(id) ON DELETE CASCADE,
    rustfs_key VARCHAR(500) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    size_bytes BIGINT NOT NULL
);

-- Create indexes for submission_files
CREATE INDEX idx_submission_files_submission_id ON submission_files(submission_id);

-- Create submission_grades table (append-only)
CREATE TABLE IF NOT EXISTS submission_grades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id UUID NOT NULL REFERENCES assignment_submissions(id) ON DELETE CASCADE,
    graded_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    score NUMERIC(10, 2) NOT NULL,
    feedback TEXT,
    revision_requested BOOLEAN NOT NULL DEFAULT false,
    revision_notes TEXT,
    graded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for submission_grades
CREATE INDEX idx_submission_grades_submission_id ON submission_grades(submission_id);
CREATE INDEX idx_submission_grades_graded_by ON submission_grades(graded_by);
CREATE INDEX idx_submission_grades_graded_at ON submission_grades(graded_at);
