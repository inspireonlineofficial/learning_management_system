package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"lms-backend/internal/domain/assessments"

	"github.com/google/uuid"
)

type assignmentRepository struct {
	db *sql.DB
}

// NewAssignmentRepository creates a new assignment repository
func NewAssignmentRepository(db *sql.DB) assessments.AssignmentRepository {
	return &assignmentRepository{db: db}
}

func (r *assignmentRepository) Create(ctx context.Context, assignment *assessments.Assignment) error {
	query := `
		INSERT INTO assignments (
			id, course_id, title, description, due_at, submission_type,
			max_file_size_mb, allow_late_submission, total_marks, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(ctx, query,
		assignment.ID,
		assignment.CourseID,
		assignment.Title,
		assignment.Description,
		assignment.DueAt,
		assignment.SubmissionType,
		assignment.MaxFileSizeMB,
		assignment.AllowLateSubmission,
		assignment.TotalMarks,
		assignment.CreatedAt,
		assignment.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create assignment: %w", err)
	}

	return nil
}

func (r *assignmentRepository) FindByID(ctx context.Context, id uuid.UUID) (*assessments.Assignment, error) {
	query := `
		SELECT id, course_id, title, description, due_at, submission_type,
			max_file_size_mb, allow_late_submission, total_marks, created_at, updated_at
		FROM assignments
		WHERE id = $1
	`

	assignment := &assessments.Assignment{}
	var description sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&assignment.ID,
		&assignment.CourseID,
		&assignment.Title,
		&description,
		&assignment.DueAt,
		&assignment.SubmissionType,
		&assignment.MaxFileSizeMB,
		&assignment.AllowLateSubmission,
		&assignment.TotalMarks,
		&assignment.CreatedAt,
		&assignment.UpdatedAt,
	)
	if description.Valid {
		assignment.Description = description.String
	}

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("assignment not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find assignment: %w", err)
	}

	return assignment, nil
}

func (r *assignmentRepository) FindByCourseID(ctx context.Context, courseID uuid.UUID) ([]*assessments.Assignment, error) {
	query := `
		SELECT id, course_id, title, description, due_at, submission_type,
			max_file_size_mb, allow_late_submission, total_marks, created_at, updated_at
		FROM assignments
		WHERE course_id = $1
		ORDER BY due_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to find assignments: %w", err)
	}
	defer rows.Close()

	var assignments []*assessments.Assignment
	for rows.Next() {
		assignment := &assessments.Assignment{}
		var description sql.NullString
		err := rows.Scan(
			&assignment.ID,
			&assignment.CourseID,
			&assignment.Title,
			&description,
			&assignment.DueAt,
			&assignment.SubmissionType,
			&assignment.MaxFileSizeMB,
			&assignment.AllowLateSubmission,
			&assignment.TotalMarks,
			&assignment.CreatedAt,
			&assignment.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan assignment: %w", err)
		}
		if description.Valid {
			assignment.Description = description.String
		}
		assignments = append(assignments, assignment)
	}

	return assignments, nil
}

func (r *assignmentRepository) Update(ctx context.Context, assignment *assessments.Assignment) error {
	query := `
		UPDATE assignments
		SET title = $2, description = $3, due_at = $4, submission_type = $5,
			max_file_size_mb = $6, allow_late_submission = $7, total_marks = $8, updated_at = $9
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		assignment.ID,
		assignment.Title,
		assignment.Description,
		assignment.DueAt,
		assignment.SubmissionType,
		assignment.MaxFileSizeMB,
		assignment.AllowLateSubmission,
		assignment.TotalMarks,
		assignment.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update assignment: %w", err)
	}

	return nil
}

func (r *assignmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM assignments WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete assignment: %w", err)
	}

	return nil
}
