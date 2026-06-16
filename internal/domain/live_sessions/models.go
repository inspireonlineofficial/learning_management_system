package live_sessions

import (
	"time"

	"github.com/google/uuid"
)

// SessionStatus represents the lifecycle state of a live session.
type SessionStatus string

const (
	SessionStatusScheduled SessionStatus = "scheduled"
	SessionStatusLive      SessionStatus = "live"
	SessionStatusEnded     SessionStatus = "ended"
	SessionStatusCancelled SessionStatus = "cancelled"
)

// LiveSession is the aggregate root for the live sessions bounded context.
// Requirements: 16.1
type LiveSession struct {
	ID                 uuid.UUID     `json:"id"`
	CourseID           uuid.UUID     `json:"course_id"`
	TeacherID          uuid.UUID     `json:"teacher_id"`
	Title              string        `json:"title"`
	ScheduledAt        time.Time     `json:"scheduled_at"`
	DurationMinutes    int           `json:"duration_minutes"`
	Status             SessionStatus `json:"status"`
	RecordSession      bool          `json:"record_session"`
	AttendeeCount      int           `json:"attendee_count"`
	RecordingRustfsKey *string       `json:"-"` // never exposed in API responses
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

// Attendance records a student's participation in a live session.
// Requirements: 16.3, 16.7
type Attendance struct {
	ID              uuid.UUID  `json:"id"`
	SessionID       uuid.UUID  `json:"session_id"`
	StudentID       uuid.UUID  `json:"student_id"`
	JoinedAt        time.Time  `json:"joined_at"`
	LeftAt          *time.Time `json:"left_at,omitempty"`
	DurationMinutes int        `json:"duration_minutes"`
}
