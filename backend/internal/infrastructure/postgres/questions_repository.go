package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"lms-backend/internal/domain/assessments"

	"github.com/google/uuid"
)

type questionRepository struct {
	db *sql.DB
}

// NewQuestionRepository creates a new question repository. All methods join
// an ambient transaction (see TxRunner) when one is present in the context.
func NewQuestionRepository(db *sql.DB) assessments.QuestionRepository {
	return &questionRepository{db: db}
}

const questionColumns = `id, quiz_id, body, type, content_type, image_url, marks, is_required,
	position, explanation, created_at, updated_at, hint, settings`

const insertQuestionSQL = `
	INSERT INTO questions (` + questionColumns + `)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
`

func questionSettingsJSON(settings assessments.QuestionSettings) []byte {
	raw, err := json.Marshal(settings)
	if err != nil {
		return []byte("{}")
	}
	return raw
}

func scanQuestion(row rowScanner) (*assessments.Question, error) {
	question := &assessments.Question{}
	var explanation sql.NullString
	var settings []byte
	err := row.Scan(
		&question.ID, &question.QuizID, &question.Body, &question.Type,
		&question.ContentType, &question.ImageURL, &question.Marks, &question.IsRequired,
		&question.Position, &explanation, &question.CreatedAt, &question.UpdatedAt,
		&question.Hint, &settings,
	)
	if err != nil {
		return nil, err
	}
	if explanation.Valid {
		question.Explanation = explanation.String
	}
	if len(settings) > 0 {
		_ = json.Unmarshal(settings, &question.Settings)
	}
	return question, nil
}

func (r *questionRepository) insert(ctx context.Context, exec sqlExecutor, question *assessments.Question) error {
	_, err := exec.ExecContext(ctx, insertQuestionSQL,
		question.ID, question.QuizID, question.Body, question.Type,
		question.ContentType, question.ImageURL, question.Marks, question.IsRequired,
		question.Position, question.Explanation, question.CreatedAt, question.UpdatedAt,
		question.Hint, questionSettingsJSON(question.Settings),
	)
	return err
}

func (r *questionRepository) Create(ctx context.Context, question *assessments.Question) error {
	return r.insert(ctx, executorForContext(ctx, r.db), question)
}

func (r *questionRepository) CreateBatch(ctx context.Context, questions []*assessments.Question) error {
	if len(questions) == 0 {
		return nil
	}
	if TxFromContext(ctx) != nil {
		exec := executorForContext(ctx, r.db)
		for _, question := range questions {
			if err := r.insert(ctx, exec, question); err != nil {
				return err
			}
		}
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, question := range questions {
		if err := r.insert(ctx, tx, question); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *questionRepository) FindByID(ctx context.Context, id uuid.UUID) (*assessments.Question, error) {
	row := executorForContext(ctx, r.db).QueryRowContext(ctx,
		`SELECT `+questionColumns+` FROM questions WHERE id = $1`, id)
	question, err := scanQuestion(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("question not found")
	}
	return question, err
}

func (r *questionRepository) FindByQuizID(ctx context.Context, quizID uuid.UUID) ([]*assessments.Question, error) {
	rows, err := executorForContext(ctx, r.db).QueryContext(ctx,
		`SELECT `+questionColumns+` FROM questions WHERE quiz_id = $1 ORDER BY position ASC, created_at ASC`, quizID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []*assessments.Question
	for rows.Next() {
		question, err := scanQuestion(rows)
		if err != nil {
			return nil, err
		}
		questions = append(questions, question)
	}
	return questions, rows.Err()
}

func (r *questionRepository) Update(ctx context.Context, question *assessments.Question) error {
	query := `
		UPDATE questions
		SET body = $2, type = $3, content_type = $4, image_url = $5, marks = $6,
			is_required = $7, position = $8, explanation = $9, updated_at = $10,
			hint = $11, settings = $12
		WHERE id = $1
	`
	_, err := executorForContext(ctx, r.db).ExecContext(ctx, query,
		question.ID, question.Body, question.Type, question.ContentType, question.ImageURL,
		question.Marks, question.IsRequired, question.Position, question.Explanation, question.UpdatedAt,
		question.Hint, questionSettingsJSON(question.Settings),
	)
	return err
}

func (r *questionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := executorForContext(ctx, r.db).ExecContext(ctx, `DELETE FROM questions WHERE id = $1`, id)
	return err
}
