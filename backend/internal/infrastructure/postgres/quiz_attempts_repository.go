package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"lms-backend/internal/domain/assessments"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

type quizAttemptRepository struct {
	db *sql.DB
}

// NewQuizAttemptRepository creates a new quiz attempt repository. All methods
// join an ambient transaction (see TxRunner) when one is present.
func NewQuizAttemptRepository(db *sql.DB) assessments.QuizAttemptRepository {
	return &quizAttemptRepository{db: db}
}

const attemptColumns = `qa.id, qa.quiz_id, qa.student_id, qa.started_at, qa.submitted_at,
	qa.score_percent, qa.passed, qa.time_taken_seconds, qa.points_awarded, qa.status,
	COALESCE(qa.draft_answers, '{}'::jsonb), qa.draft_revision, COALESCE(qa.answers, 'null'::jsonb), COALESCE(qa.results, 'null'::jsonb),
	qa.score_points, qa.max_points, qa.grading_status, qa.graded_at, qa.graded_by, qa.shuffle_seed`

func attemptScanTargets(a *assessments.QuizAttempt) []any {
	return []any{
		&a.ID, &a.QuizID, &a.StudentID, &a.StartedAt, &a.SubmittedAt,
		&a.ScorePercent, &a.Passed, &a.TimeTakenSeconds, &a.PointsAwarded, &a.Status,
		&a.DraftAnswers, &a.DraftRevision, &a.Answers, &a.Results,
		&a.ScorePoints, &a.MaxPoints, &a.GradingStatus, &a.GradedAt, &a.GradedBy, &a.ShuffleSeed,
	}
}

func gradingStatusOrDefault(status assessments.GradingStatus) string {
	switch status {
	case assessments.GradingStatusPending, assessments.GradingStatusGraded:
		return string(status)
	default:
		return string(assessments.GradingStatusAuto)
	}
}

func (r *quizAttemptRepository) Create(ctx context.Context, attempt *assessments.QuizAttempt) error {
	query := `
		INSERT INTO quiz_attempts (
			id, quiz_id, student_id, started_at, submitted_at,
			score_percent, passed, time_taken_seconds, points_awarded, status, draft_answers,
			grading_status, shuffle_seed
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, COALESCE($11::jsonb, '{}'::jsonb), $12, $13)
	`
	_, err := executorForContext(ctx, r.db).ExecContext(ctx, query,
		attempt.ID, attempt.QuizID, attempt.StudentID, attempt.StartedAt, attempt.SubmittedAt,
		attempt.ScorePercent, attempt.Passed, attempt.TimeTakenSeconds, attempt.PointsAwarded, attempt.Status,
		nullJSON(attempt.DraftAnswers), gradingStatusOrDefault(attempt.GradingStatus), attempt.ShuffleSeed,
	)
	if err != nil {
		return apperrors.NewInternalError("DB_ERROR", "failed to create quiz attempt")
	}
	return nil
}

func (r *quizAttemptRepository) FindByID(ctx context.Context, id uuid.UUID) (*assessments.QuizAttempt, error) {
	attempt := &assessments.QuizAttempt{}
	err := executorForContext(ctx, r.db).QueryRowContext(ctx,
		`SELECT `+attemptColumns+` FROM quiz_attempts qa WHERE qa.id = $1`, id,
	).Scan(attemptScanTargets(attempt)...)
	if err == sql.ErrNoRows {
		return nil, apperrors.NewNotFoundError("ATTEMPT_NOT_FOUND", "quiz attempt not found")
	}
	if err != nil {
		return nil, apperrors.NewInternalError("DB_ERROR", "failed to find quiz attempt")
	}
	return attempt, nil
}

func (r *quizAttemptRepository) queryAttempts(ctx context.Context, query string, args ...any) ([]*assessments.QuizAttempt, error) {
	rows, err := executorForContext(ctx, r.db).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperrors.NewInternalError("DB_ERROR", "failed to find quiz attempts")
	}
	defer rows.Close()

	attempts := make([]*assessments.QuizAttempt, 0)
	for rows.Next() {
		attempt := &assessments.QuizAttempt{}
		if err := rows.Scan(attemptScanTargets(attempt)...); err != nil {
			return nil, apperrors.NewInternalError("DB_ERROR", "failed to scan quiz attempt")
		}
		attempts = append(attempts, attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.NewInternalError("DB_ERROR", "failed to read quiz attempts")
	}
	return attempts, nil
}

func (r *quizAttemptRepository) FindByQuizAndStudent(ctx context.Context, quizID, studentID uuid.UUID) ([]*assessments.QuizAttempt, error) {
	return r.queryAttempts(ctx,
		`SELECT `+attemptColumns+` FROM quiz_attempts qa
		 WHERE qa.quiz_id = $1 AND qa.student_id = $2
		 ORDER BY qa.started_at DESC`, quizID, studentID)
}

func (r *quizAttemptRepository) CountAttempts(ctx context.Context, quizID, studentID uuid.UUID) (int, error) {
	var count int
	err := executorForContext(ctx, r.db).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM quiz_attempts WHERE quiz_id = $1 AND student_id = $2`, quizID, studentID,
	).Scan(&count)
	if err != nil {
		return 0, apperrors.NewInternalError("DB_ERROR", "failed to count quiz attempts")
	}
	return count, nil
}

func (r *quizAttemptRepository) GetHighestScore(ctx context.Context, quizID, studentID uuid.UUID) (*float64, error) {
	var score *float64
	err := executorForContext(ctx, r.db).QueryRowContext(ctx, `
		SELECT MAX(score_percent)
		FROM quiz_attempts
		WHERE quiz_id = $1 AND student_id = $2 AND score_percent IS NOT NULL`, quizID, studentID,
	).Scan(&score)
	if err != nil && err != sql.ErrNoRows {
		return nil, apperrors.NewInternalError("DB_ERROR", "failed to get highest score")
	}
	return score, nil
}

func (r *quizAttemptRepository) Update(ctx context.Context, attempt *assessments.QuizAttempt) error {
	result, err := executorForContext(ctx, r.db).ExecContext(ctx, `
		UPDATE quiz_attempts
		SET submitted_at = $1, score_percent = $2, passed = $3,
			time_taken_seconds = $4, points_awarded = $5, status = $6,
			draft_answers = COALESCE($7::jsonb, draft_answers, '{}'::jsonb)
		WHERE id = $8`,
		attempt.SubmittedAt, attempt.ScorePercent, attempt.Passed, attempt.TimeTakenSeconds,
		attempt.PointsAwarded, attempt.Status, nullJSON(attempt.DraftAnswers), attempt.ID,
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
	result, err := executorForContext(ctx, r.db).ExecContext(ctx, `
		UPDATE quiz_attempts
		SET draft_answers = COALESCE($1::jsonb, '{}'::jsonb)
		WHERE id = $2 AND status = 'in_progress'`,
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

func (r *quizAttemptRepository) SaveDraftRevision(ctx context.Context, attemptID uuid.UUID, draftAnswers []byte, revision int64) (bool, error) {
	result, err := executorForContext(ctx, r.db).ExecContext(ctx, `
		UPDATE quiz_attempts
		SET draft_answers = COALESCE($1::jsonb, '{}'::jsonb), draft_revision = $2
		WHERE id = $3 AND status = 'in_progress' AND draft_revision < $2`,
		nullJSON(draftAnswers), revision, attemptID,
	)
	if err != nil {
		return false, apperrors.NewInternalError("DB_ERROR", "failed to save quiz draft answers")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, apperrors.NewInternalError("DB_ERROR", "failed to check rows affected")
	}
	return rows == 1, nil
}

func (r *quizAttemptRepository) Finalize(ctx context.Context, attempt *assessments.QuizAttempt) (bool, error) {
	result, err := executorForContext(ctx, r.db).ExecContext(ctx, `
		UPDATE quiz_attempts
		SET submitted_at = $1, score_percent = $2, passed = $3, time_taken_seconds = $4,
			points_awarded = $5, status = $6, answers = $7::jsonb, results = $8::jsonb,
			score_points = $9, max_points = $10, grading_status = $11
		WHERE id = $12 AND status = 'in_progress'`,
		attempt.SubmittedAt, attempt.ScorePercent, attempt.Passed, attempt.TimeTakenSeconds,
		attempt.PointsAwarded, attempt.Status, nullJSON(attempt.Answers), nullJSON(attempt.Results),
		attempt.ScorePoints, attempt.MaxPoints, gradingStatusOrDefault(attempt.GradingStatus), attempt.ID,
	)
	if err != nil {
		return false, apperrors.NewInternalError("DB_ERROR", "failed to submit quiz attempt")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, apperrors.NewInternalError("DB_ERROR", "failed to check rows affected")
	}
	return rows == 1, nil
}

func (r *quizAttemptRepository) SaveGrading(ctx context.Context, attempt *assessments.QuizAttempt) error {
	result, err := executorForContext(ctx, r.db).ExecContext(ctx, `
		UPDATE quiz_attempts
		SET results = $1::jsonb, score_points = $2, max_points = $3, score_percent = $4,
			passed = $5, grading_status = $6, graded_at = $7, graded_by = $8, points_awarded = $9
		WHERE id = $10 AND status <> 'in_progress'`,
		nullJSON(attempt.Results), attempt.ScorePoints, attempt.MaxPoints, attempt.ScorePercent,
		attempt.Passed, gradingStatusOrDefault(attempt.GradingStatus), attempt.GradedAt, attempt.GradedBy,
		attempt.PointsAwarded, attempt.ID,
	)
	if err != nil {
		return apperrors.NewInternalError("DB_ERROR", "failed to save grading")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return apperrors.NewInternalError("DB_ERROR", "failed to check rows affected")
	}
	if rows == 0 {
		return apperrors.NewNotFoundError("ATTEMPT_NOT_FOUND", "submitted attempt not found")
	}
	return nil
}

func (r *quizAttemptRepository) FindInProgressAttempts(ctx context.Context) ([]*assessments.QuizAttempt, error) {
	return r.queryAttempts(ctx,
		`SELECT `+attemptColumns+` FROM quiz_attempts qa
		 WHERE qa.status = 'in_progress'
		 ORDER BY qa.started_at ASC`)
}

func (r *quizAttemptRepository) ListByQuiz(ctx context.Context, quizID uuid.UUID) ([]*assessments.QuizAttemptWithStudent, error) {
	rows, err := executorForContext(ctx, r.db).QueryContext(ctx, `
		SELECT `+attemptColumns+`, COALESCE(u.full_name, ''), COALESCE(u.email, ''),
			ROW_NUMBER() OVER (PARTITION BY qa.student_id ORDER BY qa.started_at ASC)
		FROM quiz_attempts qa
		LEFT JOIN users u ON u.id = qa.student_id
		WHERE qa.quiz_id = $1
		ORDER BY qa.started_at DESC`, quizID)
	if err != nil {
		return nil, apperrors.NewInternalError("DB_ERROR", "failed to list quiz attempts")
	}
	defer rows.Close()

	result := make([]*assessments.QuizAttemptWithStudent, 0)
	for rows.Next() {
		item := &assessments.QuizAttemptWithStudent{}
		targets := append(attemptScanTargets(&item.QuizAttempt), &item.StudentName, &item.StudentEmail, &item.AttemptNumber)
		if err := rows.Scan(targets...); err != nil {
			return nil, apperrors.NewInternalError("DB_ERROR", "failed to scan quiz attempt")
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.NewInternalError("DB_ERROR", "failed to read quiz attempts")
	}
	return result, nil
}

func (r *quizAttemptRepository) CountByQuizIDs(ctx context.Context, quizIDs []uuid.UUID) (map[uuid.UUID]assessments.QuizAttemptCounts, error) {
	counts := make(map[uuid.UUID]assessments.QuizAttemptCounts, len(quizIDs))
	if len(quizIDs) == 0 {
		return counts, nil
	}
	placeholders := make([]string, len(quizIDs))
	args := make([]any, len(quizIDs))
	for i, id := range quizIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	rows, err := executorForContext(ctx, r.db).QueryContext(ctx, `
		SELECT quiz_id,
			COUNT(*),
			COUNT(*) FILTER (WHERE status <> 'in_progress'),
			COUNT(*) FILTER (WHERE status = 'in_progress'),
			COUNT(*) FILTER (WHERE status <> 'in_progress' AND grading_status = 'pending')
		FROM quiz_attempts
		WHERE quiz_id IN (`+strings.Join(placeholders, ",")+`)
		GROUP BY quiz_id`, args...)
	if err != nil {
		return nil, apperrors.NewInternalError("DB_ERROR", "failed to count quiz attempts")
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var c assessments.QuizAttemptCounts
		if err := rows.Scan(&id, &c.Attempts, &c.Submitted, &c.InProgress, &c.PendingGrading); err != nil {
			return nil, apperrors.NewInternalError("DB_ERROR", "failed to scan attempt counts")
		}
		counts[id] = c
	}
	return counts, rows.Err()
}

func nullJSON(raw []byte) interface{} {
	if len(raw) == 0 {
		return nil
	}
	return raw
}

// LockStudentQuiz takes a transaction-scoped advisory lock for one student
// and quiz. It only serialises callers inside a transaction (TxRunner).
func (r *quizAttemptRepository) LockStudentQuiz(ctx context.Context, quizID, studentID uuid.UUID) error {
	if TxFromContext(ctx) == nil {
		return nil
	}
	_, err := executorForContext(ctx, r.db).ExecContext(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "quiz_attempt:"+quizID.String()+":"+studentID.String())
	if err != nil {
		return apperrors.NewInternalError("DB_ERROR", "failed to lock quiz attempt")
	}
	return nil
}
