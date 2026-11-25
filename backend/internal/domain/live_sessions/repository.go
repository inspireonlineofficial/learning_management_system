package live_sessions

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// LiveSessionRepository defines the persistence interface for live sessions.
// Requirements: 16.1
type LiveSessionRepository interface {
	Create(ctx context.Context, session *LiveSession) error
	FindByID(ctx context.Context, id uuid.UUID) (*LiveSession, error)
	FindByCourseID(ctx context.Context, courseID uuid.UUID) ([]*LiveSession, error)
	FindByTeacherID(ctx context.Context, teacherID uuid.UUID) ([]*LiveSession, error)
	FindByCourseIDs(ctx context.Context, courseIDs []uuid.UUID) ([]*LiveSession, error)
	FindByCourseIDsBetween(ctx context.Context, courseIDs []uuid.UUID, from, to *time.Time) ([]*LiveSession, error)
	Update(ctx context.Context, session *LiveSession) error
}

// AttendanceRepository defines the persistence interface for session attendance.
// Requirements: 16.3, 16.7
type AttendanceRepository interface {
	Create(ctx context.Context, attendance *Attendance) error
	FindBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*Attendance, error)
	FindBySessionAndStudent(ctx context.Context, sessionID, studentID uuid.UUID) (*Attendance, error)
	Update(ctx context.Context, attendance *Attendance) error
}
