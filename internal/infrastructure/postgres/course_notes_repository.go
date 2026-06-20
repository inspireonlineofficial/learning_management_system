package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"lms-backend/internal/domain/courses"

	"github.com/google/uuid"
)

type courseNoteRepository struct {
	db *sql.DB
}

// NewCourseNoteRepository creates a new course note repository.
func NewCourseNoteRepository(db *sql.DB) courses.CourseNoteRepository {
	return &courseNoteRepository{db: db}
}

func (r *courseNoteRepository) Create(ctx context.Context, note *courses.CourseNote) error {
	query := `
		INSERT INTO course_notes (
			id, course_id, module_id, lesson_id, title, content, file_url,
			is_free, is_published, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.ExecContext(ctx, query,
		note.ID, note.CourseID, note.ModuleID, note.LessonID, note.Title, note.Content, note.FileURL,
		note.IsFree, note.IsPublished, note.CreatedAt, note.UpdatedAt,
	)
	return err
}

func (r *courseNoteRepository) FindByID(ctx context.Context, id uuid.UUID) (*courses.CourseNote, error) {
	query := `
		SELECT id, course_id, module_id, lesson_id, title, content, file_url,
			is_free, is_published, created_at, updated_at, deleted_at
		FROM course_notes
		WHERE id = $1 AND deleted_at IS NULL
	`
	note := &courses.CourseNote{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&note.ID, &note.CourseID, &note.ModuleID, &note.LessonID, &note.Title, &note.Content, &note.FileURL,
		&note.IsFree, &note.IsPublished, &note.CreatedAt, &note.UpdatedAt, &note.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("course note not found")
	}
	return note, err
}

func (r *courseNoteRepository) FindByCourseID(ctx context.Context, courseID uuid.UUID) ([]*courses.CourseNote, error) {
	query := `
		SELECT id, course_id, module_id, lesson_id, title, content, file_url,
			is_free, is_published, created_at, updated_at, deleted_at
		FROM course_notes
		WHERE course_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []*courses.CourseNote
	for rows.Next() {
		note := &courses.CourseNote{}
		if err := rows.Scan(
			&note.ID, &note.CourseID, &note.ModuleID, &note.LessonID, &note.Title, &note.Content, &note.FileURL,
			&note.IsFree, &note.IsPublished, &note.CreatedAt, &note.UpdatedAt, &note.DeletedAt,
		); err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	return notes, rows.Err()
}

func (r *courseNoteRepository) Update(ctx context.Context, note *courses.CourseNote) error {
	query := `
		UPDATE course_notes
		SET module_id = $2, lesson_id = $3, title = $4, content = $5, file_url = $6,
			is_free = $7, is_published = $8, updated_at = $9
		WHERE id = $1 AND deleted_at IS NULL
	`
	note.UpdatedAt = time.Now().UTC()
	_, err := r.db.ExecContext(ctx, query,
		note.ID, note.ModuleID, note.LessonID, note.Title, note.Content, note.FileURL,
		note.IsFree, note.IsPublished, note.UpdatedAt,
	)
	return err
}

func (r *courseNoteRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE course_notes SET deleted_at = $1, updated_at = $1 WHERE id = $2 AND deleted_at IS NULL`,
		time.Now().UTC(), id,
	)
	return err
}
