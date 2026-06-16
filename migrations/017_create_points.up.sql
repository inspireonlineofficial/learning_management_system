-- Create point_events table (append-only)
CREATE TABLE IF NOT EXISTS point_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('video_complete', 'quiz_pass', 'quiz_perfect')),
    source_id UUID NOT NULL,
    source_title VARCHAR(255),
    points INT NOT NULL,
    bonus_points INT NOT NULL DEFAULT 0,
    earned_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index for daily dedup and history queries
CREATE INDEX idx_point_events_student_earned ON point_events(student_id, earned_at);

-- Create points_config table (singleton, id always = 1)
CREATE TABLE IF NOT EXISTS points_config (
    id INT PRIMARY KEY DEFAULT 1,
    points_per_video INT NOT NULL DEFAULT 10,
    points_per_quiz_pass INT NOT NULL DEFAULT 20,
    bonus_points_perfect_score INT NOT NULL DEFAULT 10,
    updated_at TIMESTAMPTZ,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT singleton_check CHECK (id = 1)
);

-- Insert default singleton row
INSERT INTO points_config (id, points_per_video, points_per_quiz_pass, bonus_points_perfect_score)
VALUES (1, 10, 20, 10)
ON CONFLICT (id) DO NOTHING;
