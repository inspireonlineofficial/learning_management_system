package postgres

import (
	"context"
	"database/sql"

	"lms-backend/internal/domain/assessments"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

type quizAttemptRepository struct {
	db *sql.DB
}

// NewQuizAttemptRepository creates a new quiz attempt repository
func NewQuizAttemptRepository(db *sql.DB) assessments.QuizAttemptRepository {
	return &quizAttemptRepository{db: db}
}

func (r *quizAttemptRepository) Create(ctx context.Context, attempt *assessments.QuizAttempt) error {
	query := `
		INSERT INTO quiz_attempts (
			id, quiz_id, student_id, started_at, submitted_at,
			score_percent, passed, time_taken_seconds, points_awarded, status, draft_answers
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, COALESCE($11::jsonb, '{}'::jsonb))
	`

	_, err := r.db.ExecContext(ctx, query,
		attempt.ID,
		attempt.QuizID,
		attempt.StudentID,
		attempt.StartedAt,
		attempt.SubmittedAt,
		attempt.ScorePercent,
		attempt.Passed,
		attempt.TimeTakenSeconds,
		attempt.PointsAwarded,
		attempt.Status,
		nullJSON(attempt.DraftAnswers),
	)

	if err != nil {
		return apperrors.NewInternalError("DB_ERROR", "failed to create quiz attempt")
	}

	return nil
}

func (r *quizAttemptRepository) FindByID(ctx context.Context, id uuid.UUID) (*assessments.QuizAttempt, error) {
	query := `
		SELECT id, quiz_id, student_id, started_at, submitted_at,
			   score_percent, passed, time_taken_seconds, points_awarded, status, COALESCE(draft_answers, '{}'::jsonb)
		FROM quiz_attempts
		WHERE id = $1
	`

	attempt := &assessments.QuizAttempt{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&attempt.ID,
		&attempt.QuizID,
		&attempt.StudentID,
		&attempt.StartedAt,
		&attempt.SubmittedAt,
		&attempt.ScorePercent,
		&attempt.Passed,
		&attempt.TimeTakenSeconds,
		&attempt.PointsAwarded,
		&attempt.Status,
		&attempt.DraftAnswers,
	)

	if err == sql.ErrNoRows {
		return nil, apperrors.NewNotFoundError("ATTEMPT_NOT_FOUND", "quiz attempt not found")
	}
	if err != nil {
		return nil, apperrors.NewInternalError("DB_ERROR", "failed to find quiz attempt")
	}

	return attempt, nil
}

func (r *quizAttemptRepository) FindByQuizAndStudent(ctx context.Context, quizID, studentID uuid.UUID) ([]*assessments.QuizAttempt, error) {
	query := `
		SELECT id, quiz_id, student_id, started_at, submitted_at,
			   score_percent, passed, time_taken_seconds, points_awarded, status, COALESCE(draft_answers, '{}'::jsonb)
		FROM quiz_attempts
		WHERE quiz_id = $1 AND student_id = $2
		ORDER BY started_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, quizID, studentID)
	if err != nil {
		return nil, apperrors.NewInternalError("DB_ERROR", "failed to find quiz attempts")
	}
	defer rows.Close()

	attempts := make([]*assessments.QuizAttempt, 0)
	for rows.Next() {
		attempt := &assessments.QuizAttempt{}
		err := rows.Scan(
			&attempt.ID,
			&attempt.QuizID,
			&attempt.StudentID,
			&attempt.StartedAt,
			&attempt.SubmittedAt,
			&attempt.ScorePercent,
			&attempt.Passed,
			&attempt.TimeTakenSeconds,
			&attempt.PointsAwarded,
			&attempt.Status,
			&attempt.DraftAnswers,
		)
		if err != nil {
			return nil, apperrors.NewInternalError("DB_ERROR", "failed to scan quiz attempt")
		}
		attempts = append(attempts, attempt)
	}

	return attempts, nil
}

func (r *quizAttemptRepository) CountAttempts(ctx context.Context, quizID, studentID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM quiz_attempts
		WHERE quiz_id = $1 AND student_id = $2
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, quizID, studentID).Scan(&count)
	if err != nil {
		return 0, apperrors.NewInternalError("DB_ERROR", "failed to count quiz attempts")
	}

	return count, nil
}

func (r *quizAttemptRepository) GetHighestScore(ctx context.Context, quizID, studentID uuid.UUID) (*float64, error) {
	query := `
		SELECT MAX(score_percent)
		FROM quiz_attempts
		WHERE quiz_id = $1 AND student_id = $2 AND score_percent IS NOT NULL
	`

	var score *float64
	err := r.db.QueryRowContext(ctx, query, quizID, studentID).Scan(&score)
	if err != nil && err != sql.ErrNoRows {
		return nil, apperrors.NewInternalError("DB_ERROR", "failed to get highest score")
	}

	return score, nil
}

func (r *quizAttemptRepository) Update(ctx context.Context, attempt *assessments.QuizAttempt) error {
	query := `
		UPDATE quiz_attempts
		SET submitted_at = $1, score_percent = $2, passed = $3,
			time_taken_seconds = $4, points_awarded = $5, status = $6,
			draft_answers = COALESCE($7::jsonb, draft_answers, '{}'::jsonb)
		WHERE id = $8
	`

	result, err := r.db.ExecContext(ctx, query,
		attempt.SubmittedAt,
		attempt.ScorePercent,
		attempt.Passed,
		attempt.TimeTakenSeconds,
		attempt.PointsAwarded,
		attempt.Status,
		nullJSON(attempt.DraftAnswers),
		attempt.ID,
	)

	if err != nil {
		return apperrors.NewInternalError("DB_ERROR", "failed to update quiz attempt")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewInternalError("DB_ERROR", "failed to check rows affected")
	}

	if rows == 0 {
		return apperrors.NewNotFoundError("ATTEMPT_NOT_FOUND", "quiz attempt not found")
	}

	return nil
}

func (r *quizAttemptRepository) SaveDraftAnswers(ctx context.Context, attemptID uuid.UUID, draftAnswers []byte) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE quiz_attempts
		SET draft_answers = COALESCE($1::jsonb, '{}'::jsonb)
		WHERE id = $2`,
		nullJSON(draftAnswers), attemptID,
	)
	if err != nil {
		return apperrors.NewInternalError("DB_ERROR", "failed to save quiz draft answers")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewInternalError("DB_ERROR", "failed to check rows affected")
	}
	if rows == 0 {
		return apperrors.NewNotFoundError("ATTEMPT_NOT_FOUND", "quiz attempt not found")
	}
	return nil
}

func (r *quizAttemptRepository) FindInProgressAttempts(ctx context.Context) ([]*assessments.QuizAttempt, error) {
	query := `
		SELECT id, quiz_id, student_id, started_at, submitted_at,
			   score_percent, passed, time_taken_seconds, points_awarded, status, COALESCE(draft_answers, '{}'::jsonb)
		FROM quiz_attempts
		WHERE status = 'in_progress'
		ORDER BY started_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, apperrors.NewInternalError("DB_ERROR", "failed to find in-progress attempts")
	}
	defer rows.Close()

	attempts := make([]*assessments.QuizAttempt, 0)
	for rows.Next() {
		attempt := &assessments.QuizAttempt{}
		err := rows.Scan(
			&attempt.ID,
			&attempt.QuizID,
			&attempt.StudentID,
			&attempt.StartedAt,
			&attempt.SubmittedAt,
			&attempt.ScorePercent,
			&attempt.Passed,
			&attempt.TimeTakenSeconds,
			&attempt.PointsAwarded,
			&attempt.Status,
			&attempt.DraftAnswers,
		)
		if err != nil {
			return nil, apperrors.NewInternalError("DB_ERROR", "failed to scan quiz attempt")
		}
		attempts = append(attempts, attempt)
	}

	return attempts, nil
}

func nullJSON(raw []byte) interface{} {
	if len(raw) == 0 {
		return nil
	}
	return raw
}
