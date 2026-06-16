package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"lms-backend/internal/domain/courses"

	"github.com/google/uuid"
)

type videoRepository struct {
	db *sql.DB
}

// NewVideoRepository creates a new video repository
func NewVideoRepository(db *sql.DB) courses.VideoRepository {
	return &videoRepository{db: db}
}

func (r *videoRepository) Create(ctx context.Context, video *courses.Video) error {
	query := `
		INSERT INTO videos (
			id, course_id, uploader_id, rustfs_key, status,
			duration_seconds, thumbnail_rustfs_key, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		video.ID, video.CourseID, video.UploaderID, video.RustFSKey,
		video.Status, video.DurationSeconds, video.ThumbnailRustFSKey,
		video.CreatedAt, video.UpdatedAt,
	)

	return err
}

func (r *videoRepository) FindByID(ctx context.Context, id uuid.UUID) (*courses.Video, error) {
	query := `
		SELECT id, course_id, uploader_id, rustfs_key, status,
			duration_seconds, thumbnail_rustfs_key, created_at, updated_at
		FROM videos
		WHERE id = $1
	`

	video := &courses.Video{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&video.ID, &video.CourseID, &video.UploaderID, &video.RustFSKey,
		&video.Status, &video.DurationSeconds, &video.ThumbnailRustFSKey,
		&video.CreatedAt, &video.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("video not found")
	}

	return video, err
}

func (r *videoRepository) Update(ctx context.Context, video *courses.Video) error {
	query := `
		UPDATE videos
		SET status = $2, duration_seconds = $3, thumbnail_rustfs_key = $4, updated_at = $5
		WHERE id = $1
	`

	video.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		video.ID, video.Status, video.DurationSeconds,
		video.ThumbnailRustFSKey, video.UpdatedAt,
	)

	return err
}
