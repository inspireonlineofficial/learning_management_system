package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"lms-backend/internal/domain/assessments"

	"github.com/google/uuid"
)

type assignmentSubmissionRepository struct {
	db *sql.DB
}

// NewAssignmentSubmissionRepository creates a new assignment submission repository
func NewAssignmentSubmissionRepository(db *sql.DB) assessments.AssignmentSubmissionRepository {
	return &assignmentSubmissionRepository{db: db}
}

func (r *assignmentSubmissionRepository) Create(ctx context.Context, submission *assessments.AssignmentSubmission) error {
	query := `
		INSERT INTO assignment_submissions (
			id, assignment_id, student_id, status, text_content,
			submitted_at, is_late, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		submission.ID,
		submission.AssignmentID,
		submission.StudentID,
		submission.Status,
		submission.TextContent,
		submission.SubmittedAt,
		submission.IsLate,
		submission.CreatedAt,
		submission.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create submission: %w", err)
	}

	return nil
}

func (r *assignmentSubmissionRepository) FindByID(ctx context.Context, id uuid.UUID) (*assessments.AssignmentSubmission, error) {
	query := `
		SELECT id, assignment_id, student_id, status, text_content,
			submitted_at, is_late, created_at, updated_at
		FROM assignment_submissions
		WHERE id = $1
	`

	submission := &assessments.AssignmentSubmission{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&submission.ID,
		&submission.AssignmentID,
		&submission.StudentID,
		&submission.Status,
		&submission.TextContent,
		&submission.SubmittedAt,
		&submission.IsLate,
		&submission.CreatedAt,
		&submission.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("submission not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find submission: %w", err)
	}

	return submission, nil
}

func (r *assignmentSubmissionRepository) FindByAssignmentAndStudent(ctx context.Context, assignmentID, studentID uuid.UUID) (*assessments.AssignmentSubmission, error) {
	query := `
		SELECT id, assignment_id, student_id, status, text_content,
			submitted_at, is_late, created_at, updated_at
		FROM assignment_submissions
		WHERE assignment_id = $1 AND student_id = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	submission := &assessments.AssignmentSubmission{}
	err := r.db.QueryRowContext(ctx, query, assignmentID, studentID).Scan(
		&submission.ID,
		&submission.AssignmentID,
		&submission.StudentID,
		&submission.Status,
		&submission.TextContent,
		&submission.SubmittedAt,
		&submission.IsLate,
		&submission.CreatedAt,
		&submission.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find submission: %w", err)
	}

	return submission, nil
}

func (r *assignmentSubmissionRepository) FindByAssignmentID(ctx context.Context, assignmentID uuid.UUID, page, limit int) ([]*assessments.AssignmentSubmission, int, error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM assignment_submissions WHERE assignment_id = $1`
	if err := r.db.QueryRowContext(ctx, countQuery, assignmentID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count submissions: %w", err)
	}

	// Fetch paginated results
	offset := (page - 1) * limit
	query := `
		SELECT id, assignment_id, student_id, status, text_content,
			submitted_at, is_late, created_at, updated_at
		FROM assignment_submissions
		WHERE assignment_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, assignmentID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find submissions: %w", err)
	}
	defer rows.Close()

	var submissions []*assessments.AssignmentSubmission
	for rows.Next() {
		submission := &assessments.AssignmentSubmission{}
		err := rows.Scan(
			&submission.ID,
			&submission.AssignmentID,
			&submission.StudentID,
			&submission.Status,
			&submission.TextContent,
			&submission.SubmittedAt,
			&submission.IsLate,
			&submission.CreatedAt,
			&submission.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan submission: %w", err)
		}
		submissions = append(submissions, submission)
	}

	return submissions, total, nil
}

func (r *assignmentSubmissionRepository) FindDraftByAssignmentAndStudent(ctx context.Context, assignmentID, studentID uuid.UUID) (*assessments.AssignmentSubmission, error) {
	query := `
		SELECT id, assignment_id, student_id, status, text_content,
			submitted_at, is_late, created_at, updated_at
		FROM assignment_submissions
		WHERE assignment_id = $1 AND student_id = $2 AND status = 'draft'
		ORDER BY created_at DESC
		LIMIT 1
	`

	submission := &assessments.AssignmentSubmission{}
	err := r.db.QueryRowContext(ctx, query, assignmentID, studentID).Scan(
		&submission.ID,
		&submission.AssignmentID,
		&submission.StudentID,
		&submission.Status,
		&submission.TextContent,
		&submission.SubmittedAt,
		&submission.IsLate,
		&submission.CreatedAt,
		&submission.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find draft submission: %w", err)
	}

	return submission, nil
}

func (r *assignmentSubmissionRepository) Update(ctx context.Context, submission *assessments.AssignmentSubmission) error {
	query := `
		UPDATE assignment_submissions
		SET status = $2, text_content = $3, submitted_at = $4, is_late = $5, updated_at = $6
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		submission.ID,
		submission.Status,
		submission.TextContent,
		submission.SubmittedAt,
		submission.IsLate,
		submission.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update submission: %w", err)
	}

	return nil
}

func (r *assignmentSubmissionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM assignment_submissions WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete submission: %w", err)
	}

	return nil
}
