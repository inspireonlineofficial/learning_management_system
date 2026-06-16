package postgres

import (
	"context"
	"database/sql"
	"time"

	"lms-backend/internal/domain/analytics"

	"github.com/google/uuid"
)

// AnalyticsRepository implements domain/analytics.Repository.
type AnalyticsRepository struct {
	db *sql.DB
}

// NewAnalyticsRepository creates a new AnalyticsRepository.
func NewAnalyticsRepository(db *sql.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// UpsertEnrollmentStat inserts or updates the enrollment stat for a course on a given date.
func (r *AnalyticsRepository) UpsertEnrollmentStat(ctx context.Context, stat *analytics.EnrollmentStat) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO analytics_enrollment_stats
			(id, course_id, stat_date, total_enrolled, free_enrolled, paid_enrolled, aggregated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (course_id, stat_date) DO UPDATE SET
			total_enrolled = EXCLUDED.total_enrolled,
			free_enrolled  = EXCLUDED.free_enrolled,
			paid_enrolled  = EXCLUDED.paid_enrolled,
			aggregated_at  = EXCLUDED.aggregated_at`,
		stat.ID, stat.CourseID, stat.StatDate,
		stat.TotalEnrolled, stat.FreeEnrolled, stat.PaidEnrolled, stat.AggregatedAt,
	)
	return err
}

// UpsertProgressStat inserts or updates the progress stat for a module on a given date.
func (r *AnalyticsRepository) UpsertProgressStat(ctx context.Context, stat *analytics.ProgressStat) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO analytics_progress_stats
			(id, course_id, module_id, stat_date, total_students, completed_students, in_progress_students, aggregated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (course_id, module_id, stat_date) DO UPDATE SET
			total_students       = EXCLUDED.total_students,
			completed_students   = EXCLUDED.completed_students,
			in_progress_students = EXCLUDED.in_progress_students,
			aggregated_at        = EXCLUDED.aggregated_at`,
		stat.ID, stat.CourseID, stat.ModuleID, stat.StatDate,
		stat.TotalStudents, stat.CompletedStudents, stat.InProgressStudents, stat.AggregatedAt,
	)
	return err
}

// UpsertRevenueStat inserts or updates the revenue stat for a given date.
func (r *AnalyticsRepository) UpsertRevenueStat(ctx context.Context, stat *analytics.RevenueStat) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO analytics_revenue_stats
			(id, stat_date, total_revenue, course_revenue, book_revenue, aggregated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (stat_date) DO UPDATE SET
			total_revenue  = EXCLUDED.total_revenue,
			course_revenue = EXCLUDED.course_revenue,
			book_revenue   = EXCLUDED.book_revenue,
			aggregated_at  = EXCLUDED.aggregated_at`,
		stat.ID, stat.StatDate, stat.TotalRevenue, stat.CourseRevenue, stat.BookRevenue, stat.AggregatedAt,
	)
	return err
}

// UpsertDAUStat inserts or updates the DAU stat for a given date.
func (r *AnalyticsRepository) UpsertDAUStat(ctx context.Context, stat *analytics.DAUStat) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO analytics_dau_stats (id, stat_date, active_users, aggregated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (stat_date) DO UPDATE SET
			active_users  = EXCLUDED.active_users,
			aggregated_at = EXCLUDED.aggregated_at`,
		stat.ID, stat.StatDate, stat.ActiveUsers, stat.AggregatedAt,
	)
	return err
}

// GetEnrollmentStatsByDateRange returns enrollment stats for all courses in the date range.
func (r *AnalyticsRepository) GetEnrollmentStatsByDateRange(ctx context.Context, from, to time.Time) ([]*analytics.EnrollmentStat, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, course_id, stat_date, total_enrolled, free_enrolled, paid_enrolled, aggregated_at
		FROM analytics_enrollment_stats
		WHERE stat_date >= $1 AND stat_date <= $2
		ORDER BY stat_date ASC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEnrollmentStats(rows)
}

// GetEnrollmentStatsByCourse returns enrollment stats for a specific course in the date range.
func (r *AnalyticsRepository) GetEnrollmentStatsByCourse(ctx context.Context, courseID uuid.UUID, from, to time.Time) ([]*analytics.EnrollmentStat, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, course_id, stat_date, total_enrolled, free_enrolled, paid_enrolled, aggregated_at
		FROM analytics_enrollment_stats
		WHERE course_id = $1 AND stat_date >= $2 AND stat_date <= $3
		ORDER BY stat_date ASC`, courseID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEnrollmentStats(rows)
}

// GetProgressStatsByCourse returns progress stats for all modules of a course in the date range.
func (r *AnalyticsRepository) GetProgressStatsByCourse(ctx context.Context, courseID uuid.UUID, from, to time.Time) ([]*analytics.ProgressStat, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, course_id, module_id, stat_date, total_students, completed_students, in_progress_students, aggregated_at
		FROM analytics_progress_stats
		WHERE course_id = $1 AND stat_date >= $2 AND stat_date <= $3
		ORDER BY stat_date ASC, module_id ASC`, courseID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProgressStats(rows)
}

// GetRevenueStatsByDateRange returns revenue stats for the date range.
func (r *AnalyticsRepository) GetRevenueStatsByDateRange(ctx context.Context, from, to time.Time) ([]*analytics.RevenueStat, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, stat_date, total_revenue, course_revenue, book_revenue, aggregated_at
		FROM analytics_revenue_stats
		WHERE stat_date >= $1 AND stat_date <= $2
		ORDER BY stat_date ASC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRevenueStats(rows)
}

// GetDAUStatsByDateRange returns DAU stats for the date range.
func (r *AnalyticsRepository) GetDAUStatsByDateRange(ctx context.Context, from, to time.Time) ([]*analytics.DAUStat, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, stat_date, active_users, aggregated_at
		FROM analytics_dau_stats
		WHERE stat_date >= $1 AND stat_date <= $2
		ORDER BY stat_date ASC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDAUStats(rows)
}

// CountTotalStudents returns the total number of active students.
func (r *AnalyticsRepository) CountTotalStudents(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM users
		WHERE role = 'student' AND deleted_at IS NULL`).Scan(&count)
	return count, err
}

// CountActiveStudents returns students with lesson progress updates since `since`.
func (r *AnalyticsRepository) CountActiveStudents(ctx context.Context, since time.Time) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT e.student_id)
		FROM lesson_progress lp
		JOIN enrollments e ON e.id = lp.enrollment_id
		WHERE lp.last_watched_at >= $1`, since).Scan(&count)
	return count, err
}

// CountTotalEnrollments returns total, free, and paid enrollment counts.
func (r *AnalyticsRepository) CountTotalEnrollments(ctx context.Context) (total, free, paid int, err error) {
	err = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE enrollment_type = 'free'),
			COUNT(*) FILTER (WHERE enrollment_type = 'paid')
		FROM enrollments
		WHERE status = 'active'`).Scan(&total, &free, &paid)
	return
}

// CountEnrollmentsByCourse returns total, free, and paid enrollment counts for a course.
func (r *AnalyticsRepository) CountEnrollmentsByCourse(ctx context.Context, courseID uuid.UUID) (total, free, paid int, err error) {
	err = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE enrollment_type = 'free'),
			COUNT(*) FILTER (WHERE enrollment_type = 'paid')
		FROM enrollments
		WHERE course_id = $1 AND status = 'active'`, courseID).Scan(&total, &free, &paid)
	return
}

// SumRevenueByDateRange returns total, course, and book revenue for a date range.
func (r *AnalyticsRepository) SumRevenueByDateRange(ctx context.Context, from, to time.Time) (total, courseRev, bookRev float64, err error) {
	err = r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(p.amount), 0),
			COALESCE(SUM(p.amount) FILTER (WHERE pi.item_type = 'course'), 0),
			COALESCE(SUM(p.amount) FILTER (WHERE pi.item_type = 'book'), 0)
		FROM payments p
		JOIN payment_intents pi ON pi.id = p.payment_intent_id
		WHERE p.status = 'success' AND p.paid_at >= $1 AND p.paid_at < $2`, from, to).Scan(&total, &courseRev, &bookRev)
	return
}

// CountDAU returns distinct users with lesson progress updates on a given UTC day.
func (r *AnalyticsRepository) CountDAU(ctx context.Context, day time.Time) (int, error) {
	nextDay := day.AddDate(0, 0, 1)
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT e.student_id)
		FROM lesson_progress lp
		JOIN enrollments e ON e.id = lp.enrollment_id
		WHERE lp.last_watched_at >= $1 AND lp.last_watched_at < $2`, day, nextDay).Scan(&count)
	return count, err
}

// GetModuleProgressForCourse returns per-module progress stats for a course on a given day.
func (r *AnalyticsRepository) GetModuleProgressForCourse(ctx context.Context, courseID uuid.UUID, day time.Time) ([]*analytics.ProgressStat, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			m.id AS module_id,
			COUNT(DISTINCT e.student_id) AS total_students,
			COUNT(DISTINCT e.student_id) FILTER (
				WHERE NOT EXISTS (
					SELECT 1 FROM lessons l2
					JOIN chapters c2 ON c2.id = l2.chapter_id
					WHERE c2.module_id = m.id
					  AND l2.deleted_at IS NULL
					  AND NOT EXISTS (
						SELECT 1 FROM lesson_progress lp2
						WHERE lp2.enrollment_id = e.id AND lp2.lesson_id = l2.id AND lp2.completed = true
					  )
				)
			) AS completed_students,
			COUNT(DISTINCT e.student_id) FILTER (
				WHERE EXISTS (
					SELECT 1 FROM lessons l3
					JOIN chapters c3 ON c3.id = l3.chapter_id
					WHERE c3.module_id = m.id
					  AND l3.deleted_at IS NULL
					  AND EXISTS (
						SELECT 1 FROM lesson_progress lp3
						WHERE lp3.enrollment_id = e.id AND lp3.lesson_id = l3.id
					  )
				)
			) AS in_progress_students
		FROM modules m
		JOIN enrollments e ON e.course_id = $1 AND e.status = 'active'
		WHERE m.course_id = $1 AND m.deleted_at IS NULL
		GROUP BY m.id`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*analytics.ProgressStat
	for rows.Next() {
		s := &analytics.ProgressStat{
			ID:           uuid.New(),
			CourseID:     courseID,
			StatDate:     day,
			AggregatedAt: time.Now().UTC(),
		}
		if err := rows.Scan(&s.ModuleID, &s.TotalStudents, &s.CompletedStudents, &s.InProgressStudents); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// ListCourseIDs returns all non-deleted course IDs.
func (r *AnalyticsRepository) ListCourseIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id FROM courses WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// --- scan helpers ---

func scanEnrollmentStats(rows *sql.Rows) ([]*analytics.EnrollmentStat, error) {
	var stats []*analytics.EnrollmentStat
	for rows.Next() {
		s := &analytics.EnrollmentStat{}
		if err := rows.Scan(&s.ID, &s.CourseID, &s.StatDate, &s.TotalEnrolled, &s.FreeEnrolled, &s.PaidEnrolled, &s.AggregatedAt); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func scanProgressStats(rows *sql.Rows) ([]*analytics.ProgressStat, error) {
	var stats []*analytics.ProgressStat
	for rows.Next() {
		s := &analytics.ProgressStat{}
		if err := rows.Scan(&s.ID, &s.CourseID, &s.ModuleID, &s.StatDate, &s.TotalStudents, &s.CompletedStudents, &s.InProgressStudents, &s.AggregatedAt); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func scanRevenueStats(rows *sql.Rows) ([]*analytics.RevenueStat, error) {
	var stats []*analytics.RevenueStat
	for rows.Next() {
		s := &analytics.RevenueStat{}
		if err := rows.Scan(&s.ID, &s.StatDate, &s.TotalRevenue, &s.CourseRevenue, &s.BookRevenue, &s.AggregatedAt); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func scanDAUStats(rows *sql.Rows) ([]*analytics.DAUStat, error) {
	var stats []*analytics.DAUStat
	for rows.Next() {
		s := &analytics.DAUStat{}
		if err := rows.Scan(&s.ID, &s.StatDate, &s.ActiveUsers, &s.AggregatedAt); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}
