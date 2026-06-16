package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"lms-backend/internal/domain/assessments"

	"github.com/google/uuid"
)

type submissionFileRepository struct {
	db *sql.DB
}

// NewSubmissionFileRepository creates a new submission file repository
func NewSubmissionFileRepository(db *sql.DB) assessments.SubmissionFileRepository {
	return &submissionFileRepository{db: db}
}

func (r *submissionFileRepository) Create(ctx context.Context, file *assessments.SubmissionFile) error {
	query := `
		INSERT INTO submission_files (
			id, submission_id, rustfs_key, original_filename, mime_type, size_bytes
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		file.ID,
		file.SubmissionID,
		file.RustFSKey,
		file.OriginalFilename,
		file.MimeType,
		file.SizeBytes,
	)

	if err != nil {
		return fmt.Errorf("failed to create submission file: %w", err)
	}

	return nil
}

func (r *submissionFileRepository) CreateBatch(ctx context.Context, files []*assessments.SubmissionFile) error {
	if len(files) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO submission_files (
			id, submission_id, rustfs_key, original_filename, mime_type, size_bytes
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, file := range files {
		_, err := stmt.ExecContext(ctx,
			file.ID,
			file.SubmissionID,
			file.RustFSKey,
			file.OriginalFilename,
			file.MimeType,
			file.SizeBytes,
		)
		if err != nil {
			return fmt.Errorf("failed to insert submission file: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *submissionFileRepository) FindBySubmissionID(ctx context.Context, submissionID uuid.UUID) ([]*assessments.SubmissionFile, error) {
	query := `
		SELECT id, submission_id, rustfs_key, original_filename, mime_type, size_bytes
		FROM submission_files
		WHERE submission_id = $1
		ORDER BY original_filename ASC
	`

	rows, err := r.db.QueryContext(ctx, query, submissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find submission files: %w", err)
	}
	defer rows.Close()

	var files []*assessments.SubmissionFile
	for rows.Next() {
		file := &assessments.SubmissionFile{}
		err := rows.Scan(
			&file.ID,
			&file.SubmissionID,
			&file.RustFSKey,
			&file.OriginalFilename,
			&file.MimeType,
			&file.SizeBytes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan submission file: %w", err)
		}
		files = append(files, file)
	}

	return files, nil
}

func (r *submissionFileRepository) CountBySubmissionID(ctx context.Context, submissionID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM submission_files WHERE submission_id = $1`

	var count int
	err := r.db.QueryRowContext(ctx, query, submissionID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count submission files: %w", err)
	}

	return count, nil
}

func (r *submissionFileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM submission_files WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete submission file: %w", err)
	}

	return nil
}
