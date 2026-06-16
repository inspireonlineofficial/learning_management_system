package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"lms-backend/internal/domain/courses"

	"github.com/google/uuid"
)

type lessonRepository struct {
	db *sql.DB
}

// NewLessonRepository creates a new lesson repository
func NewLessonRepository(db *sql.DB) courses.LessonRepository {
	return &lessonRepository{db: db}
}

func (r *lessonRepository) Create(ctx context.Context, lesson *courses.Lesson) error {
	query := `
		INSERT INTO lessons (
			id, chapter_id, title, type, video_id, duration_seconds,
			is_free_preview, is_downloadable, position, status,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.ExecContext(ctx, query,
		lesson.ID, lesson.ChapterID, lesson.Title, lesson.Type,
		lesson.VideoID, lesson.DurationSeconds, lesson.IsFreePreview,
		lesson.IsDownloadable, lesson.Position, lesson.Status,
		lesson.CreatedAt, lesson.UpdatedAt,
	)

	return err
}

func (r *lessonRepository) FindByID(ctx context.Context, id uuid.UUID) (*courses.Lesson, error) {
	query := `
		SELECT id, chapter_id, title, type, video_id, duration_seconds,
			is_free_preview, is_downloadable, position, status,
			created_at, updated_at, deleted_at
		FROM lessons
		WHERE id = $1 AND deleted_at IS NULL
	`

	lesson := &courses.Lesson{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&lesson.ID, &lesson.ChapterID, &lesson.Title, &lesson.Type,
		&lesson.VideoID, &lesson.DurationSeconds, &lesson.IsFreePreview,
		&lesson.IsDownloadable, &lesson.Position, &lesson.Status,
		&lesson.CreatedAt, &lesson.UpdatedAt, &lesson.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("lesson not found")
	}

	return lesson, err
}

func (r *lessonRepository) FindByChapterID(ctx context.Context, chapterID uuid.UUID) ([]*courses.Lesson, error) {
	query := `
		SELECT id, chapter_id, title, type, video_id, duration_seconds,
			is_free_preview, is_downloadable, position, status,
			created_at, updated_at, deleted_at
		FROM lessons
		WHERE chapter_id = $1 AND deleted_at IS NULL
		ORDER BY position ASC
	`

	rows, err := r.db.QueryContext(ctx, query, chapterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lessons []*courses.Lesson
	for rows.Next() {
		lesson := &courses.Lesson{}
		err := rows.Scan(
			&lesson.ID, &lesson.ChapterID, &lesson.Title, &lesson.Type,
			&lesson.VideoID, &lesson.DurationSeconds, &lesson.IsFreePreview,
			&lesson.IsDownloadable, &lesson.Position, &lesson.Status,
			&lesson.CreatedAt, &lesson.UpdatedAt, &lesson.DeletedAt,
		)
		if err != nil {
			return nil, err
		}
		lessons = append(lessons, lesson)
	}

	return lessons, rows.Err()
}

func (r *lessonRepository) Update(ctx context.Context, lesson *courses.Lesson) error {
	query := `
		UPDATE lessons
		SET title = $2, type = $3, video_id = $4, duration_seconds = $5,
			is_free_preview = $6, is_downloadable = $7, position = $8,
			status = $9, updated_at = $10
		WHERE id = $1 AND deleted_at IS NULL
	`

	lesson.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		lesson.ID, lesson.Title, lesson.Type, lesson.VideoID,
		lesson.DurationSeconds, lesson.IsFreePreview, lesson.IsDownloadable,
		lesson.Position, lesson.Status, lesson.UpdatedAt,
	)

	return err
}

func (r *lessonRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE lessons SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

func (r *lessonRepository) Reorder(ctx context.Context, chapterID uuid.UUID, positions map[uuid.UUID]int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `UPDATE lessons SET position = $1, updated_at = $2 WHERE id = $3 AND chapter_id = $4 AND deleted_at IS NULL`
	now := time.Now()

	for lessonID, position := range positions {
		_, err := tx.ExecContext(ctx, query, position, now, lessonID, chapterID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
