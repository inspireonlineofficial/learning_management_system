package postgres

import (
	"context"
	"database/sql"
	"time"

	domainpoints "lms-backend/internal/domain/points"

	"github.com/google/uuid"
)

// PointEventRepository implements domain/points.PointEventRepository.
type PointEventRepository struct {
	db *sql.DB
}

// NewPointEventRepository creates a new PointEventRepository.
func NewPointEventRepository(db *sql.DB) *PointEventRepository {
	return &PointEventRepository{db: db}
}

func (r *PointEventRepository) Create(ctx context.Context, event *domainpoints.PointEvent) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO point_events (id, student_id, type, source_id, source_title, points, bonus_points, earned_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		event.ID, event.StudentID, string(event.Type), event.SourceID,
		event.SourceTitle, event.Points, event.BonusPoints, event.EarnedAt,
	)
	return err
}

func (r *PointEventRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainpoints.PointEvent, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, student_id, type, source_id, source_title, points, bonus_points, earned_at
		FROM point_events WHERE id = $1`, id)
	return scanPointEvent(row)
}

func (r *PointEventRepository) FindByStudentID(ctx context.Context, studentID uuid.UUID, page, limit int) ([]*domainpoints.PointEvent, int, error) {
	offset := (page - 1) * limit

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM point_events WHERE student_id = $1`, studentID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, student_id, type, source_id, source_title, points, bonus_points, earned_at
		FROM point_events WHERE student_id = $1
		ORDER BY earned_at DESC
		LIMIT $2 OFFSET $3`, studentID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*domainpoints.PointEvent
	for rows.Next() {
		e, err := scanPointEventRow(rows)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}
	return events, total, rows.Err()
}

func (r *PointEventRepository) ExistsForSourceOnDay(ctx context.Context, studentID, sourceID uuid.UUID, eventType domainpoints.PointEventType, day time.Time) (bool, error) {
	nextDay := day.AddDate(0, 0, 1)
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM point_events
			WHERE student_id = $1 AND source_id = $2 AND type = $3
			  AND earned_at >= $4 AND earned_at < $5
		)`, studentID, sourceID, string(eventType), day, nextDay).Scan(&exists)
	return exists, err
}

func (r *PointEventRepository) ExistsPassingForSource(ctx context.Context, studentID, sourceID uuid.UUID, eventType domainpoints.PointEventType) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM point_events
			WHERE student_id = $1 AND source_id = $2 AND type = $3
		)`, studentID, sourceID, string(eventType)).Scan(&exists)
	return exists, err
}

func (r *PointEventRepository) SumByStudentID(ctx context.Context, studentID uuid.UUID) (int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(points + bonus_points), 0)
		FROM point_events WHERE student_id = $1`, studentID).Scan(&total)
	return total, err
}

func (r *PointEventRepository) SumByStudentIDSince(ctx context.Context, studentID uuid.UUID, since time.Time) (int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(points + bonus_points), 0)
		FROM point_events WHERE student_id = $1 AND earned_at >= $2`, studentID, since).Scan(&total)
	return total, err
}

// PointsConfigRepository implements domain/points.PointsConfigRepository.
type PointsConfigRepository struct {
	db *sql.DB
}

// NewPointsConfigRepository creates a new PointsConfigRepository.
func NewPointsConfigRepository(db *sql.DB) *PointsConfigRepository {
	return &PointsConfigRepository{db: db}
}

func (r *PointsConfigRepository) Get(ctx context.Context) (*domainpoints.PointsConfig, error) {
	cfg := &domainpoints.PointsConfig{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, points_per_video, points_per_quiz_pass, bonus_points_perfect_score, updated_at, updated_by
		FROM points_config WHERE id = 1`).Scan(
		&cfg.ID, &cfg.PointsPerVideo, &cfg.PointsPerQuizPass, &cfg.BonusPointsPerfectScore,
		&cfg.UpdatedAt, &cfg.UpdatedBy,
	)
	if err == sql.ErrNoRows {
		// Return defaults if not yet seeded
		return &domainpoints.PointsConfig{
			ID:                      1,
			PointsPerVideo:          10,
			PointsPerQuizPass:       20,
			BonusPointsPerfectScore: 10,
		}, nil
	}
	return cfg, err
}

func (r *PointsConfigRepository) Update(ctx context.Context, cfg *domainpoints.PointsConfig) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO points_config (id, points_per_video, points_per_quiz_pass, bonus_points_perfect_score, updated_at, updated_by)
		VALUES (1, $1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			points_per_video = EXCLUDED.points_per_video,
			points_per_quiz_pass = EXCLUDED.points_per_quiz_pass,
			bonus_points_perfect_score = EXCLUDED.bonus_points_perfect_score,
			updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by`,
		cfg.PointsPerVideo, cfg.PointsPerQuizPass, cfg.BonusPointsPerfectScore,
		cfg.UpdatedAt, cfg.UpdatedBy,
	)
	return err
}

// PointsRankRepository implements application/points.PointsRankRepository.
type PointsRankRepository struct {
	db *sql.DB
}

// NewPointsRankRepository creates a new PointsRankRepository.
func NewPointsRankRepository(db *sql.DB) *PointsRankRepository {
	return &PointsRankRepository{db: db}
}

func (r *PointsRankRepository) CountStudentsWithMoreTotalPoints(ctx context.Context, totalPoints int) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT student_id)
		FROM point_events
		GROUP BY student_id
		HAVING SUM(points + bonus_points) > $1`, totalPoints).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return count, err
}

func (r *PointsRankRepository) CountStudentsWithMoreWeeklyPoints(ctx context.Context, weeklyPoints int, weekStart time.Time) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT student_id)
		FROM point_events
		WHERE earned_at >= $1
		GROUP BY student_id
		HAVING SUM(points + bonus_points) > $2`, weekStart, weeklyPoints).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return count, err
}

func (r *PointsRankRepository) FindEventsForDay(ctx context.Context, studentID uuid.UUID, day time.Time) ([]*domainpoints.PointEvent, error) {
	nextDay := day.AddDate(0, 0, 1)
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, student_id, type, source_id, source_title, points, bonus_points, earned_at
		FROM point_events
		WHERE student_id = $1 AND earned_at >= $2 AND earned_at < $3
		ORDER BY earned_at ASC`, studentID, day, nextDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domainpoints.PointEvent
	for rows.Next() {
		e, err := scanPointEventRow(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// scanPointEvent scans a single row into a PointEvent.
func scanPointEvent(row *sql.Row) (*domainpoints.PointEvent, error) {
	e := &domainpoints.PointEvent{}
	var eventType string
	err := row.Scan(&e.ID, &e.StudentID, &eventType, &e.SourceID, &e.SourceTitle, &e.Points, &e.BonusPoints, &e.EarnedAt)
	if err != nil {
		return nil, err
	}
	e.Type = domainpoints.PointEventType(eventType)
	return e, nil
}

// scanPointEventRow scans a rows.Next() row into a PointEvent.
func scanPointEventRow(rows *sql.Rows) (*domainpoints.PointEvent, error) {
	e := &domainpoints.PointEvent{}
	var eventType string
	err := rows.Scan(&e.ID, &e.StudentID, &eventType, &e.SourceID, &e.SourceTitle, &e.Points, &e.BonusPoints, &e.EarnedAt)
	if err != nil {
		return nil, err
	}
	e.Type = domainpoints.PointEventType(eventType)
	return e, nil
}
