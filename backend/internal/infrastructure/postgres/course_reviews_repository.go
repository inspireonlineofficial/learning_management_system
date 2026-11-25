package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"lms-backend/internal/domain/courses"

	"github.com/google/uuid"
)

type courseReviewRepository struct {
	db *sql.DB
}

// NewCourseReviewRepository creates a new course review repository
func NewCourseReviewRepository(db *sql.DB) courses.CourseReviewRepository {
	return &courseReviewRepository{db: db}
}

func (r *courseReviewRepository) Upsert(ctx context.Context, review *courses.CourseReview) error {
	query := `
		INSERT INTO course_reviews (id, course_id, student_id, rating, comment, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (course_id, student_id)
		DO UPDATE SET rating = $4, comment = $5, updated_at = $7
	`

	_, err := r.db.ExecContext(ctx, query,
		review.ID, review.CourseID, review.StudentID, review.Rating,
		review.Comment, review.CreatedAt, review.UpdatedAt,
	)

	return err
}

func (r *courseReviewRepository) FindByID(ctx context.Context, id uuid.UUID) (*courses.CourseReview, error) {
	query := `
		SELECT id, course_id, student_id, rating, comment, created_at, updated_at
		FROM course_reviews
		WHERE id = $1
	`

	review := &courses.CourseReview{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&review.ID, &review.CourseID, &review.StudentID, &review.Rating,
		&review.Comment, &review.CreatedAt, &review.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("review not found")
	}

	return review, err
}

func (r *courseReviewRepository) FindByCourseID(ctx context.Context, courseID uuid.UUID, page, limit int) ([]*courses.CourseReview, int, error) {
	offset := (page - 1) * limit

	countQuery := `SELECT COUNT(*) FROM course_reviews WHERE course_id = $1`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, courseID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, course_id, student_id, rating, comment, created_at, updated_at
		FROM course_reviews
		WHERE course_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, courseID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reviews []*courses.CourseReview
	for rows.Next() {
		review := &courses.CourseReview{}
		err := rows.Scan(
			&review.ID, &review.CourseID, &review.StudentID, &review.Rating,
			&review.Comment, &review.CreatedAt, &review.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		reviews = append(reviews, review)
	}

	return reviews, total, rows.Err()
}

func (r *courseReviewRepository) FindByStudentAndCourse(ctx context.Context, studentID, courseID uuid.UUID) (*courses.CourseReview, error) {
	query := `
		SELECT id, course_id, student_id, rating, comment, created_at, updated_at
		FROM course_reviews
		WHERE student_id = $1 AND course_id = $2
	`

	review := &courses.CourseReview{}
	err := r.db.QueryRowContext(ctx, query, studentID, courseID).Scan(
		&review.ID, &review.CourseID, &review.StudentID, &review.Rating,
		&review.Comment, &review.CreatedAt, &review.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No existing review
	}

	return review, err
}

func (r *courseReviewRepository) DeleteByStudentAndCourse(ctx context.Context, studentID, courseID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM course_reviews WHERE student_id = $1 AND course_id = $2`, studentID, courseID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("review not found")
	}
	return nil
}

func (r *courseReviewRepository) GetRatingDistribution(ctx context.Context, courseID uuid.UUID) (map[int]int, error) {
	query := `
		SELECT rating, COUNT(*) as count
		FROM course_reviews
		WHERE course_id = $1
		GROUP BY rating
	`

	rows, err := r.db.QueryContext(ctx, query, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	distribution := make(map[int]int)
	for rows.Next() {
		var rating, count int
		err := rows.Scan(&rating, &count)
		if err != nil {
			return nil, err
		}
		distribution[rating] = count
	}

	return distribution, rows.Err()
}
