package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	appanalytics "lms-backend/internal/application/analytics"

	"github.com/google/uuid"
)

// AnalyticsLiveRepository implements application/analytics.LiveQueryRepo.
// These queries run against live tables (not pre-aggregated) and are used
// for data that cannot be pre-aggregated efficiently.
type AnalyticsLiveRepository struct {
	db *sql.DB
}

// NewAnalyticsLiveRepository creates a new AnalyticsLiveRepository.
func NewAnalyticsLiveRepository(db *sql.DB) *AnalyticsLiveRepository {
	return &AnalyticsLiveRepository{db: db}
}

// CountCourses returns total, free, and paid published course counts.
func (r *AnalyticsLiveRepository) CountCourses(ctx context.Context) (all, free, paid int, err error) {
	err = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE price_type = 'free'),
			COUNT(*) FILTER (WHERE price_type = 'paid')
		FROM courses
		WHERE status = 'published' AND deleted_at IS NULL`).Scan(&all, &free, &paid)
	return
}

// GetModuleTitles returns a map of module_id → title for a course.
func (r *AnalyticsLiveRepository) GetModuleTitles(ctx context.Context, courseID uuid.UUID) (map[uuid.UUID]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title FROM modules
		WHERE course_id = $1 AND deleted_at IS NULL`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	titles := make(map[uuid.UUID]string)
	for rows.Next() {
		var id uuid.UUID
		var title string
		if err := rows.Scan(&id, &title); err != nil {
			return nil, err
		}
		titles[id] = title
	}
	return titles, rows.Err()
}

// GetQuizStatsForCourse returns average score, pass rate, and attempt count for a course.
func (r *AnalyticsLiveRepository) GetQuizStatsForCourse(ctx context.Context, courseID uuid.UUID) (avgScore, passRate float64, totalAttempts int, err error) {
	err = r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(AVG(qa.score_percent), 0),
			COALESCE(
				COUNT(*) FILTER (WHERE qa.passed = true)::float / NULLIF(COUNT(*), 0) * 100,
				0
			),
			COUNT(*)
		FROM quiz_attempts qa
		JOIN quizzes q ON q.id = qa.quiz_id
		WHERE q.course_id = $1 AND qa.status = 'submitted'`, courseID).Scan(&avgScore, &passRate, &totalAttempts)
	return
}

// GetStudentProgressInCourse returns per-student progress rows for a course.
func (r *AnalyticsLiveRepository) GetStudentProgressInCourse(ctx context.Context, courseID uuid.UUID, page, limit int) ([]appanalytics.StudentProgressEntry, int, error) {
	offset := (page - 1) * limit

	var total int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM enrollments
		WHERE course_id = $1 AND status = 'active'`, courseID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			e.student_id,
			u.full_name,
			u.email,
			e.progress_percent,
			(
				SELECT COUNT(DISTINCT m.id)
				FROM modules m
				WHERE m.course_id = $1 AND m.deleted_at IS NULL
				  AND NOT EXISTS (
					SELECT 1 FROM lessons l
					JOIN chapters c ON c.id = l.chapter_id
					WHERE c.module_id = m.id AND l.deleted_at IS NULL
					  AND NOT EXISTS (
						SELECT 1 FROM lesson_progress lp
						WHERE lp.enrollment_id = e.id AND lp.lesson_id = l.id AND lp.completed = true
					  )
				  )
			) AS modules_completed,
			(
				SELECT COUNT(DISTINCT m2.id)
				FROM modules m2
				WHERE m2.course_id = $1 AND m2.deleted_at IS NULL
				  AND EXISTS (
					SELECT 1 FROM lessons l2
					JOIN chapters c2 ON c2.id = l2.chapter_id
					WHERE c2.module_id = m2.id AND l2.deleted_at IS NULL
					  AND EXISTS (
						SELECT 1 FROM lesson_progress lp2
						WHERE lp2.enrollment_id = e.id AND lp2.lesson_id = l2.id
					  )
				  )
			) AS modules_in_progress,
			e.enrolled_at,
			MAX(lp.last_watched_at) AS last_active_at
		FROM enrollments e
		JOIN users u ON u.id = e.student_id
		LEFT JOIN lesson_progress lp ON lp.enrollment_id = e.id
		WHERE e.course_id = $1 AND e.status = 'active'
		GROUP BY e.student_id, u.full_name, u.email, e.progress_percent, e.enrolled_at, e.id
		ORDER BY e.progress_percent DESC
		LIMIT $2 OFFSET $3`, courseID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []appanalytics.StudentProgressEntry
	for rows.Next() {
		var entry appanalytics.StudentProgressEntry
		var lastActive sql.NullTime
		if err := rows.Scan(
			&entry.StudentID, &entry.StudentName,
			&entry.StudentEmail,
			&entry.OverallProgressPercent,
			&entry.ModulesCompleted, &entry.ModulesInProgress,
			&entry.EnrolledAt,
			&lastActive,
		); err != nil {
			return nil, 0, err
		}
		if lastActive.Valid {
			entry.LastActiveAt = &lastActive.Time
		}
		entries = append(entries, entry)
	}
	return entries, total, rows.Err()
}

// GetStudentPointsHistory30d returns daily point totals for the last 30 days.
func (r *AnalyticsLiveRepository) GetStudentPointsHistory30d(ctx context.Context, studentID uuid.UUID) ([]appanalytics.PointsHistoryEntry, error) {
	since := time.Now().UTC().AddDate(0, 0, -30)
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			DATE(earned_at) AS day,
			COALESCE(SUM(points + bonus_points), 0) AS total
		FROM point_events
		WHERE student_id = $1 AND earned_at >= $2
		GROUP BY day
		ORDER BY day ASC`, studentID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []appanalytics.PointsHistoryEntry
	for rows.Next() {
		var e appanalytics.PointsHistoryEntry
		var day time.Time
		if err := rows.Scan(&day, &e.Points); err != nil {
			return nil, err
		}
		e.Date = day.Format("2006-01-02")
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// GetStudentCourseProgress returns all course progress entries for a student.
func (r *AnalyticsLiveRepository) GetStudentCourseProgress(ctx context.Context, studentID uuid.UUID) ([]appanalytics.CourseProgressEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT e.course_id, c.title, e.progress_percent, e.enrolled_at
		FROM enrollments e
		JOIN courses c ON c.id = e.course_id
		WHERE e.student_id = $1 AND e.status = 'active' AND c.deleted_at IS NULL
		ORDER BY e.enrolled_at DESC`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []appanalytics.CourseProgressEntry
	for rows.Next() {
		var e appanalytics.CourseProgressEntry
		if err := rows.Scan(&e.CourseID, &e.CourseTitle, &e.ProgressPercent, &e.EnrolledAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// CountStudentsWithMoreTotalPoints returns the count of students with more total points than the given student.
func (r *AnalyticsLiveRepository) CountStudentsWithMoreTotalPoints(ctx context.Context, studentID uuid.UUID) (int, error) {
	// Get the student's total points first
	var myPoints int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(points + bonus_points), 0)
		FROM point_events WHERE student_id = $1`, studentID).Scan(&myPoints); err != nil {
		return 0, err
	}

	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM (
			SELECT student_id, SUM(points + bonus_points) AS total
			FROM point_events
			GROUP BY student_id
			HAVING SUM(points + bonus_points) > $1
		) ranked`, myPoints).Scan(&count)
	return count, err
}

// GetTeacherCourseIDs returns all course IDs owned by the teacher.
func (r *AnalyticsLiveRepository) GetTeacherCourseIDs(ctx context.Context, teacherID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id FROM courses
		WHERE teacher_id = $1 AND deleted_at IS NULL`, teacherID)
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

// GetCourseTitles returns a map of course_id → title for the given IDs.
func (r *AnalyticsLiveRepository) GetCourseTitles(ctx context.Context, courseIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	if len(courseIDs) == 0 {
		return map[uuid.UUID]string{}, nil
	}

	// Build parameterised IN clause
	args := make([]interface{}, len(courseIDs))
	placeholders := ""
	for i, id := range courseIDs {
		args[i] = id
		if i > 0 {
			placeholders += ","
		}
		placeholders += fmt.Sprintf("$%d", i+1)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, title FROM courses WHERE id IN (`+placeholders+`) AND deleted_at IS NULL`,
		args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	titles := make(map[uuid.UUID]string)
	for rows.Next() {
		var id uuid.UUID
		var title string
		if err := rows.Scan(&id, &title); err != nil {
			return nil, err
		}
		titles[id] = title
	}
	return titles, rows.Err()
}

// GetCourseRevenue returns total revenue for a course.
func (r *AnalyticsLiveRepository) GetCourseRevenue(ctx context.Context, courseID uuid.UUID) (float64, error) {
	var revenue float64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(p.amount), 0)
		FROM payments p
		JOIN payment_intents pi ON pi.id = p.payment_intent_id
		WHERE pi.item_type = 'course' AND pi.item_id = $1 AND p.status = 'success'`, courseID).Scan(&revenue)
	return revenue, err
}

// GetCourseCompletionRate returns the fraction of enrolled students who completed the course.
func (r *AnalyticsLiveRepository) GetCourseCompletionRate(ctx context.Context, courseID uuid.UUID) (float64, error) {
	var rate float64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(
			COUNT(*) FILTER (WHERE progress_percent = 100)::float / NULLIF(COUNT(*), 0) * 100,
			0
		)
		FROM enrollments
		WHERE course_id = $1 AND status = 'active'`, courseID).Scan(&rate)
	return rate, err
}

// GetAdminStats returns aggregate admin platform stats.
func (r *AnalyticsLiveRepository) GetAdminStats(ctx context.Context) (*appanalytics.AdminStatsResponse, error) {
	resp := &appanalytics.AdminStatsResponse{}

	// Total users
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`).Scan(&resp.TotalUsers)
	if err != nil {
		return nil, err
	}

	// Active users 30d
	thirtyDaysAgo := time.Now().UTC().AddDate(0, 0, -30)
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT e.student_id)
		FROM lesson_progress lp
		JOIN enrollments e ON e.id = lp.enrollment_id
		WHERE lp.last_watched_at >= $1`, thirtyDaysAgo).Scan(&resp.ActiveUsers30d)
	if err != nil {
		return nil, err
	}

	// Total courses
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM courses WHERE deleted_at IS NULL`).Scan(&resp.TotalCourses)
	if err != nil {
		return nil, err
	}

	// Published courses
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM courses WHERE status = 'published' AND deleted_at IS NULL`).Scan(&resp.PublishedCourses)
	if err != nil {
		return nil, err
	}

	// Total enrollments
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM enrollments WHERE status = 'active'`).Scan(&resp.TotalEnrollments)
	if err != nil {
		return nil, err
	}

	// Total revenue
	err = r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(p.amount), 0.0)
		FROM payments p
		WHERE p.status = 'success'`).Scan(&resp.TotalRevenue)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// ListCoursesAnalytics returns course analytics list.
func (r *AnalyticsLiveRepository) ListCoursesAnalytics(ctx context.Context) ([]appanalytics.CourseAnalyticsListEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			c.id AS course_id,
			(SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id AND e.status = 'active') AS enrolled,
			(SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id AND e.status = 'active' AND e.progress_percent = 100) AS completed,
			COALESCE((SELECT AVG(e.progress_percent) / 100.0 FROM enrollments e WHERE e.course_id = c.id AND e.status = 'active'), 0.0) AS avg_progress,
			COALESCE((SELECT SUM(p.amount) FROM payments p JOIN payment_intents pi ON pi.id = p.payment_intent_id WHERE pi.item_type = 'course' AND pi.item_id = c.id AND p.status = 'success'), 0.0) AS revenue,
			COALESCE((SELECT AVG(cr.rating) FROM course_reviews cr WHERE cr.course_id = c.id), 0.0) AS rating
		FROM courses c
		WHERE c.deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []appanalytics.CourseAnalyticsListEntry
	for rows.Next() {
		var e appanalytics.CourseAnalyticsListEntry
		if err := rows.Scan(&e.CourseID, &e.Enrolled, &e.Completed, &e.AvgProgress, &e.Revenue, &e.Rating); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

// ListStudentsAnalytics returns student analytics list.
func (r *AnalyticsLiveRepository) ListStudentsAnalytics(ctx context.Context) ([]appanalytics.StudentAnalyticsListEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			u.id AS student_id,
			(SELECT COUNT(*) FROM enrollments e WHERE e.student_id = u.id AND e.status = 'active') AS enrolled_courses,
			COALESCE((SELECT COUNT(*) FROM lesson_progress lp JOIN enrollments e ON lp.enrollment_id = e.id WHERE e.student_id = u.id AND lp.completed = true) * 0.5, 0.0) AS hours_learned,
			COALESCE((SELECT AVG(qa.score_percent) FROM quiz_attempts qa WHERE qa.student_id = u.id AND qa.status = 'submitted'), 0.0) AS avg_score,
			COALESCE((SELECT COUNT(DISTINCT date(earned_at)) FROM point_events WHERE student_id = u.id), 0) AS streak,
			(SELECT COUNT(*) FROM certificates c WHERE c.student_id = u.id) AS certificates
		FROM users u
		WHERE u.role = 'student' AND u.deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []appanalytics.StudentAnalyticsListEntry
	for rows.Next() {
		var e appanalytics.StudentAnalyticsListEntry
		if err := rows.Scan(&e.StudentID, &e.EnrolledCourses, &e.HoursLearned, &e.AvgScore, &e.Streak, &e.Certificates); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

// GetStudentDashboardStats returns dashboard stats for a student.
func (r *AnalyticsLiveRepository) GetStudentDashboardStats(ctx context.Context, studentID uuid.UUID) (*appanalytics.StudentDashboardStats, error) {
	stats := &appanalytics.StudentDashboardStats{}

	// Enrolled courses
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM enrollments
		WHERE student_id = $1 AND status = 'active'`, studentID).Scan(&stats.EnrolledCourses)
	if err != nil {
		return nil, err
	}

	// Completed courses
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM enrollments
		WHERE student_id = $1 AND status = 'active' AND progress_percent = 100`, studentID).Scan(&stats.CompletedCourses)
	if err != nil {
		return nil, err
	}

	// Hours learned
	err = r.db.QueryRowContext(ctx, `
		SELECT COALESCE(COUNT(*) * 0.5, 0.0)
		FROM lesson_progress lp
		JOIN enrollments e ON e.id = lp.enrollment_id
		WHERE e.student_id = $1 AND lp.completed = true`, studentID).Scan(&stats.HoursLearned)
	if err != nil {
		return nil, err
	}

	// Total points
	err = r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(points + bonus_points), 0)
		FROM point_events WHERE student_id = $1`, studentID).Scan(&stats.Points)
	if err != nil {
		return nil, err
	}

	// Streak days
	err = r.db.QueryRowContext(ctx, `
		SELECT COALESCE(COUNT(DISTINCT date(earned_at)), 0)
		FROM point_events WHERE student_id = $1`, studentID).Scan(&stats.StreakDays)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// GetStudentDashboardEnrollments returns active dashboard enrollments for a student.
func (r *AnalyticsLiveRepository) GetStudentDashboardEnrollments(ctx context.Context, studentID uuid.UUID) ([]appanalytics.StudentDashboardEnrollment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			e.id, e.enrolled_at, e.progress_percent, e.completed_at,
			c.id AS course_id, c.title AS course_title, c.thumbnail_url, c.subject
		FROM enrollments e
		JOIN courses c ON c.id = e.course_id
		WHERE e.student_id = $1 AND e.status = 'active' AND c.deleted_at IS NULL
		ORDER BY e.enrolled_at DESC
		LIMIT 3`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var enrollments []appanalytics.StudentDashboardEnrollment
	for rows.Next() {
		var e appanalytics.StudentDashboardEnrollment
		var thumb sql.NullString
		var subject sql.NullString
		if err := rows.Scan(
			&e.ID, &e.EnrolledAt, &e.ProgressPercent, &e.CompletedAt,
			&e.Course.ID, &e.Course.Title, &thumb, &subject,
		); err != nil {
			return nil, err
		}
		if thumb.Valid {
			e.Course.CoverURL = thumb.String
		}
		if subject.Valid {
			e.Course.Category = &appanalytics.StudentDashboardCategory{Name: subject.String}
		}
		enrollments = append(enrollments, e)
	}
	return enrollments, rows.Err()
}

// GetStudentDashboardUpcomingSessions returns upcoming live sessions for a student.
func (r *AnalyticsLiveRepository) GetStudentDashboardUpcomingSessions(ctx context.Context, studentID uuid.UUID) ([]appanalytics.UpcomingLiveSession, error) {
	// Query live sessions for courses the student is enrolled in
	rows, err := r.db.QueryContext(ctx, `
		SELECT ls.id, ls.title, ls.scheduled_at, c.title AS course_title
		FROM live_sessions ls
		JOIN courses c ON c.id = ls.course_id
		JOIN enrollments e ON e.course_id = c.id
		WHERE e.student_id = $1 AND e.status = 'active' AND ls.scheduled_at >= NOW()
		ORDER BY ls.scheduled_at ASC
		LIMIT 5`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []appanalytics.UpcomingLiveSession
	for rows.Next() {
		var s appanalytics.UpcomingLiveSession
		if err := rows.Scan(&s.ID, &s.Title, &s.StartsAt, &s.CourseTitle); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}
