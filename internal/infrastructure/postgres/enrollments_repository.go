package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"lms-backend/internal/domain/enrollments"

	"github.com/google/uuid"
)

type enrollmentRepository struct {
	db *sql.DB
}

// NewEnrollmentRepository creates a new enrollment repository
func NewEnrollmentRepository(db *sql.DB) enrollments.EnrollmentRepository {
	return &enrollmentRepository{db: db}
}

// Create creates a new enrollment with INSERT ... ON CONFLICT DO NOTHING for idempotency
func (r *enrollmentRepository) Create(ctx context.Context, enrollment *enrollments.Enrollment) error {
	query := `
		INSERT INTO enrollments (
			id, student_id, course_id, enrollment_type, status,
			progress_percent, enrolled_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (student_id, course_id) DO NOTHING
	`

	exec := executorForContext(ctx, r.db)
	result, err := exec.ExecContext(ctx, query,
		enrollment.ID,
		enrollment.StudentID,
		enrollment.CourseID,
		enrollment.EnrollmentType,
		enrollment.Status,
		enrollment.ProgressPercent,
		enrollment.EnrolledAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create enrollment: %w", err)
	}

	// Check if a row was actually inserted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// If no rows were affected, it means the enrollment already exists (ON CONFLICT DO NOTHING)
	if rowsAffected == 0 {
		return fmt.Errorf("enrollment already exists: unique constraint violation")
	}

	return nil
}

func (r *enrollmentRepository) FindByID(ctx context.Context, id uuid.UUID) (*enrollments.Enrollment, error) {
	query := `
		SELECT id, student_id, course_id, enrollment_type, status,
			progress_percent, completed_at, enrolled_at
		FROM enrollments
		WHERE id = $1
	`

	enrollment := &enrollments.Enrollment{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&enrollment.ID,
		&enrollment.StudentID,
		&enrollment.CourseID,
		&enrollment.EnrollmentType,
		&enrollment.Status,
		&enrollment.ProgressPercent,
		&enrollment.CompletedAt,
		&enrollment.EnrolledAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("enrollment not found")
	}

	return enrollment, err
}

func (r *enrollmentRepository) FindByStudentAndCourse(ctx context.Context, studentID, courseID uuid.UUID) (*enrollments.Enrollment, error) {
	query := `
		SELECT id, student_id, course_id, enrollment_type, status,
			progress_percent, completed_at, enrolled_at
		FROM enrollments
		WHERE student_id = $1 AND course_id = $2
	`

	enrollment := &enrollments.Enrollment{}
	err := r.db.QueryRowContext(ctx, query, studentID, courseID).Scan(
		&enrollment.ID,
		&enrollment.StudentID,
		&enrollment.CourseID,
		&enrollment.EnrollmentType,
		&enrollment.Status,
		&enrollment.ProgressPercent,
		&enrollment.CompletedAt,
		&enrollment.EnrolledAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("enrollment not found")
	}

	return enrollment, err
}

func (r *enrollmentRepository) FindByStudentID(ctx context.Context, studentID uuid.UUID, page, limit int) ([]*enrollments.Enrollment, int, error) {
	offset := (page - 1) * limit

	// Count total
	countQuery := `SELECT COUNT(*) FROM enrollments WHERE student_id = $1`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, studentID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Query enrollments
	query := `
		SELECT id, student_id, course_id, enrollment_type, status,
			progress_percent, completed_at, enrolled_at
		FROM enrollments
		WHERE student_id = $1
		ORDER BY enrolled_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, studentID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var enrollmentList []*enrollments.Enrollment
	for rows.Next() {
		enrollment := &enrollments.Enrollment{}
		err := rows.Scan(
			&enrollment.ID,
			&enrollment.StudentID,
			&enrollment.CourseID,
			&enrollment.EnrollmentType,
			&enrollment.Status,
			&enrollment.ProgressPercent,
			&enrollment.CompletedAt,
			&enrollment.EnrolledAt,
		)
		if err != nil {
			return nil, 0, err
		}
		enrollmentList = append(enrollmentList, enrollment)
	}

	return enrollmentList, total, rows.Err()
}

func (r *enrollmentRepository) FindByCourseID(ctx context.Context, courseID uuid.UUID, page, limit int) ([]*enrollments.Enrollment, int, error) {
	offset := (page - 1) * limit

	// Count total
	countQuery := `SELECT COUNT(*) FROM enrollments WHERE course_id = $1`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, courseID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Query enrollments
	query := `
		SELECT id, student_id, course_id, enrollment_type, status,
			progress_percent, completed_at, enrolled_at
		FROM enrollments
		WHERE course_id = $1
		ORDER BY enrolled_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, courseID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var enrollmentList []*enrollments.Enrollment
	for rows.Next() {
		enrollment := &enrollments.Enrollment{}
		err := rows.Scan(
			&enrollment.ID,
			&enrollment.StudentID,
			&enrollment.CourseID,
			&enrollment.EnrollmentType,
			&enrollment.Status,
			&enrollment.ProgressPercent,
			&enrollment.CompletedAt,
			&enrollment.EnrolledAt,
		)
		if err != nil {
			return nil, 0, err
		}
		enrollmentList = append(enrollmentList, enrollment)
	}

	return enrollmentList, total, rows.Err()
}

func (r *enrollmentRepository) Update(ctx context.Context, enrollment *enrollments.Enrollment) error {
	query := `
		UPDATE enrollments
		SET status = $2, progress_percent = $3, completed_at = $4
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		enrollment.ID,
		enrollment.Status,
		enrollment.ProgressPercent,
		enrollment.CompletedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update enrollment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("enrollment not found")
	}

	return nil
}

func (r *enrollmentRepository) UpdateProgressPercent(ctx context.Context, enrollmentID uuid.UUID, progressPercent float64) error {
	query := `
		UPDATE enrollments
		SET progress_percent = $2
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, enrollmentID, progressPercent)
	if err != nil {
		return fmt.Errorf("failed to update progress percent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("enrollment not found")
	}

	return nil
}

func (r *enrollmentRepository) Exists(ctx context.Context, studentID, courseID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM enrollments WHERE student_id = $1 AND course_id = $2)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, studentID, courseID).Scan(&exists)
	return exists, err
}

// RecalculateProgressPercent recalculates the progress percentage based on completed lessons
func (r *enrollmentRepository) RecalculateProgressPercent(ctx context.Context, enrollmentID uuid.UUID) error {
	query := `
		WITH enrollment_course AS (
			SELECT course_id FROM enrollments WHERE id = $1
		),
		total_lessons AS (
			SELECT COUNT(*) as total
			FROM lessons l
			JOIN chapters c ON l.chapter_id = c.id
			JOIN modules m ON c.module_id = m.id
			JOIN enrollment_course ec ON m.course_id = ec.course_id
			WHERE l.deleted_at IS NULL
				AND c.deleted_at IS NULL
				AND m.deleted_at IS NULL
				AND l.status = 'published'
		),
		completed_lessons AS (
			SELECT COUNT(*) as completed
			FROM lesson_progress
			WHERE enrollment_id = $1 AND completed = true
		)
		UPDATE enrollments
		SET progress_percent = CASE
			WHEN (SELECT total FROM total_lessons) = 0 THEN 0
			ELSE ROUND((SELECT completed FROM completed_lessons)::numeric / (SELECT total FROM total_lessons)::numeric * 100, 2)
		END,
		completed_at = CASE
			WHEN (SELECT total FROM total_lessons) > 0 
				AND (SELECT completed FROM completed_lessons) = (SELECT total FROM total_lessons)
				AND completed_at IS NULL
			THEN NOW()
			ELSE completed_at
		END
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, enrollmentID)
	if err != nil {
		return fmt.Errorf("failed to recalculate progress percent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("enrollment not found")
	}

	return nil
}

// CountTotalLessons counts the total number of published lessons in a course
func (r *enrollmentRepository) CountTotalLessons(ctx context.Context, courseID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM lessons l
		JOIN chapters c ON l.chapter_id = c.id
		JOIN modules m ON c.module_id = m.id
		WHERE m.course_id = $1
			AND l.deleted_at IS NULL
			AND c.deleted_at IS NULL
			AND m.deleted_at IS NULL
			AND l.status = 'published'
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, courseID).Scan(&count)
	return count, err
}
