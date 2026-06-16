package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"lms-backend/internal/domain/courses"

	"github.com/google/uuid"
)

type chapterRepository struct {
	db *sql.DB
}

// NewChapterRepository creates a new chapter repository
func NewChapterRepository(db *sql.DB) courses.ChapterRepository {
	return &chapterRepository{db: db}
}

func (r *chapterRepository) Create(ctx context.Context, chapter *courses.Chapter) error {
	query := `
		INSERT INTO chapters (id, module_id, title, position, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		chapter.ID, chapter.ModuleID, chapter.Title, chapter.Position,
		chapter.CreatedAt, chapter.UpdatedAt,
	)

	return err
}

func (r *chapterRepository) FindByID(ctx context.Context, id uuid.UUID) (*courses.Chapter, error) {
	query := `
		SELECT id, module_id, title, position, created_at, updated_at, deleted_at
		FROM chapters
		WHERE id = $1 AND deleted_at IS NULL
	`

	chapter := &courses.Chapter{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&chapter.ID, &chapter.ModuleID, &chapter.Title, &chapter.Position,
		&chapter.CreatedAt, &chapter.UpdatedAt, &chapter.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("chapter not found")
	}

	return chapter, err
}

func (r *chapterRepository) FindByModuleID(ctx context.Context, moduleID uuid.UUID) ([]*courses.Chapter, error) {
	query := `
		SELECT id, module_id, title, position, created_at, updated_at, deleted_at
		FROM chapters
		WHERE module_id = $1 AND deleted_at IS NULL
		ORDER BY position ASC
	`

	rows, err := r.db.QueryContext(ctx, query, moduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chapters []*courses.Chapter
	for rows.Next() {
		chapter := &courses.Chapter{}
		err := rows.Scan(
			&chapter.ID, &chapter.ModuleID, &chapter.Title, &chapter.Position,
			&chapter.CreatedAt, &chapter.UpdatedAt, &chapter.DeletedAt,
		)
		if err != nil {
			return nil, err
		}
		chapters = append(chapters, chapter)
	}

	return chapters, rows.Err()
}

func (r *chapterRepository) Update(ctx context.Context, chapter *courses.Chapter) error {
	query := `
		UPDATE chapters
		SET title = $2, position = $3, updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
	`

	chapter.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		chapter.ID, chapter.Title, chapter.Position, chapter.UpdatedAt,
	)

	return err
}

func (r *chapterRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE chapters SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

func (r *chapterRepository) CascadeSoftDelete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()

	// Soft delete all lessons in this chapter
	_, err = tx.ExecContext(ctx, `
		UPDATE lessons
		SET deleted_at = $1
		WHERE chapter_id = $2 AND deleted_at IS NULL
	`, now, id)
	if err != nil {
		return err
	}

	// Soft delete the chapter itself
	_, err = tx.ExecContext(ctx, `
		UPDATE chapters
		SET deleted_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`, now, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *chapterRepository) Reorder(ctx context.Context, moduleID uuid.UUID, positions map[uuid.UUID]int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `UPDATE chapters SET position = $1, updated_at = $2 WHERE id = $3 AND module_id = $4 AND deleted_at IS NULL`
	now := time.Now()

	for chapterID, position := range positions {
		_, err := tx.ExecContext(ctx, query, position, now, chapterID, moduleID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
