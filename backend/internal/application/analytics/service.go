package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"lms-backend/internal/domain/analytics"
	"lms-backend/pkg/apperrors"
	"lms-backend/pkg/logger"

	"github.com/google/uuid"
)

// Service defines the analytics use cases.
type Service interface {
	GetAdminOverview(ctx context.Context, cmd GetAdminOverviewCommand) (*AdminOverviewResponse, error)
	GetCourseAnalytics(ctx context.Context, cmd GetCourseAnalyticsCommand) (*CourseAnalyticsResponse, error)
	GetCourseStudents(ctx context.Context, cmd GetCourseStudentsCommand) (*CourseStudentsResponse, error)
	GetTeacherCourseStudents(ctx context.Context, cmd GetTeacherCourseStudentsCommand) (*CourseStudentsResponse, error)
	GetStudentAnalytics(ctx context.Context, cmd GetStudentAnalyticsCommand) (*StudentAnalyticsResponse, error)
	GetTeacherAnalytics(ctx context.Context, cmd GetTeacherAnalyticsCommand) (*TeacherAnalyticsResponse, error)
	GetTeacherStudentAnalytics(ctx context.Context, cmd GetTeacherStudentAnalyticsCommand) (*StudentAnalyticsResponse, error)

	GetAdminStats(ctx context.Context) (*AdminStatsResponse, error)
	ListCoursesAnalytics(ctx context.Context) (*CourseAnalyticsListResponse, error)
	ListStudentsAnalytics(ctx context.Context) (*StudentAnalyticsListResponse, error)
	GetStudentDashboard(ctx context.Context, studentID uuid.UUID) (*StudentDashboardResponse, error)

	// InvalidateStudentDashboard drops the cached dashboard response for the
	// given student. Callers should invoke this whenever a backend event
	// invalidates the snapshot — typically when a course the student is
	// enrolled in is soft-deleted (admin/teacher delete), so the next
	// /v1/student/dashboard read returns an up-to-date stats object instead
	// of a 5-minute-old cached copy that still references the deleted course.
	InvalidateStudentDashboard(ctx context.Context, studentID uuid.UUID) error
}

// Cache defines the Redis caching interface for analytics responses.
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// LiveQueryRepo provides live (non-pre-aggregated) queries needed for analytics.
type LiveQueryRepo interface {
	CountCourses(ctx context.Context) (all, free, paid int, err error)
	GetModuleTitles(ctx context.Context, courseID uuid.UUID) (map[uuid.UUID]string, error)
	GetQuizStatsForCourse(ctx context.Context, courseID uuid.UUID) (avgScore, passRate float64, totalAttempts int, err error)
	GetStudentProgressInCourse(ctx context.Context, courseID uuid.UUID, page, limit int) ([]StudentProgressEntry, int, error)
	GetStudentPointsHistory30d(ctx context.Context, studentID uuid.UUID) ([]PointsHistoryEntry, error)
	GetStudentCourseProgress(ctx context.Context, studentID uuid.UUID) ([]CourseProgressEntry, error)
	CountStudentsWithMoreTotalPoints(ctx context.Context, studentID uuid.UUID) (int, error)
	GetTeacherCourseIDs(ctx context.Context, teacherID uuid.UUID) ([]uuid.UUID, error)
	GetCourseTitles(ctx context.Context, courseIDs []uuid.UUID) (map[uuid.UUID]string, error)
	GetCourseRevenue(ctx context.Context, courseID uuid.UUID) (float64, error)
	GetCourseCompletionRate(ctx context.Context, courseID uuid.UUID) (float64, error)

	GetAdminStats(ctx context.Context) (*AdminStatsResponse, error)
	ListCoursesAnalytics(ctx context.Context) ([]CourseAnalyticsListEntry, error)
	ListStudentsAnalytics(ctx context.Context) ([]StudentAnalyticsListEntry, error)
	GetStudentDashboardStats(ctx context.Context, studentID uuid.UUID) (*StudentDashboardStats, error)
	GetStudentDashboardEnrollments(ctx context.Context, studentID uuid.UUID) ([]StudentDashboardEnrollment, error)
	GetStudentDashboardUpcomingSessions(ctx context.Context, studentID uuid.UUID) ([]UpcomingLiveSession, error)
}

const cacheTTL = 5 * time.Minute

type service struct {
	repo     analytics.Repository
	liveRepo LiveQueryRepo
	cache    Cache
}

// NewService creates a new analytics service.
func NewService(repo analytics.Repository, liveRepo LiveQueryRepo, cache Cache) Service {
	return &service{repo: repo, liveRepo: liveRepo, cache: cache}
}

// GetAdminOverview returns platform-wide analytics. Requirement 23.1
func (s *service) GetAdminOverview(ctx context.Context, cmd GetAdminOverviewCommand) (*AdminOverviewResponse, error) {
	cacheKey := fmt.Sprintf("analytics:overview:%s:%s", cmd.From.Format("2006-01-02"), cmd.To.Format("2006-01-02"))
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var resp AdminOverviewResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return &resp, nil
		}
	}

	now := time.Now().UTC()

	// Course counts
	allCourses, freeCourses, paidCourses, err := s.liveRepo.CountCourses(ctx)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to count courses")
	}

	// Student counts
	totalStudents, err := s.repo.CountTotalStudents(ctx)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to count students")
	}
	activeStudents, err := s.repo.CountActiveStudents(ctx, now.AddDate(0, 0, -30))
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to count active students")
	}

	// Enrollment counts
	totalEnroll, freeEnroll, paidEnroll, err := s.repo.CountTotalEnrollments(ctx)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to count enrollments")
	}

	// Revenue
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	epoch := time.Time{}

	monthRev, _, _, err := s.repo.SumRevenueByDateRange(ctx, monthStart, now)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to sum monthly revenue")
	}
	yearRev, _, _, err := s.repo.SumRevenueByDateRange(ctx, yearStart, now)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to sum yearly revenue")
	}
	allTimeRev, _, _, err := s.repo.SumRevenueByDateRange(ctx, epoch, now)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to sum all-time revenue")
	}

	// DAU for requested date range
	dauStats, err := s.repo.GetDAUStatsByDateRange(ctx, cmd.From, cmd.To)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to get DAU stats")
	}
	dauEntries := make([]DAUEntry, 0, len(dauStats))
	for _, d := range dauStats {
		dauEntries = append(dauEntries, DAUEntry{
			Date:        d.StatDate.Format("2006-01-02"),
			ActiveUsers: d.ActiveUsers,
		})
	}

	resp := &AdminOverviewResponse{
		TotalCourses:      CourseCounts{All: allCourses, Free: freeCourses, Paid: paidCourses},
		TotalStudents:     totalStudents,
		ActiveStudents30d: activeStudents,
		TotalEnrollments:  EnrollmentCounts{All: totalEnroll, Free: freeEnroll, Paid: paidEnroll},
		Revenue:           RevenueSummary{ThisMonth: monthRev, ThisYear: yearRev, AllTime: allTimeRev},
		DailyActiveUsers:  dauEntries,
	}

	s.cacheResponse(ctx, cacheKey, resp)
	return resp, nil
}

// GetCourseAnalytics returns per-module completion rates, quiz stats, and enrollment trends. Requirement 23.2
func (s *service) GetCourseAnalytics(ctx context.Context, cmd GetCourseAnalyticsCommand) (*CourseAnalyticsResponse, error) {
	cacheKey := fmt.Sprintf("analytics:course:%s:%s:%s", cmd.CourseID, cmd.From.Format("2006-01-02"), cmd.To.Format("2006-01-02"))
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var resp CourseAnalyticsResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return &resp, nil
		}
	}

	// Module titles
	moduleTitles, err := s.liveRepo.GetModuleTitles(ctx, cmd.CourseID)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to get module titles")
	}

	// Progress stats from pre-aggregated table
	progressStats, err := s.repo.GetProgressStatsByCourse(ctx, cmd.CourseID, cmd.From, cmd.To)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to get progress stats")
	}

	// Aggregate per-module (use latest stat per module)
	latestByModule := make(map[uuid.UUID]*analytics.ProgressStat)
	for _, ps := range progressStats {
		if existing, ok := latestByModule[ps.ModuleID]; !ok || ps.StatDate.After(existing.StatDate) {
			latestByModule[ps.ModuleID] = ps
		}
	}

	moduleCompletions := make([]ModuleCompletionRate, 0, len(latestByModule))
	var dropOff *ModuleDropOff
	var maxDropOffRate float64

	for moduleID, ps := range latestByModule {
		rate := 0.0
		if ps.TotalStudents > 0 {
			rate = float64(ps.CompletedStudents) / float64(ps.TotalStudents) * 100
		}
		title := moduleTitles[moduleID]
		moduleCompletions = append(moduleCompletions, ModuleCompletionRate{
			ModuleID:          moduleID,
			ModuleTitle:       title,
			TotalStudents:     ps.TotalStudents,
			CompletedStudents: ps.CompletedStudents,
			CompletionRate:    rate,
		})

		dropOffRate := 100 - rate
		if dropOffRate > maxDropOffRate {
			maxDropOffRate = dropOffRate
			dropOff = &ModuleDropOff{
				ModuleID:    moduleID,
				ModuleTitle: title,
				DropOffRate: dropOffRate,
			}
		}
	}

	// Quiz stats
	avgScore, passRate, totalAttempts, err := s.liveRepo.GetQuizStatsForCourse(ctx, cmd.CourseID)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to get quiz stats")
	}

	// Enrollment over time from pre-aggregated table
	enrollStats, err := s.repo.GetEnrollmentStatsByCourse(ctx, cmd.CourseID, cmd.From, cmd.To)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to get enrollment stats")
	}
	enrollEntries := make([]EnrollmentEntry, 0, len(enrollStats))
	for _, es := range enrollStats {
		enrollEntries = append(enrollEntries, EnrollmentEntry{
			Date:          es.StatDate.Format("2006-01-02"),
			TotalEnrolled: es.TotalEnrolled,
		})
	}

	resp := &CourseAnalyticsResponse{
		CourseID:           cmd.CourseID,
		ModuleCompletions:  moduleCompletions,
		DropOffModule:      dropOff,
		QuizStats:          QuizStats{AverageScore: avgScore, PassRate: passRate, TotalAttempts: totalAttempts},
		EnrollmentOverTime: enrollEntries,
	}

	s.cacheResponse(ctx, cacheKey, resp)
	return resp, nil
}

// GetCourseStudents returns per-student progress within a course. Requirement 23.3
func (s *service) GetCourseStudents(ctx context.Context, cmd GetCourseStudentsCommand) (*CourseStudentsResponse, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.Limit < 1 || cmd.Limit > 100 {
		cmd.Limit = 20
	}

	cacheKey := fmt.Sprintf("analytics:course_students:%s:%d:%d", cmd.CourseID, cmd.Page, cmd.Limit)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var resp CourseStudentsResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return &resp, nil
		}
	}

	students, total, err := s.liveRepo.GetStudentProgressInCourse(ctx, cmd.CourseID, cmd.Page, cmd.Limit)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to get student progress")
	}

	totalPages := (total + cmd.Limit - 1) / cmd.Limit
	if totalPages < 1 {
		totalPages = 1
	}

	resp := &CourseStudentsResponse{
		Students: students,
		Meta:     PaginationMeta{Page: cmd.Page, Limit: cmd.Limit, Total: total, TotalPages: totalPages},
	}

	s.cacheResponse(ctx, cacheKey, resp)
	return resp, nil
}

// GetTeacherCourseStudents returns per-student progress for a course owned by the teacher.
func (s *service) GetTeacherCourseStudents(ctx context.Context, cmd GetTeacherCourseStudentsCommand) (*CourseStudentsResponse, error) {
	courseIDs, err := s.liveRepo.GetTeacherCourseIDs(ctx, cmd.TeacherID)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to load teacher courses")
	}

	ownsCourse := false
	for _, courseID := range courseIDs {
		if courseID == cmd.CourseID {
			ownsCourse = true
			break
		}
	}
	if !ownsCourse {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "course does not belong to this teacher")
	}

	return s.GetCourseStudents(ctx, GetCourseStudentsCommand{
		CourseID: cmd.CourseID,
		Page:     cmd.Page,
		Limit:    cmd.Limit,
	})
}

// GetStudentAnalytics returns a student's points history, course progress, and rank. Requirement 23.4
func (s *service) GetStudentAnalytics(ctx context.Context, cmd GetStudentAnalyticsCommand) (*StudentAnalyticsResponse, error) {
	cacheKey := fmt.Sprintf("analytics:student:%s", cmd.StudentID)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var resp StudentAnalyticsResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return &resp, nil
		}
	}

	pointsHistory, err := s.liveRepo.GetStudentPointsHistory30d(ctx, cmd.StudentID)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to get points history")
	}

	courseProgress, err := s.liveRepo.GetStudentCourseProgress(ctx, cmd.StudentID)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to get course progress")
	}

	studentsAhead, err := s.liveRepo.CountStudentsWithMoreTotalPoints(ctx, cmd.StudentID)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to get global rank")
	}

	resp := &StudentAnalyticsResponse{
		StudentID:      cmd.StudentID,
		PointsHistory:  pointsHistory,
		CourseProgress: courseProgress,
		GlobalRank:     studentsAhead + 1,
	}

	s.cacheResponse(ctx, cacheKey, resp)
	return resp, nil
}

// GetTeacherAnalytics returns analytics scoped exclusively to the teacher's own courses. Requirement 23.5
func (s *service) GetTeacherAnalytics(ctx context.Context, cmd GetTeacherAnalyticsCommand) (*TeacherAnalyticsResponse, error) {
	cacheKey := fmt.Sprintf("analytics:teacher:%s:%s:%s", cmd.TeacherID, cmd.From.Format("2006-01-02"), cmd.To.Format("2006-01-02"))
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var resp TeacherAnalyticsResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return &resp, nil
		}
	}

	// Scope exclusively to teacher's own courses — no data from other teachers. Requirement 23.5
	courseIDs, err := s.liveRepo.GetTeacherCourseIDs(ctx, cmd.TeacherID)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to get teacher courses")
	}

	courseTitles, err := s.liveRepo.GetCourseTitles(ctx, courseIDs)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to get course titles")
	}

	courseStats := make([]TeacherCourseStats, 0, len(courseIDs))
	var totalRevenue float64

	for _, courseID := range courseIDs {
		total, _, _, err := s.repo.CountEnrollmentsByCourse(ctx, courseID)
		if err != nil {
			logger.Error(ctx, "Failed to count enrollments for teacher course", "course_id", courseID, "error", err)
			continue
		}

		completionRate, err := s.liveRepo.GetCourseCompletionRate(ctx, courseID)
		if err != nil {
			logger.Error(ctx, "Failed to get completion rate", "course_id", courseID, "error", err)
		}

		avgScore, _, _, err := s.liveRepo.GetQuizStatsForCourse(ctx, courseID)
		if err != nil {
			logger.Error(ctx, "Failed to get quiz stats", "course_id", courseID, "error", err)
		}

		revenue, err := s.liveRepo.GetCourseRevenue(ctx, courseID)
		if err != nil {
			logger.Error(ctx, "Failed to get course revenue", "course_id", courseID, "error", err)
		}
		totalRevenue += revenue

		courseStats = append(courseStats, TeacherCourseStats{
			CourseID:         courseID,
			CourseTitle:      courseTitles[courseID],
			TotalEnrolled:    total,
			CompletionRate:   completionRate,
			AverageQuizScore: avgScore,
			Revenue:          revenue,
		})
	}

	resp := &TeacherAnalyticsResponse{
		TeacherID:    cmd.TeacherID,
		Courses:      courseStats,
		TotalRevenue: totalRevenue,
	}

	s.cacheResponse(ctx, cacheKey, resp)
	return resp, nil
}

// GetTeacherStudentAnalytics returns analytics for a student only when the student is enrolled
// in at least one course owned by the authenticated teacher.
func (s *service) GetTeacherStudentAnalytics(ctx context.Context, cmd GetTeacherStudentAnalyticsCommand) (*StudentAnalyticsResponse, error) {
	cacheKey := fmt.Sprintf("analytics:teacher:%s:student:%s", cmd.TeacherID, cmd.StudentID)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var resp StudentAnalyticsResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return &resp, nil
		}
	}

	teacherCourseIDs, err := s.liveRepo.GetTeacherCourseIDs(ctx, cmd.TeacherID)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to get teacher courses")
	}
	teacherCourses := make(map[uuid.UUID]struct{}, len(teacherCourseIDs))
	for _, courseID := range teacherCourseIDs {
		teacherCourses[courseID] = struct{}{}
	}

	courseProgress, err := s.liveRepo.GetStudentCourseProgress(ctx, cmd.StudentID)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to get student course progress")
	}
	scopedProgress := make([]CourseProgressEntry, 0, len(courseProgress))
	for _, progress := range courseProgress {
		if _, ok := teacherCourses[progress.CourseID]; ok {
			scopedProgress = append(scopedProgress, progress)
		}
	}
	if len(scopedProgress) == 0 {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "student is not enrolled in a course owned by this teacher")
	}

	pointsHistory, err := s.liveRepo.GetStudentPointsHistory30d(ctx, cmd.StudentID)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to get points history")
	}

	studentsAhead, err := s.liveRepo.CountStudentsWithMoreTotalPoints(ctx, cmd.StudentID)
	if err != nil {
		return nil, apperrors.NewInternalError("ANALYTICS_QUERY_FAILED", "failed to get global rank")
	}

	resp := &StudentAnalyticsResponse{
		StudentID:      cmd.StudentID,
		PointsHistory:  pointsHistory,
		CourseProgress: scopedProgress,
		GlobalRank:     studentsAhead + 1,
	}

	s.cacheResponse(ctx, cacheKey, resp)
	return resp, nil
}

// cacheResponse serialises and caches a response for 5 minutes.
func (s *service) cacheResponse(ctx context.Context, key string, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	if err := s.cache.Set(ctx, key, string(data), cacheTTL); err != nil {
		logger.Error(ctx, "Failed to cache analytics response", "key", key, "error", err)
	}
}

// GetAdminStats returns admin platform stats.
func (s *service) GetAdminStats(ctx context.Context) (*AdminStatsResponse, error) {
	cacheKey := "analytics:admin_stats"
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var resp AdminStatsResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return &resp, nil
		}
	}

	result, err := s.liveRepo.GetAdminStats(ctx)
	if err != nil {
		return nil, err
	}

	s.cacheResponse(ctx, cacheKey, result)
	return result, nil
}

// ListCoursesAnalytics returns course analytics list.
func (s *service) ListCoursesAnalytics(ctx context.Context) (*CourseAnalyticsListResponse, error) {
	cacheKey := "analytics:courses_list"
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var resp CourseAnalyticsListResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return &resp, nil
		}
	}

	items, err := s.liveRepo.ListCoursesAnalytics(ctx)
	if err != nil {
		return nil, err
	}

	resp := &CourseAnalyticsListResponse{Items: items}
	s.cacheResponse(ctx, cacheKey, resp)
	return resp, nil
}

// ListStudentsAnalytics returns student analytics list.
func (s *service) ListStudentsAnalytics(ctx context.Context) (*StudentAnalyticsListResponse, error) {
	cacheKey := "analytics:students_list"
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var resp StudentAnalyticsListResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return &resp, nil
		}
	}

	items, err := s.liveRepo.ListStudentsAnalytics(ctx)
	if err != nil {
		return nil, err
	}

	resp := &StudentAnalyticsListResponse{Items: items}
	s.cacheResponse(ctx, cacheKey, resp)
	return resp, nil
}

// GetStudentDashboard returns dashboard information for a specific student.
func (s *service) GetStudentDashboard(ctx context.Context, studentID uuid.UUID) (*StudentDashboardResponse, error) {
	cacheKey := fmt.Sprintf("analytics:student_dashboard:%s", studentID)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var resp StudentDashboardResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return &resp, nil
		}
	}

	stats, err := s.liveRepo.GetStudentDashboardStats(ctx, studentID)
	if err != nil {
		return nil, err
	}

	continueLearning, err := s.liveRepo.GetStudentDashboardEnrollments(ctx, studentID)
	if err != nil {
		return nil, err
	}
	if continueLearning == nil {
		continueLearning = make([]StudentDashboardEnrollment, 0)
	}

	upcomingLive, err := s.liveRepo.GetStudentDashboardUpcomingSessions(ctx, studentID)
	if err != nil {
		return nil, err
	}
	if upcomingLive == nil {
		upcomingLive = make([]UpcomingLiveSession, 0)
	}

	recentAchievements := make([]RecentAchievement, 0)

	resp := &StudentDashboardResponse{
		Stats:              *stats,
		ContinueLearning:   continueLearning,
		UpcomingLive:       upcomingLive,
		RecentAchievements: recentAchievements,
	}

	s.cacheResponse(ctx, cacheKey, resp)
	return resp, nil
}

// InvalidateStudentDashboard removes the cached student dashboard snapshot so
// the next GetStudentDashboard call recomputes from the database. The cache
// is also used by related stats and continue_learning responses, so dropping
// the dashboard key is enough to refresh every read on the /student page.
func (s *service) InvalidateStudentDashboard(ctx context.Context, studentID uuid.UUID) error {
	cacheKey := fmt.Sprintf("analytics:student_dashboard:%s", studentID)
	if err := s.cache.Delete(ctx, cacheKey); err != nil {
		logger.Error(ctx, "Failed to invalidate student dashboard cache", "key", cacheKey, "error", err)
		return err
	}
	return nil
}
