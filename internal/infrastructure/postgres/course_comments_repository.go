package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"lms-backend/internal/domain/courses"

	"github.com/google/uuid"
)

type courseCommentRepository struct {
	db *sql.DB
}

// NewCourseCommentRepository creates a course discussion repository.
func NewCourseCommentRepository(db *sql.DB) courses.CourseCommentRepository {
	return &courseCommentRepository{db: db}
}

func (r *courseCommentRepository) Create(ctx context.Context, comment *courses.CourseComment) error {
	query := `
		INSERT INTO course_comments (
			id, course_id, module_id, lesson_id, quiz_id, user_id, parent_comment_id,
			content, is_pinned, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.ExecContext(ctx, query,
		comment.ID, comment.CourseID, comment.ModuleID, comment.LessonID, comment.QuizID,
		comment.UserID, comment.ParentCommentID, comment.Content, comment.IsPinned,
		comment.CreatedAt, comment.UpdatedAt,
	)
	return err
}

func (r *courseCommentRepository) FindByID(ctx context.Context, id uuid.UUID) (*courses.CourseComment, error) {
	query := `
		SELECT id, course_id, module_id, lesson_id, quiz_id, user_id, parent_comment_id,
			content, is_pinned, deleted_at, created_at, updated_at
		FROM course_comments
		WHERE id = $1 AND deleted_at IS NULL
	`
	comment := &courses.CourseComment{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&comment.ID, &comment.CourseID, &comment.ModuleID, &comment.LessonID, &comment.QuizID,
		&comment.UserID, &comment.ParentCommentID, &comment.Content, &comment.IsPinned,
		&comment.DeletedAt, &comment.CreatedAt, &comment.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("comment not found")
	}
	return comment, err
}

func (r *courseCommentRepository) FindByCourseID(ctx context.Context, courseID uuid.UUID, page, limit int) ([]*courses.CourseComment, int, error) {
	offset := (page - 1) * limit
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM course_comments WHERE course_id = $1 AND deleted_at IS NULL`, courseID).Scan(&total); err != nil {
		return nil, 0, err
	}
	query := `
		SELECT id, course_id, module_id, lesson_id, quiz_id, user_id, parent_comment_id,
			content, is_pinned, deleted_at, created_at, updated_at
		FROM course_comments
		WHERE course_id = $1 AND deleted_at IS NULL
		ORDER BY is_pinned DESC, created_at ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, query, courseID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var comments []*courses.CourseComment
	for rows.Next() {
		comment := &courses.CourseComment{}
		if err := rows.Scan(
			&comment.ID, &comment.CourseID, &comment.ModuleID, &comment.LessonID, &comment.QuizID,
			&comment.UserID, &comment.ParentCommentID, &comment.Content, &comment.IsPinned,
			&comment.DeletedAt, &comment.CreatedAt, &comment.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		comments = append(comments, comment)
	}
	return comments, total, rows.Err()
}

func (r *courseCommentRepository) Update(ctx context.Context, comment *courses.CourseComment) error {
	query := `
		UPDATE course_comments
		SET content = $2, is_pinned = $3, updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, comment.ID, comment.Content, comment.IsPinned, comment.UpdatedAt)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("comment not found")
	}
	return nil
}

func (r *courseCommentRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `UPDATE course_comments SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("comment not found")
	}
	return nil
}
