package live_sessions

import (
	"time"

	"github.com/google/uuid"
)

// ScheduleSessionCommand creates a new live session.
// Requirements: 16.1
type ScheduleSessionCommand struct {
	TeacherID       uuid.UUID
	CourseID        uuid.UUID
	Title           string
	ScheduledAt     time.Time
	DurationMinutes int
	RecordSession   bool
}

// StartSessionCommand transitions a session to live status.
// Requirements: 16.2
type StartSessionCommand struct {
	SessionID uuid.UUID
	TeacherID uuid.UUID
}

// JoinSessionCommand records student attendance and returns a room token.
// Requirements: 16.3, 16.4
type JoinSessionCommand struct {
	SessionID uuid.UUID
	StudentID uuid.UUID
}

// EndSessionCommand transitions a session to ended status.
// Requirements: 16.5
type EndSessionCommand struct {
	SessionID uuid.UUID
	TeacherID uuid.UUID
}

// RescheduleOrCancelSessionCommand updates a session's schedule or cancels it.
// Requirements: 16.6
type RescheduleOrCancelSessionCommand struct {
	SessionID       uuid.UUID
	TeacherID       uuid.UUID
	ScheduledAt     *time.Time
	DurationMinutes *int
	Title           *string
	Cancel          bool
}

// GetAttendanceCommand retrieves attendance records for a session.
// Requirements: 16.7
type GetAttendanceCommand struct {
	SessionID uuid.UUID
	TeacherID uuid.UUID
}

// ListTeacherSessionsCommand retrieves all sessions owned by a teacher.
type ListTeacherSessionsCommand struct {
	TeacherID uuid.UUID
}

// GetTeacherSessionCommand retrieves a single teacher-owned session.
type GetTeacherSessionCommand struct {
	SessionID uuid.UUID
	TeacherID uuid.UUID
}

// ListStudentSessionsCommand retrieves sessions across a student's enrolled courses.
type ListStudentSessionsCommand struct {
	StudentID uuid.UUID
	From      *time.Time
	To        *time.Time
}

// GetStudentSessionCommand retrieves a single session for an enrolled student.
type GetStudentSessionCommand struct {
	SessionID uuid.UUID
	StudentID uuid.UUID
}
