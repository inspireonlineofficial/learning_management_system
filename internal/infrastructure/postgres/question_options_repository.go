package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"lms-backend/internal/domain/assessments"

	"github.com/google/uuid"
)

type questionOptionRepository struct {
	db *sql.DB
}

// NewQuestionOptionRepository creates a new question option repository
func NewQuestionOptionRepository(db *sql.DB) assessments.QuestionOptionRepository {
	return &questionOptionRepository{db: db}
}

func (r *questionOptionRepository) Create(ctx context.Context, option *assessments.QuestionOption) error {
	query := `
		INSERT INTO question_options (
			id, question_id, body, content_type, image_url, is_correct, position
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		option.ID, option.QuestionID, option.Body, option.ContentType, option.ImageURL, option.IsCorrect, option.Position,
	)

	return err
}

func (r *questionOptionRepository) CreateBatch(ctx context.Context, options []*assessments.QuestionOption) error {
	if len(options) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO question_options (
			id, question_id, body, content_type, image_url, is_correct, position
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, option := range options {
		_, err := stmt.ExecContext(ctx,
			option.ID, option.QuestionID, option.Body, option.ContentType, option.ImageURL, option.IsCorrect, option.Position,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *questionOptionRepository) FindByQuestionID(ctx context.Context, questionID uuid.UUID) ([]*assessments.QuestionOption, error) {
	query := `
		SELECT id, question_id, body, content_type, image_url, is_correct, position
		FROM question_options
		WHERE question_id = $1
		ORDER BY position ASC
	`

	rows, err := r.db.QueryContext(ctx, query, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []*assessments.QuestionOption
	for rows.Next() {
		option := &assessments.QuestionOption{}
		err := rows.Scan(
			&option.ID, &option.QuestionID, &option.Body, &option.ContentType, &option.ImageURL, &option.IsCorrect, &option.Position,
		)
		if err != nil {
			return nil, err
		}
		options = append(options, option)
	}

	return options, rows.Err()
}

func (r *questionOptionRepository) FindByQuestionIDs(ctx context.Context, questionIDs []uuid.UUID) (map[uuid.UUID][]*assessments.QuestionOption, error) {
	if len(questionIDs) == 0 {
		return make(map[uuid.UUID][]*assessments.QuestionOption), nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(questionIDs))
	args := make([]interface{}, len(questionIDs))
	for i, id := range questionIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, question_id, body, content_type, image_url, is_correct, position
		FROM question_options
		WHERE question_id IN (%s)
		ORDER BY question_id, position ASC
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	optionsMap := make(map[uuid.UUID][]*assessments.QuestionOption)
	for rows.Next() {
		option := &assessments.QuestionOption{}
		err := rows.Scan(
			&option.ID, &option.QuestionID, &option.Body, &option.ContentType, &option.ImageURL, &option.IsCorrect, &option.Position,
		)
		if err != nil {
			return nil, err
		}
		optionsMap[option.QuestionID] = append(optionsMap[option.QuestionID], option)
	}

	return optionsMap, rows.Err()
}

func (r *questionOptionRepository) Update(ctx context.Context, option *assessments.QuestionOption) error {
	query := `
		UPDATE question_options
		SET body = $2, content_type = $3, image_url = $4, is_correct = $5, position = $6
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		option.ID, option.Body, option.ContentType, option.ImageURL, option.IsCorrect, option.Position,
	)

	return err
}

func (r *questionOptionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM question_options WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
