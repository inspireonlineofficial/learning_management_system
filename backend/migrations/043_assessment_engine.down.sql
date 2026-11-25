-- Reverts 043. Data in the new columns is discarded.
--
-- Questions of the new types (numeric, long_answer) cannot satisfy the old
-- type check. The restored constraint is added NOT VALID so the rollback
-- never fails on existing rows; new inserts are still checked. Convert or
-- delete those questions before relying on the old application version,
-- which does not know how to grade them.

DROP INDEX IF EXISTS idx_quiz_attempts_pending_grading;
DROP INDEX IF EXISTS idx_quiz_attempts_quiz_started;

ALTER TABLE quiz_attempts DROP CONSTRAINT IF EXISTS quiz_attempts_grading_status_check;
ALTER TABLE quiz_attempts
    DROP COLUMN IF EXISTS shuffle_seed,
    DROP COLUMN IF EXISTS graded_by,
    DROP COLUMN IF EXISTS graded_at,
    DROP COLUMN IF EXISTS grading_status,
    DROP COLUMN IF EXISTS max_points,
    DROP COLUMN IF EXISTS score_points,
    DROP COLUMN IF EXISTS results,
    DROP COLUMN IF EXISTS answers,
    DROP COLUMN IF EXISTS draft_revision;

ALTER TABLE questions DROP CONSTRAINT IF EXISTS questions_type_check;
ALTER TABLE questions ADD CONSTRAINT questions_type_check
    CHECK (type IN ('single', 'multiple', 'true_false', 'short_answer')) NOT VALID;
ALTER TABLE questions
    DROP COLUMN IF EXISTS settings,
    DROP COLUMN IF EXISTS hint;

ALTER TABLE quizzes DROP CONSTRAINT IF EXISTS quizzes_availability_check;
ALTER TABLE quizzes DROP CONSTRAINT IF EXISTS quizzes_calculator_policy_check;
ALTER TABLE quizzes DROP CONSTRAINT IF EXISTS quizzes_mode_check;
ALTER TABLE quizzes
    DROP COLUMN IF EXISTS reference_material,
    DROP COLUMN IF EXISTS calculator_policy,
    DROP COLUMN IF EXISTS available_until,
    DROP COLUMN IF EXISTS available_from,
    DROP COLUMN IF EXISTS partial_credit,
    DROP COLUMN IF EXISTS shuffle_options,
    DROP COLUMN IF EXISTS mode,
    DROP COLUMN IF EXISTS instructions;
