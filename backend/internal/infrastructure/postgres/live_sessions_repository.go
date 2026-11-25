package postgres

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	domain "lms-backend/internal/domain/live_sessions"

	"github.com/google/uuid"
)

// LiveSessionRepository implements domain/live_sessions.LiveSessionRepository.
type LiveSessionRepository struct {
	db *sql.DB
}

// NewLiveSessionRepository creates a new LiveSessionRepository.
func NewLiveSessionRepository(db *sql.DB) *LiveSessionRepository {
	return &LiveSessionRepository{db: db}
}

func (r *LiveSessionRepository) Create(ctx context.Context, s *domain.LiveSession) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO live_sessions
			(id, course_id, teacher_id, title, scheduled_at, duration_minutes, status, record_session, attendee_count, recording_rustfs_key, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		s.ID, s.CourseID, s.TeacherID, s.Title, s.ScheduledAt, s.DurationMinutes,
		string(s.Status), s.RecordSession, s.AttendeeCount, s.RecordingRustfsKey,
		s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (r *LiveSessionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.LiveSession, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, course_id, teacher_id, title, scheduled_at, duration_minutes, status,
		       record_session, attendee_count, recording_rustfs_key, created_at, updated_at
		FROM live_sessions WHERE id = $1`, id)
	return scanLiveSession(row)
}

func (r *LiveSessionRepository) FindByCourseID(ctx context.Context, courseID uuid.UUID) ([]*domain.LiveSession, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, course_id, teacher_id, title, scheduled_at, duration_minutes, status,
		       record_session, attendee_count, recording_rustfs_key, created_at, updated_at
		FROM live_sessions WHERE course_id = $1
		ORDER BY scheduled_at ASC`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*domain.LiveSession
	for rows.Next() {
		s, err := scanLiveSessionRow(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *LiveSessionRepository) FindByTeacherID(ctx context.Context, teacherID uuid.UUID) ([]*domain.LiveSession, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, course_id, teacher_id, title, scheduled_at, duration_minutes, status,
		       record_session, attendee_count, recording_rustfs_key, created_at, updated_at
		FROM live_sessions
		WHERE teacher_id = $1
		ORDER BY scheduled_at DESC`, teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*domain.LiveSession
	for rows.Next() {
		s, err := scanLiveSessionRow(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *LiveSessionRepository) FindByCourseIDs(ctx context.Context, courseIDs []uuid.UUID) ([]*domain.LiveSession, error) {
	if len(courseIDs) == 0 {
		return []*domain.LiveSession{}, nil
	}

	placeholders := make([]string, len(courseIDs))
	args := make([]interface{}, len(courseIDs))
	for index, courseID := range courseIDs {
		placeholders[index] = "$" + strconv.Itoa(index+1)
		args[index] = courseID
	}

	query := `
		SELECT id, course_id, teacher_id, title, scheduled_at, duration_minutes, status,
		       record_session, attendee_count, recording_rustfs_key, created_at, updated_at
		FROM live_sessions
		WHERE course_id IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY scheduled_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*domain.LiveSession
	for rows.Next() {
		s, err := scanLiveSessionRow(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *LiveSessionRepository) FindByCourseIDsBetween(ctx context.Context, courseIDs []uuid.UUID, from, to *time.Time) ([]*domain.LiveSession, error) {
	if len(courseIDs) == 0 {
		return []*domain.LiveSession{}, nil
	}
	placeholders := make([]string, len(courseIDs))
	args := make([]interface{}, len(courseIDs))
	for index, courseID := range courseIDs {
		placeholders[index] = "$" + strconv.Itoa(index+1)
		args[index] = courseID
	}
	conditions := []string{"course_id IN (" + strings.Join(placeholders, ",") + ")"}
	next := len(args) + 1
	if from != nil {
		conditions = append(conditions, "scheduled_at >= $"+strconv.Itoa(next))
		args = append(args, *from)
		next++
	}
	if to != nil {
		conditions = append(conditions, "scheduled_at <= $"+strconv.Itoa(next))
		args = append(args, *to)
	}
	query := `
		SELECT id, course_id, teacher_id, title, scheduled_at, duration_minutes, status,
		       record_session, attendee_count, recording_rustfs_key, created_at, updated_at
		FROM live_sessions
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY scheduled_at DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []*domain.LiveSession
	for rows.Next() {
		s, err := scanLiveSessionRow(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *LiveSessionRepository) Update(ctx context.Context, s *domain.LiveSession) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE live_sessions SET
			title = $2, scheduled_at = $3, duration_minutes = $4, status = $5,
			record_session = $6, attendee_count = $7, recording_rustfs_key = $8, updated_at = $9
		WHERE id = $1`,
		s.ID, s.Title, s.ScheduledAt, s.DurationMinutes, string(s.Status),
		s.RecordSession, s.AttendeeCount, s.RecordingRustfsKey, s.UpdatedAt,
	)
	return err
}

// AttendanceRepository implements domain/live_sessions.AttendanceRepository.
type AttendanceRepository struct {
	db *sql.DB
}

// NewAttendanceRepository creates a new AttendanceRepository.
func NewAttendanceRepository(db *sql.DB) *AttendanceRepository {
	return &AttendanceRepository{db: db}
}

func (r *AttendanceRepository) Create(ctx context.Context, a *domain.Attendance) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO attendance (id, session_id, student_id, joined_at, left_at, duration_minutes)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (session_id, student_id) DO NOTHING`,
		a.ID, a.SessionID, a.StudentID, a.JoinedAt, a.LeftAt, a.DurationMinutes,
	)
	return err
}

func (r *AttendanceRepository) FindBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*domain.Attendance, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, session_id, student_id, joined_at, left_at, duration_minutes
		FROM attendance WHERE session_id = $1
		ORDER BY joined_at ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*domain.Attendance
	for rows.Next() {
		a, err := scanAttendanceRow(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, a)
	}
	return records, rows.Err()
}

func (r *AttendanceRepository) FindBySessionAndStudent(ctx context.Context, sessionID, studentID uuid.UUID) (*domain.Attendance, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, session_id, student_id, joined_at, left_at, duration_minutes
		FROM attendance WHERE session_id = $1 AND student_id = $2`, sessionID, studentID)
	return scanAttendance(row)
}

func (r *AttendanceRepository) Update(ctx context.Context, a *domain.Attendance) error {
	durationMinutes := 0
	if a.LeftAt != nil {
		durationMinutes = int(a.LeftAt.Sub(a.JoinedAt) / time.Minute)
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE attendance SET left_at = $2, duration_minutes = $3 WHERE id = $1`,
		a.ID, a.LeftAt, durationMinutes,
	)
	return err
}

// scanLiveSession scans a single *sql.Row into a LiveSession.
func scanLiveSession(row *sql.Row) (*domain.LiveSession, error) {
	s := &domain.LiveSession{}
	var status string
	err := row.Scan(
		&s.ID, &s.CourseID, &s.TeacherID, &s.Title, &s.ScheduledAt, &s.DurationMinutes,
		&status, &s.RecordSession, &s.AttendeeCount, &s.RecordingRustfsKey,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.Status = domain.SessionStatus(status)
	return s, nil
}

// scanLiveSessionRow scans a rows.Next() row into a LiveSession.
func scanLiveSessionRow(rows *sql.Rows) (*domain.LiveSession, error) {
	s := &domain.LiveSession{}
	var status string
	err := rows.Scan(
		&s.ID, &s.CourseID, &s.TeacherID, &s.Title, &s.ScheduledAt, &s.DurationMinutes,
		&status, &s.RecordSession, &s.AttendeeCount, &s.RecordingRustfsKey,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.Status = domain.SessionStatus(status)
	return s, nil
}

// scanAttendance scans a single *sql.Row into an Attendance.
func scanAttendance(row *sql.Row) (*domain.Attendance, error) {
	a := &domain.Attendance{}
	err := row.Scan(&a.ID, &a.SessionID, &a.StudentID, &a.JoinedAt, &a.LeftAt, &a.DurationMinutes)
	if err != nil {
		return nil, err
	}
	return a, nil
}

// scanAttendanceRow scans a rows.Next() row into an Attendance.
func scanAttendanceRow(rows *sql.Rows) (*domain.Attendance, error) {
	a := &domain.Attendance{}
	err := rows.Scan(&a.ID, &a.SessionID, &a.StudentID, &a.JoinedAt, &a.LeftAt, &a.DurationMinutes)
	if err != nil {
		return nil, err
	}
	return a, nil
}
