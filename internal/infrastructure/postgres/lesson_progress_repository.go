package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"lms-backend/internal/domain/enrollments"

	"github.com/google/uuid"
)

type lessonProgressRepository struct {
	db *sql.DB
}

// NewLessonProgressRepository creates a new lesson progress repository
func NewLessonProgressRepository(db *sql.DB) enrollments.LessonProgressRepository {
	return &lessonProgressRepository{db: db}
}

// Upsert creates or updates lesson progress
func (r *lessonProgressRepository) Upsert(ctx context.Context, progress *enrollments.LessonProgress) error {
	query := `
		INSERT INTO lesson_progress (
			id, enrollment_id, lesson_id, position_seconds, watched_percent,
			completed, completed_at, last_watched_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (enrollment_id, lesson_id) DO UPDATE SET
			position_seconds = EXCLUDED.position_seconds,
			watched_percent = EXCLUDED.watched_percent,
			completed = EXCLUDED.completed,
			completed_at = EXCLUDED.completed_at,
			last_watched_at = EXCLUDED.last_watched_at
		RETURNING id
	`

	err := r.db.QueryRowContext(ctx, query,
		progress.ID,
		progress.EnrollmentID,
		progress.LessonID,
		progress.PositionSeconds,
		progress.WatchedPercent,
		progress.Completed,
		progress.CompletedAt,
		progress.LastWatchedAt,
	).Scan(&progress.ID)

	if err != nil {
		return fmt.Errorf("failed to upsert lesson progress: %w", err)
	}

	return nil
}

func (r *lessonProgressRepository) FindByID(ctx context.Context, id uuid.UUID) (*enrollments.LessonProgress, error) {
	query := `
		SELECT id, enrollment_id, lesson_id, position_seconds, watched_percent,
			completed, completed_at, last_watched_at
		FROM lesson_progress
		WHERE id = $1
	`

	progress := &enrollments.LessonProgress{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&progress.ID,
		&progress.EnrollmentID,
		&progress.LessonID,
		&progress.PositionSeconds,
		&progress.WatchedPercent,
		&progress.Completed,
		&progress.CompletedAt,
		&progress.LastWatchedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("lesson progress not found")
	}

	return progress, err
}

func (r *lessonProgressRepository) FindByEnrollmentAndLesson(ctx context.Context, enrollmentID, lessonID uuid.UUID) (*enrollments.LessonProgress, error) {
	query := `
		SELECT id, enrollment_id, lesson_id, position_seconds, watched_percent,
			completed, completed_at, last_watched_at
		FROM lesson_progress
		WHERE enrollment_id = $1 AND lesson_id = $2
	`

	progress := &enrollments.LessonProgress{}
	err := r.db.QueryRowContext(ctx, query, enrollmentID, lessonID).Scan(
		&progress.ID,
		&progress.EnrollmentID,
		&progress.LessonID,
		&progress.PositionSeconds,
		&progress.WatchedPercent,
		&progress.Completed,
		&progress.CompletedAt,
		&progress.LastWatchedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("lesson progress not found")
	}

	return progress, err
}

func (r *lessonProgressRepository) FindByEnrollmentID(ctx context.Context, enrollmentID uuid.UUID) ([]*enrollments.LessonProgress, error) {
	query := `
		SELECT id, enrollment_id, lesson_id, position_seconds, watched_percent,
			completed, completed_at, last_watched_at
		FROM lesson_progress
		WHERE enrollment_id = $1
		ORDER BY last_watched_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, enrollmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var progressList []*enrollments.LessonProgress
	for rows.Next() {
		progress := &enrollments.LessonProgress{}
		err := rows.Scan(
			&progress.ID,
			&progress.EnrollmentID,
			&progress.LessonID,
			&progress.PositionSeconds,
			&progress.WatchedPercent,
			&progress.Completed,
			&progress.CompletedAt,
			&progress.LastWatchedAt,
		)
		if err != nil {
			return nil, err
		}
		progressList = append(progressList, progress)
	}

	return progressList, rows.Err()
}

func (r *lessonProgressRepository) CountCompletedLessons(ctx context.Context, enrollmentID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM lesson_progress WHERE enrollment_id = $1 AND completed = true`

	var count int
	err := r.db.QueryRowContext(ctx, query, enrollmentID).Scan(&count)
	return count, err
}
