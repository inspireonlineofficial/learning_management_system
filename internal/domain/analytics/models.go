package analytics

import (
	"time"

	"github.com/google/uuid"
)

// EnrollmentStat is a pre-aggregated daily enrollment snapshot per course.
type EnrollmentStat struct {
	ID            uuid.UUID
	CourseID      uuid.UUID
	StatDate      time.Time
	TotalEnrolled int
	FreeEnrolled  int
	PaidEnrolled  int
	AggregatedAt  time.Time
}

// ProgressStat is a pre-aggregated daily completion snapshot per module per course.
type ProgressStat struct {
	ID                 uuid.UUID
	CourseID           uuid.UUID
	ModuleID           uuid.UUID
	StatDate           time.Time
	TotalStudents      int
	CompletedStudents  int
	InProgressStudents int
	AggregatedAt       time.Time
}

// RevenueStat is a pre-aggregated daily revenue snapshot.
type RevenueStat struct {
	ID            uuid.UUID
	StatDate      time.Time
	TotalRevenue  float64
	CourseRevenue float64
	BookRevenue   float64
	AggregatedAt  time.Time
}

// DAUStat is a pre-aggregated daily active users count.
type DAUStat struct {
	ID           uuid.UUID
	StatDate     time.Time
	ActiveUsers  int
	AggregatedAt time.Time
}
