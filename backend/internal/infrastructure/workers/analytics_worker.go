package workers

import (
	"context"
	"time"

	"lms-backend/internal/domain/analytics"
	"lms-backend/pkg/logger"

	"github.com/google/uuid"
)

// AnalyticsWorker runs a scheduled hourly pre-aggregation job.
// It populates the analytics_* tables to avoid full-table scans on hot read paths.
// Requirement 23.6
type AnalyticsWorker struct {
	repo     analytics.Repository
	interval time.Duration
}

// NewAnalyticsWorker creates a new AnalyticsWorker.
// interval is how often the aggregation runs (typically 1 hour in production).
func NewAnalyticsWorker(repo analytics.Repository, interval time.Duration) *AnalyticsWorker {
	return &AnalyticsWorker{repo: repo, interval: interval}
}

// Run starts the aggregation loop. It blocks until ctx is cancelled.
func (w *AnalyticsWorker) Run(ctx context.Context) {
	logger.Info(ctx, "Analytics aggregation worker started", "interval", w.interval)

	// Run immediately on startup, then on each tick.
	w.aggregate(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info(ctx, "Analytics aggregation worker shutting down")
			return
		case <-ticker.C:
			w.aggregate(ctx)
		}
	}
}

// aggregate runs one full aggregation pass for yesterday and today (UTC).
func (w *AnalyticsWorker) aggregate(ctx context.Context) {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)

	for _, day := range []time.Time{yesterday, today} {
		if err := w.aggregateDay(ctx, day); err != nil {
			logger.Error(ctx, "Analytics aggregation failed", "day", day.Format("2006-01-02"), "error", err)
		}
	}
}

// aggregateDay aggregates all metrics for a single UTC calendar day.
func (w *AnalyticsWorker) aggregateDay(ctx context.Context, day time.Time) error {
	now := time.Now().UTC()

	// 1. DAU
	dau, err := w.repo.CountDAU(ctx, day)
	if err != nil {
		return err
	}
	if err := w.repo.UpsertDAUStat(ctx, &analytics.DAUStat{
		ID:           uuid.New(),
		StatDate:     day,
		ActiveUsers:  dau,
		AggregatedAt: now,
	}); err != nil {
		return err
	}

	// 2. Revenue for the day
	dayEnd := day.AddDate(0, 0, 1)
	totalRev, courseRev, bookRev, err := w.repo.SumRevenueByDateRange(ctx, day, dayEnd)
	if err != nil {
		return err
	}
	if err := w.repo.UpsertRevenueStat(ctx, &analytics.RevenueStat{
		ID:            uuid.New(),
		StatDate:      day,
		TotalRevenue:  totalRev,
		CourseRevenue: courseRev,
		BookRevenue:   bookRev,
		AggregatedAt:  now,
	}); err != nil {
		return err
	}

	// 3. Per-course enrollment and progress stats
	courseIDs, err := w.repo.ListCourseIDs(ctx)
	if err != nil {
		return err
	}

	for _, courseID := range courseIDs {
		total, free, paid, err := w.repo.CountEnrollmentsByCourse(ctx, courseID)
		if err != nil {
			logger.Error(ctx, "Failed to count enrollments for course", "course_id", courseID, "error", err)
			continue
		}
		if err := w.repo.UpsertEnrollmentStat(ctx, &analytics.EnrollmentStat{
			ID:            uuid.New(),
			CourseID:      courseID,
			StatDate:      day,
			TotalEnrolled: total,
			FreeEnrolled:  free,
			PaidEnrolled:  paid,
			AggregatedAt:  now,
		}); err != nil {
			logger.Error(ctx, "Failed to upsert enrollment stat", "course_id", courseID, "error", err)
			continue
		}

		// Per-module progress stats
		moduleStats, err := w.repo.GetModuleProgressForCourse(ctx, courseID, day)
		if err != nil {
			logger.Error(ctx, "Failed to get module progress", "course_id", courseID, "error", err)
			continue
		}
		for _, ms := range moduleStats {
			if err := w.repo.UpsertProgressStat(ctx, ms); err != nil {
				logger.Error(ctx, "Failed to upsert progress stat", "course_id", courseID, "module_id", ms.ModuleID, "error", err)
			}
		}
	}

	logger.Info(ctx, "Analytics aggregation complete", "day", day.Format("2006-01-02"), "courses", len(courseIDs), "dau", dau)
	return nil
}
