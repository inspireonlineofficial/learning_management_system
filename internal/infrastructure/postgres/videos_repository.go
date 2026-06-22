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
			duration_seconds, thumbnail_rustfs_key,
			hls_manifest_key, transcoded_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(ctx, query,
		video.ID, video.CourseID, video.UploaderID, video.RustFSKey,
		video.Status, video.DurationSeconds, video.ThumbnailRustFSKey,
		video.HLSManifestKey, video.TranscodedAt,
		video.CreatedAt, video.UpdatedAt,
	)

	return err
}

func (r *videoRepository) FindByID(ctx context.Context, id uuid.UUID) (*courses.Video, error) {
	query := `
		SELECT id, course_id, uploader_id, rustfs_key, status,
			duration_seconds, thumbnail_rustfs_key,
			hls_manifest_key, transcoded_at,
			created_at, updated_at
		FROM videos
		WHERE id = $1
	`

	video := &courses.Video{}
	// hls_manifest_key is nullable (it is only populated after a successful
	// transcode). database/sql cannot scan a SQL NULL into a plain Go string,
	// so we route it through sql.NullString and flatten back to "" when the
	// column is NULL. Without this the worker crashes on every row that has
	// not yet been transcoded.
	var hlsManifestKey sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&video.ID, &video.CourseID, &video.UploaderID, &video.RustFSKey,
		&video.Status, &video.DurationSeconds, &video.ThumbnailRustFSKey,
		&hlsManifestKey, &video.TranscodedAt,
		&video.CreatedAt, &video.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("video not found")
		}
		return nil, err
	}
	video.HLSManifestKey = nullStringToString(hlsManifestKey)

	return video, nil
}

// nullStringToString flattens a sql.NullString into a plain string, returning
// "" when the SQL value is NULL. Used for nullable TEXT columns whose domain
// representation is a non-pointer string.
func nullStringToString(ns sql.NullString) string {
	if !ns.Valid {
		return ""
	}
	return ns.String
}

func (r *videoRepository) Update(ctx context.Context, video *courses.Video) error {
	query := `
		UPDATE videos
		SET status = $2, duration_seconds = $3, thumbnail_rustfs_key = $4,
			hls_manifest_key = $5, transcoded_at = $6, updated_at = $7
		WHERE id = $1
	`

	video.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		video.ID, video.Status, video.DurationSeconds,
		video.ThumbnailRustFSKey, video.HLSManifestKey, video.TranscodedAt,
		video.UpdatedAt,
	)

	return err
}

// Delete removes a video row. Used by the upload-init rollback path when the
// subsequent S3 multipart upload start fails: we'd rather not leave an orphan
// "processing" row that the teacher has to manually clean up.
func (r *videoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM videos WHERE id = $1`, id)
	return err
}
