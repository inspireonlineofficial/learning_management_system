package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"lms-backend/internal/domain/assessments"

	"github.com/google/uuid"
)

type submissionGradeRepository struct {
	db *sql.DB
}

// NewSubmissionGradeRepository creates a new submission grade repository
func NewSubmissionGradeRepository(db *sql.DB) assessments.SubmissionGradeRepository {
	return &submissionGradeRepository{db: db}
}

func (r *submissionGradeRepository) Create(ctx context.Context, grade *assessments.SubmissionGrade) error {
	query := `
		INSERT INTO submission_grades (
			id, submission_id, graded_by, score, feedback,
			revision_requested, revision_notes, graded_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		grade.ID,
		grade.SubmissionID,
		grade.GradedBy,
		grade.Score,
		grade.Feedback,
		grade.RevisionRequested,
		grade.RevisionNotes,
		grade.GradedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create submission grade: %w", err)
	}

	return nil
}

func (r *submissionGradeRepository) FindByID(ctx context.Context, id uuid.UUID) (*assessments.SubmissionGrade, error) {
	query := `
		SELECT id, submission_id, graded_by, score, feedback,
			revision_requested, revision_notes, graded_at
		FROM submission_grades
		WHERE id = $1
	`

	grade := &assessments.SubmissionGrade{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&grade.ID,
		&grade.SubmissionID,
		&grade.GradedBy,
		&grade.Score,
		&grade.Feedback,
		&grade.RevisionRequested,
		&grade.RevisionNotes,
		&grade.GradedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("submission grade not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find submission grade: %w", err)
	}

	return grade, nil
}

func (r *submissionGradeRepository) FindBySubmissionID(ctx context.Context, submissionID uuid.UUID) ([]*assessments.SubmissionGrade, error) {
	query := `
		SELECT id, submission_id, graded_by, score, feedback,
			revision_requested, revision_notes, graded_at
		FROM submission_grades
		WHERE submission_id = $1
		ORDER BY graded_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, submissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find submission grades: %w", err)
	}
	defer rows.Close()

	var grades []*assessments.SubmissionGrade
	for rows.Next() {
		grade := &assessments.SubmissionGrade{}
		err := rows.Scan(
			&grade.ID,
			&grade.SubmissionID,
			&grade.GradedBy,
			&grade.Score,
			&grade.Feedback,
			&grade.RevisionRequested,
			&grade.RevisionNotes,
			&grade.GradedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan submission grade: %w", err)
		}
		grades = append(grades, grade)
	}

	return grades, nil
}

func (r *submissionGradeRepository) GetLatestGrade(ctx context.Context, submissionID uuid.UUID) (*assessments.SubmissionGrade, error) {
	query := `
		SELECT id, submission_id, graded_by, score, feedback,
			revision_requested, revision_notes, graded_at
		FROM submission_grades
		WHERE submission_id = $1
		ORDER BY graded_at DESC
		LIMIT 1
	`

	grade := &assessments.SubmissionGrade{}
	err := r.db.QueryRowContext(ctx, query, submissionID).Scan(
		&grade.ID,
		&grade.SubmissionID,
		&grade.GradedBy,
		&grade.Score,
		&grade.Feedback,
		&grade.RevisionRequested,
		&grade.RevisionNotes,
		&grade.GradedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find latest submission grade: %w", err)
	}

	return grade, nil
}
