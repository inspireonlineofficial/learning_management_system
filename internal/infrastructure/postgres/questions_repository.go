package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"lms-backend/internal/domain/assessments"

	"github.com/google/uuid"
)

type questionRepository struct {
	db *sql.DB
}

// NewQuestionRepository creates a new question repository
func NewQuestionRepository(db *sql.DB) assessments.QuestionRepository {
	return &questionRepository{db: db}
}

func (r *questionRepository) Create(ctx context.Context, question *assessments.Question) error {
	query := `
		INSERT INTO questions (
			id, quiz_id, body, type, content_type, image_url, marks, is_required, position, explanation, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.ExecContext(ctx, query,
		question.ID, question.QuizID, question.Body, question.Type,
		question.ContentType, question.ImageURL, question.Marks, question.IsRequired,
		question.Position, question.Explanation, question.CreatedAt, question.UpdatedAt,
	)

	return err
}

func (r *questionRepository) CreateBatch(ctx context.Context, questions []*assessments.Question) error {
	if len(questions) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO questions (
			id, quiz_id, body, type, content_type, image_url, marks, is_required, position, explanation, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, question := range questions {
		_, err := stmt.ExecContext(ctx,
			question.ID, question.QuizID, question.Body, question.Type,
			question.ContentType, question.ImageURL, question.Marks, question.IsRequired,
			question.Position, question.Explanation, question.CreatedAt, question.UpdatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *questionRepository) FindByID(ctx context.Context, id uuid.UUID) (*assessments.Question, error) {
	query := `
		SELECT id, quiz_id, body, type, content_type, image_url, marks, is_required, position, explanation, created_at, updated_at
		FROM questions
		WHERE id = $1
	`

	question := &assessments.Question{}
	var explanation sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&question.ID, &question.QuizID, &question.Body, &question.Type,
		&question.ContentType, &question.ImageURL, &question.Marks, &question.IsRequired,
		&question.Position, &explanation, &question.CreatedAt, &question.UpdatedAt,
	)
	if explanation.Valid {
		question.Explanation = explanation.String
	}

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("question not found")
	}

	return question, err
}

func (r *questionRepository) FindByQuizID(ctx context.Context, quizID uuid.UUID) ([]*assessments.Question, error) {
	query := `
		SELECT id, quiz_id, body, type, content_type, image_url, marks, is_required, position, explanation, created_at, updated_at
		FROM questions
		WHERE quiz_id = $1
		ORDER BY position ASC
	`

	rows, err := r.db.QueryContext(ctx, query, quizID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []*assessments.Question
	for rows.Next() {
		question := &assessments.Question{}
		var explanation sql.NullString
		err := rows.Scan(
			&question.ID, &question.QuizID, &question.Body, &question.Type,
			&question.ContentType, &question.ImageURL, &question.Marks, &question.IsRequired,
			&question.Position, &explanation, &question.CreatedAt, &question.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if explanation.Valid {
			question.Explanation = explanation.String
		}
		questions = append(questions, question)
	}

	return questions, rows.Err()
}

func (r *questionRepository) Update(ctx context.Context, question *assessments.Question) error {
	query := `
		UPDATE questions
		SET body = $2, type = $3, content_type = $4, image_url = $5, marks = $6,
			is_required = $7, position = $8, explanation = $9, updated_at = $10
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		question.ID, question.Body, question.Type, question.ContentType, question.ImageURL,
		question.Marks, question.IsRequired, question.Position, question.Explanation, question.UpdatedAt,
	)

	return err
}

func (r *questionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM questions WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
