package analytics

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines all persistence operations for the analytics context.
type Repository interface {
	// UpsertEnrollmentStat inserts or updates the enrollment stat for a course on a given date.
	UpsertEnrollmentStat(ctx context.Context, stat *EnrollmentStat) error
	// UpsertProgressStat inserts or updates the progress stat for a module on a given date.
	UpsertProgressStat(ctx context.Context, stat *ProgressStat) error
	// UpsertRevenueStat inserts or updates the revenue stat for a given date.
	UpsertRevenueStat(ctx context.Context, stat *RevenueStat) error
	// UpsertDAUStat inserts or updates the DAU stat for a given date.
	UpsertDAUStat(ctx context.Context, stat *DAUStat) error

	// GetEnrollmentStatsByDateRange returns enrollment stats for all courses in the date range.
	GetEnrollmentStatsByDateRange(ctx context.Context, from, to time.Time) ([]*EnrollmentStat, error)
	// GetEnrollmentStatsByCourse returns enrollment stats for a specific course in the date range.
	GetEnrollmentStatsByCourse(ctx context.Context, courseID uuid.UUID, from, to time.Time) ([]*EnrollmentStat, error)
	// GetProgressStatsByCourse returns progress stats for all modules of a course in the date range.
	GetProgressStatsByCourse(ctx context.Context, courseID uuid.UUID, from, to time.Time) ([]*ProgressStat, error)
	// GetRevenueStatsByDateRange returns revenue stats for the date range.
	GetRevenueStatsByDateRange(ctx context.Context, from, to time.Time) ([]*RevenueStat, error)
	// GetDAUStatsByDateRange returns DAU stats for the date range.
	GetDAUStatsByDateRange(ctx context.Context, from, to time.Time) ([]*DAUStat, error)

	// --- Live aggregation queries (used by the worker) ---

	// CountTotalStudents returns the total number of students in the system.
	CountTotalStudents(ctx context.Context) (int, error)
	// CountActiveStudents returns students who had at least one lesson progress update since `since`.
	CountActiveStudents(ctx context.Context, since time.Time) (int, error)
	// CountTotalEnrollments returns total, free, and paid enrollment counts.
	CountTotalEnrollments(ctx context.Context) (total, free, paid int, err error)
	// CountEnrollmentsByCourse returns total, free, and paid enrollment counts for a course.
	CountEnrollmentsByCourse(ctx context.Context, courseID uuid.UUID) (total, free, paid int, err error)
	// SumRevenueByDateRange returns total, course, and book revenue for a date range.
	SumRevenueByDateRange(ctx context.Context, from, to time.Time) (total, courseRev, bookRev float64, err error)
	// CountDAU returns distinct users with lesson progress updates on a given UTC day.
	CountDAU(ctx context.Context, day time.Time) (int, error)
	// GetModuleProgressForCourse returns per-module progress stats for a course on a given day.
	GetModuleProgressForCourse(ctx context.Context, courseID uuid.UUID, day time.Time) ([]*ProgressStat, error)
	// ListCourseIDs returns all non-deleted course IDs (for the worker to iterate).
	ListCourseIDs(ctx context.Context) ([]uuid.UUID, error)
}
