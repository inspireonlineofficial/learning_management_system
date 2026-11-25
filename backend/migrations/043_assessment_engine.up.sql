-- 043: assessment engine upgrade (additive and reversible).
--
-- * Quiz settings: instructions, description is already present (036),
--   practice/exam mode, option shuffling, partial credit, availability window,
--   calculator policy and permitted reference material.
-- * Questions: hints and per-type settings (numeric answer + tolerance, word
--   limits) plus two new types: numeric (auto-graded) and long_answer
--   (manually graded).
-- * Attempts: ordered autosave (draft_revision), the final submitted answers,
--   a per-question result snapshot (question text, options, answer key, marks
--   awarded) so later edits to a quiz never change historical attempts, mark
--   totals, manual-grading status, and a per-attempt shuffle seed so the
--   question order is stable across refreshes.
--
-- Existing rows: quizzes default to exam mode with no window; existing
-- attempts keep their score_percent and are marked grading_status = 'auto'.

ALTER TABLE quizzes
    ADD COLUMN IF NOT EXISTS instructions TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS mode VARCHAR(20) NOT NULL DEFAULT 'exam',
    ADD COLUMN IF NOT EXISTS shuffle_options BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS partial_credit BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS available_from TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS available_until TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS calculator_policy VARCHAR(20) NOT NULL DEFAULT 'none',
    ADD COLUMN IF NOT EXISTS reference_material TEXT NOT NULL DEFAULT '';

ALTER TABLE quizzes DROP CONSTRAINT IF EXISTS quizzes_mode_check;
ALTER TABLE quizzes ADD CONSTRAINT quizzes_mode_check CHECK (mode IN ('practice', 'exam'));
ALTER TABLE quizzes DROP CONSTRAINT IF EXISTS quizzes_calculator_policy_check;
ALTER TABLE quizzes ADD CONSTRAINT quizzes_calculator_policy_check
    CHECK (calculator_policy IN ('none', 'basic', 'scientific', 'graphing'));
ALTER TABLE quizzes DROP CONSTRAINT IF EXISTS quizzes_availability_check;
ALTER TABLE quizzes ADD CONSTRAINT quizzes_availability_check
    CHECK (available_from IS NULL OR available_until IS NULL OR available_until > available_from);

ALTER TABLE questions
    ADD COLUMN IF NOT EXISTS hint TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS settings JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE questions DROP CONSTRAINT IF EXISTS questions_type_check;
ALTER TABLE questions ADD CONSTRAINT questions_type_check
    CHECK (type IN ('single', 'multiple', 'true_false', 'short_answer', 'numeric', 'long_answer'));

ALTER TABLE quiz_attempts
    ADD COLUMN IF NOT EXISTS draft_revision BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS answers JSONB,
    ADD COLUMN IF NOT EXISTS results JSONB,
    ADD COLUMN IF NOT EXISTS score_points NUMERIC(10, 2),
    ADD COLUMN IF NOT EXISTS max_points NUMERIC(10, 2),
    ADD COLUMN IF NOT EXISTS grading_status VARCHAR(20) NOT NULL DEFAULT 'auto',
    ADD COLUMN IF NOT EXISTS graded_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS graded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS shuffle_seed BIGINT NOT NULL DEFAULT 0;

ALTER TABLE quiz_attempts DROP CONSTRAINT IF EXISTS quiz_attempts_grading_status_check;
ALTER TABLE quiz_attempts ADD CONSTRAINT quiz_attempts_grading_status_check
    CHECK (grading_status IN ('auto', 'pending', 'graded'));

-- Teacher monitoring lists attempts per quiz, newest first.
CREATE INDEX IF NOT EXISTS idx_quiz_attempts_quiz_started ON quiz_attempts(quiz_id, started_at DESC);
-- Grading queue.
CREATE INDEX IF NOT EXISTS idx_quiz_attempts_pending_grading
    ON quiz_attempts(quiz_id) WHERE grading_status = 'pending';
