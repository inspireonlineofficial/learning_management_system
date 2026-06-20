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

// NewQuizRepository creates a new quiz repository
func NewQuizRepository(db *sql.DB) assessments.QuizRepository {
	return &quizRepository{db: db}
}

func (r *quizRepository) Create(ctx context.Context, quiz *assessments.Quiz) error {
	query := `
		INSERT INTO quizzes (
			id, course_id, lesson_id, title, time_limit_seconds, max_attempts,
			passing_score_percent, shuffle_questions, show_answers_after_submission,
			is_free, is_published, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.db.ExecContext(ctx, query,
		quiz.ID, quiz.CourseID, quiz.LessonID, quiz.Title, quiz.TimeLimitSeconds,
		quiz.MaxAttempts, quiz.PassingScorePercent, quiz.ShuffleQuestions,
		quiz.ShowAnswersAfterSubmission, quiz.IsFree, quiz.IsPublished, quiz.CreatedAt, quiz.UpdatedAt,
	)

	return err
}

func (r *quizRepository) FindByID(ctx context.Context, id uuid.UUID) (*assessments.Quiz, error) {
	query := `
		SELECT id, course_id, lesson_id, title, time_limit_seconds, max_attempts,
			passing_score_percent, shuffle_questions, show_answers_after_submission,
			is_free, is_published, created_at, updated_at
		FROM quizzes
		WHERE id = $1
	`

	quiz := &assessments.Quiz{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&quiz.ID, &quiz.CourseID, &quiz.LessonID, &quiz.Title, &quiz.TimeLimitSeconds,
		&quiz.MaxAttempts, &quiz.PassingScorePercent, &quiz.ShuffleQuestions,
		&quiz.ShowAnswersAfterSubmission, &quiz.IsFree, &quiz.IsPublished, &quiz.CreatedAt, &quiz.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("quiz not found")
	}

	return quiz, err
}

func (r *quizRepository) FindByCourseID(ctx context.Context, courseID uuid.UUID) ([]*assessments.Quiz, error) {
	query := `
		SELECT id, course_id, lesson_id, title, time_limit_seconds, max_attempts,
			passing_score_percent, shuffle_questions, show_answers_after_submission,
			is_free, is_published, created_at, updated_at
		FROM quizzes
		WHERE course_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quizzes []*assessments.Quiz
	for rows.Next() {
		quiz := &assessments.Quiz{}
		err := rows.Scan(
			&quiz.ID, &quiz.CourseID, &quiz.LessonID, &quiz.Title, &quiz.TimeLimitSeconds,
			&quiz.MaxAttempts, &quiz.PassingScorePercent, &quiz.ShuffleQuestions,
			&quiz.ShowAnswersAfterSubmission, &quiz.IsFree, &quiz.IsPublished, &quiz.CreatedAt, &quiz.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		quizzes = append(quizzes, quiz)
	}

	return quizzes, rows.Err()
}

func (r *quizRepository) FindByLessonID(ctx context.Context, lessonID uuid.UUID) ([]*assessments.Quiz, error) {
	query := `
		SELECT id, course_id, lesson_id, title, time_limit_seconds, max_attempts,
			passing_score_percent, shuffle_questions, show_answers_after_submission,
			is_free, is_published, created_at, updated_at
		FROM quizzes
		WHERE lesson_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, lessonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quizzes []*assessments.Quiz
	for rows.Next() {
		quiz := &assessments.Quiz{}
		err := rows.Scan(
			&quiz.ID, &quiz.CourseID, &quiz.LessonID, &quiz.Title, &quiz.TimeLimitSeconds,
			&quiz.MaxAttempts, &quiz.PassingScorePercent, &quiz.ShuffleQuestions,
			&quiz.ShowAnswersAfterSubmission, &quiz.IsFree, &quiz.IsPublished, &quiz.CreatedAt, &quiz.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		quizzes = append(quizzes, quiz)
	}

	return quizzes, rows.Err()
}

func (r *quizRepository) Update(ctx context.Context, quiz *assessments.Quiz) error {
	query := `
		UPDATE quizzes
		SET title = $2, time_limit_seconds = $3, max_attempts = $4,
			passing_score_percent = $5, shuffle_questions = $6,
			show_answers_after_submission = $7, is_free = $8, is_published = $9, updated_at = $10
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		quiz.ID, quiz.Title, quiz.TimeLimitSeconds, quiz.MaxAttempts,
		quiz.PassingScorePercent, quiz.ShuffleQuestions,
		quiz.ShowAnswersAfterSubmission, quiz.IsFree, quiz.IsPublished, quiz.UpdatedAt,
	)

	return err
}

func (r *quizRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM quizzes WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
