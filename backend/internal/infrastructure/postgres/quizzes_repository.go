package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"lms-backend/internal/domain/assessments"

	"github.com/google/uuid"
)

type quizRepository struct {
	db *sql.DB
}

// NewQuizRepository creates a new quiz repository. All methods join an
// ambient transaction (see TxRunner) when one is present in the context.
func NewQuizRepository(db *sql.DB) assessments.QuizRepository {
	return &quizRepository{db: db}
}

const quizColumns = `
	id, course_id, lesson_id, title, time_limit_seconds, max_attempts,
	passing_score_percent, shuffle_questions, show_answers_after_submission,
	is_free, is_published, created_at, updated_at,
	description, instructions, mode, shuffle_options, partial_credit,
	available_from, available_until, calculator_policy, reference_material`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanQuiz(row rowScanner) (*assessments.Quiz, error) {
	quiz := &assessments.Quiz{}
	var mode string
	err := row.Scan(
		&quiz.ID, &quiz.CourseID, &quiz.LessonID, &quiz.Title, &quiz.TimeLimitSeconds,
		&quiz.MaxAttempts, &quiz.PassingScorePercent, &quiz.ShuffleQuestions,
		&quiz.ShowAnswersAfterSubmission, &quiz.IsFree, &quiz.IsPublished, &quiz.CreatedAt, &quiz.UpdatedAt,
		&quiz.Description, &quiz.Instructions, &mode, &quiz.ShuffleOptions, &quiz.PartialCredit,
		&quiz.AvailableFrom, &quiz.AvailableUntil, &quiz.CalculatorPolicy, &quiz.ReferenceMaterial,
	)
	quiz.Mode = assessments.QuizMode(mode)
	return quiz, err
}

func quizModeOrDefault(mode assessments.QuizMode) string {
	if mode == assessments.QuizModePractice {
		return string(assessments.QuizModePractice)
	}
	return string(assessments.QuizModeExam)
}

func calculatorPolicyOrDefault(policy string) string {
	switch policy {
	case "basic", "scientific", "graphing":
		return policy
	default:
		return "none"
	}
}

func (r *quizRepository) Create(ctx context.Context, quiz *assessments.Quiz) error {
	query := `
		INSERT INTO quizzes (` + quizColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
		        $14, $15, $16, $17, $18, $19, $20, $21, $22)
	`
	_, err := executorForContext(ctx, r.db).ExecContext(ctx, query,
		quiz.ID, quiz.CourseID, quiz.LessonID, quiz.Title, quiz.TimeLimitSeconds,
		quiz.MaxAttempts, quiz.PassingScorePercent, quiz.ShuffleQuestions,
		quiz.ShowAnswersAfterSubmission, quiz.IsFree, quiz.IsPublished, quiz.CreatedAt, quiz.UpdatedAt,
		quiz.Description, quiz.Instructions, quizModeOrDefault(quiz.Mode), quiz.ShuffleOptions, quiz.PartialCredit,
		quiz.AvailableFrom, quiz.AvailableUntil, calculatorPolicyOrDefault(quiz.CalculatorPolicy), quiz.ReferenceMaterial,
	)
	return err
}

func (r *quizRepository) FindByID(ctx context.Context, id uuid.UUID) (*assessments.Quiz, error) {
	row := executorForContext(ctx, r.db).QueryRowContext(ctx,
		`SELECT `+quizColumns+` FROM quizzes WHERE id = $1`, id)
	quiz, err := scanQuiz(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("quiz not found")
	}
	if err != nil {
		return nil, err
	}
	return quiz, nil
}

func (r *quizRepository) findMany(ctx context.Context, where string, arg any) ([]*assessments.Quiz, error) {
	rows, err := executorForContext(ctx, r.db).QueryContext(ctx,
		`SELECT `+quizColumns+` FROM quizzes WHERE `+where+` ORDER BY created_at ASC`, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quizzes []*assessments.Quiz
	for rows.Next() {
		quiz, err := scanQuiz(rows)
		if err != nil {
			return nil, err
		}
		quizzes = append(quizzes, quiz)
	}
	return quizzes, rows.Err()
}

func (r *quizRepository) FindByCourseID(ctx context.Context, courseID uuid.UUID) ([]*assessments.Quiz, error) {
	return r.findMany(ctx, "course_id = $1", courseID)
}

func (r *quizRepository) FindByLessonID(ctx context.Context, lessonID uuid.UUID) ([]*assessments.Quiz, error) {
	return r.findMany(ctx, "lesson_id = $1", lessonID)
}

func (r *quizRepository) Update(ctx context.Context, quiz *assessments.Quiz) error {
	// lesson_id is now updated too (it was previously ignored on update).
	query := `
		UPDATE quizzes
		SET title = $2, time_limit_seconds = $3, max_attempts = $4,
			passing_score_percent = $5, shuffle_questions = $6,
			show_answers_after_submission = $7, is_free = $8, is_published = $9, updated_at = $10,
			lesson_id = $11, description = $12, instructions = $13, mode = $14,
			shuffle_options = $15, partial_credit = $16, available_from = $17,
			available_until = $18, calculator_policy = $19, reference_material = $20
		WHERE id = $1
	`
	_, err := executorForContext(ctx, r.db).ExecContext(ctx, query,
		quiz.ID, quiz.Title, quiz.TimeLimitSeconds, quiz.MaxAttempts,
		quiz.PassingScorePercent, quiz.ShuffleQuestions,
		quiz.ShowAnswersAfterSubmission, quiz.IsFree, quiz.IsPublished, quiz.UpdatedAt,
		quiz.LessonID, quiz.Description, quiz.Instructions, quizModeOrDefault(quiz.Mode),
		quiz.ShuffleOptions, quiz.PartialCredit, quiz.AvailableFrom,
		quiz.AvailableUntil, calculatorPolicyOrDefault(quiz.CalculatorPolicy), quiz.ReferenceMaterial,
	)
	return err
}

func (r *quizRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := executorForContext(ctx, r.db).ExecContext(ctx, `DELETE FROM quizzes WHERE id = $1`, id)
	return err
}
