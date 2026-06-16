package analytics

import (
	"time"

	"github.com/google/uuid"
)

// GetAdminOverviewCommand is the input for the admin overview analytics endpoint.
type GetAdminOverviewCommand struct {
	From time.Time
	To   time.Time
}

// GetCourseAnalyticsCommand is the input for per-course analytics.
type GetCourseAnalyticsCommand struct {
	CourseID uuid.UUID
	From     time.Time
	To       time.Time
}

// GetCourseStudentsCommand is the input for per-student progress within a course.
type GetCourseStudentsCommand struct {
	CourseID uuid.UUID
	Page     int
	Limit    int
}

// GetTeacherCourseStudentsCommand scopes a course roster to the owning teacher.
type GetTeacherCourseStudentsCommand struct {
	TeacherID uuid.UUID
	CourseID  uuid.UUID
	Page      int
	Limit     int
}

// GetStudentAnalyticsCommand is the input for a specific student's analytics.
type GetStudentAnalyticsCommand struct {
	StudentID uuid.UUID
}

// GetTeacherAnalyticsCommand is the input for teacher-scoped analytics.
// TeacherID is used to scope results exclusively to the teacher's own courses.
type GetTeacherAnalyticsCommand struct {
	TeacherID uuid.UUID
	From      time.Time
	To        time.Time
}

// GetTeacherStudentAnalyticsCommand scopes a student analytics drill-down to a teacher's courses.
type GetTeacherStudentAnalyticsCommand struct {
	TeacherID uuid.UUID
	StudentID uuid.UUID
}
