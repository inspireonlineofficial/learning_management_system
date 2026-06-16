package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"lms-backend/internal/domain/courses"

	"github.com/google/uuid"
)

type moduleRepository struct {
	db *sql.DB
}

// NewModuleRepository creates a new module repository
func NewModuleRepository(db *sql.DB) courses.ModuleRepository {
	return &moduleRepository{db: db}
}

func (r *moduleRepository) Create(ctx context.Context, module *courses.Module) error {
	query := `
		INSERT INTO modules (id, course_id, title, position, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		module.ID, module.CourseID, module.Title, module.Position,
		module.CreatedAt, module.UpdatedAt,
	)

	return err
}

func (r *moduleRepository) FindByID(ctx context.Context, id uuid.UUID) (*courses.Module, error) {
	query := `
		SELECT id, course_id, title, position, created_at, updated_at, deleted_at
		FROM modules
		WHERE id = $1 AND deleted_at IS NULL
	`

	module := &courses.Module{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&module.ID, &module.CourseID, &module.Title, &module.Position,
		&module.CreatedAt, &module.UpdatedAt, &module.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("module not found")
	}

	return module, err
}

func (r *moduleRepository) FindByCourseID(ctx context.Context, courseID uuid.UUID) ([]*courses.Module, error) {
	query := `
		SELECT id, course_id, title, position, created_at, updated_at, deleted_at
		FROM modules
		WHERE course_id = $1 AND deleted_at IS NULL
		ORDER BY position ASC
	`

	rows, err := r.db.QueryContext(ctx, query, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []*courses.Module
	for rows.Next() {
		module := &courses.Module{}
		err := rows.Scan(
			&module.ID, &module.CourseID, &module.Title, &module.Position,
			&module.CreatedAt, &module.UpdatedAt, &module.DeletedAt,
		)
		if err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}

	return modules, rows.Err()
}

func (r *moduleRepository) Update(ctx context.Context, module *courses.Module) error {
	query := `
		UPDATE modules
		SET title = $2, position = $3, updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
	`

	module.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		module.ID, module.Title, module.Position, module.UpdatedAt,
	)

	return err
}

func (r *moduleRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE modules SET deleted_at = $1 WHERE id = $2 AND deleted_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

func (r *moduleRepository) CascadeSoftDelete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()

	// Soft delete all lessons in chapters of this module
	_, err = tx.ExecContext(ctx, `
		UPDATE lessons
		SET deleted_at = $1
		WHERE chapter_id IN (
			SELECT id FROM chapters WHERE module_id = $2 AND deleted_at IS NULL
		) AND deleted_at IS NULL
	`, now, id)
	if err != nil {
		return err
	}

	// Soft delete all chapters of this module
	_, err = tx.ExecContext(ctx, `
		UPDATE chapters
		SET deleted_at = $1
		WHERE module_id = $2 AND deleted_at IS NULL
	`, now, id)
	if err != nil {
		return err
	}

	// Soft delete the module itself
	_, err = tx.ExecContext(ctx, `
		UPDATE modules
		SET deleted_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`, now, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *moduleRepository) Reorder(ctx context.Context, courseID uuid.UUID, positions map[uuid.UUID]int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `UPDATE modules SET position = $1, updated_at = $2 WHERE id = $3 AND course_id = $4 AND deleted_at IS NULL`
	now := time.Now()

	for moduleID, position := range positions {
		_, err := tx.ExecContext(ctx, query, position, now, moduleID, courseID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
