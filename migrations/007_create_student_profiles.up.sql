-- Create student_profiles table
CREATE TABLE IF NOT EXISTS student_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    school_name VARCHAR(200) NOT NULL,
    class_grade VARCHAR(50) NOT NULL,
    roll_number VARCHAR(30) NOT NULL,
    date_of_birth DATE NOT NULL,
    gender VARCHAR(30),
    guardian_name VARCHAR(100),
    guardian_contact VARCHAR(30),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add index for faster lookups
CREATE INDEX idx_student_profiles_user_id ON student_profiles(user_id);

-- Add check constraint for date_of_birth (must be in the past)
ALTER TABLE student_profiles ADD CONSTRAINT chk_date_of_birth_past CHECK (date_of_birth < CURRENT_DATE);

-- Add check constraint for age (must be <= 30 years)
ALTER TABLE student_profiles ADD CONSTRAINT chk_age_limit CHECK (
    date_of_birth >= CURRENT_DATE - INTERVAL '30 years'
);
