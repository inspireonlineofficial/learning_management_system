package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"lms-backend/internal/domain/courses"

	"github.com/google/uuid"
)

type courseRepository struct {
	db *sql.DB
}

// NewCourseRepository creates a new course repository
func NewCourseRepository(db *sql.DB) courses.CourseRepository {
	return &courseRepository{db: db}
}

func (r *courseRepository) Create(ctx context.Context, course *courses.Course) error {
	query := `
		INSERT INTO courses (
			id, teacher_id, title, slug, short_description, description,
			subject, level, price_type, price, currency, prerequisites,
			visibility, learning_outcomes, requirements, target_audience,
			estimated_duration_minutes, thumbnail_url, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
	`

	_, err := r.db.ExecContext(ctx, query,
		course.ID, course.TeacherID, course.Title, course.Slug,
		course.ShortDescription, course.Description, course.Subject,
		course.Level, course.PriceType, course.Price, course.Currency,
		course.Prerequisites, course.Visibility, course.LearningOutcomes,
		course.Requirements, course.TargetAudience, course.EstimatedDurationMinutes,
		course.ThumbnailURL, course.Status,
		course.CreatedAt, course.UpdatedAt,
	)

	return err
}

func (r *courseRepository) FindByID(ctx context.Context, id uuid.UUID) (*courses.Course, error) {
	query := `
		SELECT id, teacher_id, title, slug, short_description, description,
			subject, level, price_type, price, currency, prerequisites,
			visibility, learning_outcomes, requirements, target_audience,
			estimated_duration_minutes, thumbnail_url, status, rating_average, rating_count, total_enrolled,
			published_at, created_at, updated_at, deleted_at
		FROM courses
		WHERE id = $1 AND deleted_at IS NULL
	`

	course := &courses.Course{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&course.ID, &course.TeacherID, &course.Title, &course.Slug,
		&course.ShortDescription, &course.Description, &course.Subject,
		&course.Level, &course.PriceType, &course.Price, &course.Currency,
		&course.Prerequisites, &course.Visibility, &course.LearningOutcomes,
		&course.Requirements, &course.TargetAudience, &course.EstimatedDurationMinutes,
		&course.ThumbnailURL, &course.Status,
		&course.RatingAverage, &course.RatingCount, &course.TotalEnrolled,
		&course.PublishedAt, &course.CreatedAt, &course.UpdatedAt, &course.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("course not found")
	}

	return course, err
}

func (r *courseRepository) FindBySlug(ctx context.Context, slug string) (*courses.Course, error) {
	query := `
		SELECT id, teacher_id, title, slug, short_description, description,
			subject, level, price_type, price, currency, prerequisites,
			visibility, learning_outcomes, requirements, target_audience,
			estimated_duration_minutes, thumbnail_url, status, rating_average, rating_count, total_enrolled,
			published_at, created_at, updated_at, deleted_at
		FROM courses
		WHERE slug = $1 AND deleted_at IS NULL
	`

	course := &courses.Course{}
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&course.ID, &course.TeacherID, &course.Title, &course.Slug,
		&course.ShortDescription, &course.Description, &course.Subject,
		&course.Level, &course.PriceType, &course.Price, &course.Currency,
		&course.Prerequisites, &course.Visibility, &course.LearningOutcomes,
		&course.Requirements, &course.TargetAudience, &course.EstimatedDurationMinutes,
		&course.ThumbnailURL, &course.Status,
		&course.RatingAverage, &course.RatingCount, &course.TotalEnrolled,
		&course.PublishedAt, &course.CreatedAt, &course.UpdatedAt, &course.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("course not found")
	}

	return course, err
}

func (r *courseRepository) FindByTeacherID(ctx context.Context, teacherID uuid.UUID, page, limit int) ([]*courses.Course, int, error) {
	offset := (page - 1) * limit

	countQuery := `SELECT COUNT(*) FROM courses WHERE teacher_id = $1 AND deleted_at IS NULL`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, teacherID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, teacher_id, title, slug, short_description, description,
			subject, level, price_type, price, currency, prerequisites,
			visibility, learning_outcomes, requirements, target_audience,
			estimated_duration_minutes, thumbnail_url, status, rating_average, rating_count, total_enrolled,
			published_at, created_at, updated_at, deleted_at
		FROM courses
		WHERE teacher_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, teacherID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var courseList []*courses.Course
	for rows.Next() {
		course := &courses.Course{}
		err := rows.Scan(
			&course.ID, &course.TeacherID, &course.Title, &course.Slug,
			&course.ShortDescription, &course.Description, &course.Subject,
			&course.Level, &course.PriceType, &course.Price, &course.Currency,
			&course.Prerequisites, &course.Visibility, &course.LearningOutcomes,
			&course.Requirements, &course.TargetAudience, &course.EstimatedDurationMinutes,
			&course.ThumbnailURL, &course.Status,
			&course.RatingAverage, &course.RatingCount, &course.TotalEnrolled,
			&course.PublishedAt, &course.CreatedAt, &course.UpdatedAt, &course.DeletedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		courseList = append(courseList, course)
	}

	return courseList, total, rows.Err()
}

func (r *courseRepository) Update(ctx context.Context, course *courses.Course) error {
	query := `
		UPDATE courses
		SET title = $2, slug = $3, short_description = $4, description = $5,
			subject = $6, level = $7, price_type = $8, price = $9, currency = $10,
			prerequisites = $11, visibility = $12, learning_outcomes = $13,
			requirements = $14, target_audience = $15, estimated_duration_minutes = $16,
			thumbnail_url = $17, status = $18, published_at = $19, updated_at = $20
		WHERE id = $1 AND deleted_at IS NULL
	`

	course.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		course.ID, course.Title, course.Slug, course.ShortDescription,
		course.Description, course.Subject, course.Level, course.PriceType,
		course.Price, course.Currency, course.Prerequisites, course.Visibility,
		course.LearningOutcomes, course.Requirements, course.TargetAudience,
		course.EstimatedDurationMinutes, course.ThumbnailURL, course.Status,
		course.PublishedAt, course.UpdatedAt,
	)

	return err
}

func (r *courseRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE courses SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

func (r *courseRepository) List(ctx context.Context, filters courses.CourseFilters, page, limit int) ([]*courses.Course, int, error) {
	offset := (page - 1) * limit

	// Build WHERE clause
	whereClauses := []string{"deleted_at IS NULL"}
	args := []interface{}{}
	argPos := 1

	if filters.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, filters.Status)
		argPos++
	}

	if filters.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", argPos, argPos))
		args = append(args, "%"+filters.Search+"%")
		argPos++
	}

	if filters.Subject != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("subject = $%d", argPos))
		args = append(args, filters.Subject)
		argPos++
	}

	if filters.Level != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("level = $%d", argPos))
		args = append(args, filters.Level)
		argPos++
	}

	if filters.PriceType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("price_type = $%d", argPos))
		args = append(args, filters.PriceType)
		argPos++
	}

	if filters.TeacherID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("teacher_id = $%d", argPos))
		args = append(args, *filters.TeacherID)
		argPos++
	}

	if filters.MinPrice != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("price >= $%d", argPos))
		args = append(args, *filters.MinPrice)
		argPos++
	}

	if filters.MaxPrice != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("price <= $%d", argPos))
		args = append(args, *filters.MaxPrice)
		argPos++
	}

	whereClause := strings.Join(whereClauses, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM courses WHERE %s", whereClause)
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Build ORDER BY clause
	orderBy := "created_at DESC"
	switch filters.SortBy {
	case "newest":
		orderBy = "created_at DESC"
	case "popular":
		orderBy = "total_enrolled DESC"
	case "rating":
		orderBy = "rating_average DESC"
	case "price_asc":
		orderBy = "price ASC"
	case "price_desc":
		orderBy = "price DESC"
	}

	// Query courses
	query := fmt.Sprintf(`
		SELECT id, teacher_id, title, slug, short_description, description,
			subject, level, price_type, price, currency, prerequisites,
			visibility, learning_outcomes, requirements, target_audience,
			estimated_duration_minutes, thumbnail_url, status, rating_average, rating_count, total_enrolled,
			published_at, created_at, updated_at, deleted_at
		FROM courses
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argPos, argPos+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var courseList []*courses.Course
	for rows.Next() {
		course := &courses.Course{}
		err := rows.Scan(
			&course.ID, &course.TeacherID, &course.Title, &course.Slug,
			&course.ShortDescription, &course.Description, &course.Subject,
			&course.Level, &course.PriceType, &course.Price, &course.Currency,
			&course.Prerequisites, &course.Visibility, &course.LearningOutcomes,
			&course.Requirements, &course.TargetAudience, &course.EstimatedDurationMinutes,
			&course.ThumbnailURL, &course.Status,
			&course.RatingAverage, &course.RatingCount, &course.TotalEnrolled,
			&course.PublishedAt, &course.CreatedAt, &course.UpdatedAt, &course.DeletedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		courseList = append(courseList, course)
	}

	return courseList, total, rows.Err()
}

func (r *courseRepository) CountPublishedLessons(ctx context.Context, courseID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM lessons l
		JOIN chapters c ON l.chapter_id = c.id
		JOIN modules m ON c.module_id = m.id
		WHERE m.course_id = $1
			AND l.status = 'published'
			AND l.deleted_at IS NULL
			AND c.deleted_at IS NULL
			AND m.deleted_at IS NULL
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, courseID).Scan(&count)
	return count, err
}
