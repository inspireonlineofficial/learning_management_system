package live_sessions

import (
	"time"

	"github.com/google/uuid"
)

// LiveSessionResponse is the standard response for a live session.
type LiveSessionResponse struct {
	ID              uuid.UUID `json:"id"`
	CourseID        uuid.UUID `json:"course_id"`
	TeacherID       uuid.UUID `json:"teacher_id"`
	Title           string    `json:"title"`
	ScheduledAt     time.Time `json:"scheduled_at"`
	DurationMinutes int       `json:"duration_minutes"`
	Status          string    `json:"status"`
	RecordSession   bool      `json:"record_session"`
	AttendeeCount   int       `json:"attendee_count"`
	IsToday         bool      `json:"is_today"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// StartSessionResponse includes the teacher-scoped room token.
// Requirements: 16.2
type StartSessionResponse struct {
	Session   LiveSessionResponse `json:"session"`
	RoomToken string              `json:"room_token"`
}

// JoinSessionResponse includes the student-scoped room token.
// Requirements: 16.3
type JoinSessionResponse struct {
	SessionID uuid.UUID `json:"session_id"`
	RoomToken string    `json:"room_token"`
}

// AttendanceRecord is a single student's attendance entry.
// Requirements: 16.7
type AttendanceRecord struct {
	StudentID       uuid.UUID  `json:"student_id"`
	JoinedAt        time.Time  `json:"joined_at"`
	LeftAt          *time.Time `json:"left_at,omitempty"`
	DurationMinutes int        `json:"duration_minutes"`
}

// AttendanceResponse is the response for GetAttendance.
type AttendanceResponse struct {
	SessionID uuid.UUID          `json:"session_id"`
	Records   []AttendanceRecord `json:"records"`
}

// LiveSessionListResponse wraps a list of sessions.
type LiveSessionListResponse struct {
	Sessions []LiveSessionResponse `json:"sessions"`
}
