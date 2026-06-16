package analytics

import (
	"context"
	"testing"
	"time"

	domainanalytics "lms-backend/internal/domain/analytics"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// ─── Mock domain/analytics.Repository ────────────────────────────────────────

type mockAnalyticsRepo struct {
	courseIDs      []uuid.UUID
	enrollByID     map[uuid.UUID]struct{ total, free, paid int }
	totalStudents  int
	activeStudents int
	totalEnroll    int
	freeEnroll     int
	paidEnroll     int
}

func newMockAnalyticsRepo(courseIDs []uuid.UUID) *mockAnalyticsRepo {
	return &mockAnalyticsRepo{
		courseIDs:  courseIDs,
		enrollByID: make(map[uuid.UUID]struct{ total, free, paid int }),
	}
}

func (m *mockAnalyticsRepo) UpsertEnrollmentStat(_ context.Context, _ *domainanalytics.EnrollmentStat) error {
	return nil
}
func (m *mockAnalyticsRepo) UpsertProgressStat(_ context.Context, _ *domainanalytics.ProgressStat) error {
	return nil
}
func (m *mockAnalyticsRepo) UpsertRevenueStat(_ context.Context, _ *domainanalytics.RevenueStat) error {
	return nil
}
func (m *mockAnalyticsRepo) UpsertDAUStat(_ context.Context, _ *domainanalytics.DAUStat) error {
	return nil
}
func (m *mockAnalyticsRepo) GetEnrollmentStatsByDateRange(_ context.Context, _, _ time.Time) ([]*domainanalytics.EnrollmentStat, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) GetEnrollmentStatsByCourse(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]*domainanalytics.EnrollmentStat, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) GetProgressStatsByCourse(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]*domainanalytics.ProgressStat, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) GetRevenueStatsByDateRange(_ context.Context, _, _ time.Time) ([]*domainanalytics.RevenueStat, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) GetDAUStatsByDateRange(_ context.Context, _, _ time.Time) ([]*domainanalytics.DAUStat, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) CountTotalStudents(_ context.Context) (int, error) {
	return m.totalStudents, nil
}
func (m *mockAnalyticsRepo) CountActiveStudents(_ context.Context, _ time.Time) (int, error) {
	return m.activeStudents, nil
}
func (m *mockAnalyticsRepo) CountTotalEnrollments(_ context.Context) (total, free, paid int, err error) {
	return m.totalEnroll, m.freeEnroll, m.paidEnroll, nil
}
func (m *mockAnalyticsRepo) CountEnrollmentsByCourse(_ context.Context, courseID uuid.UUID) (total, free, paid int, err error) {
	if e, ok := m.enrollByID[courseID]; ok {
		return e.total, e.free, e.paid, nil
	}
	return 0, 0, 0, nil
}
func (m *mockAnalyticsRepo) SumRevenueByDateRange(_ context.Context, _, _ time.Time) (total, courseRev, bookRev float64, err error) {
	return 0, 0, 0, nil
}
func (m *mockAnalyticsRepo) CountDAU(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}
func (m *mockAnalyticsRepo) GetModuleProgressForCourse(_ context.Context, _ uuid.UUID, _ time.Time) ([]*domainanalytics.ProgressStat, error) {
	return nil, nil
}
func (m *mockAnalyticsRepo) ListCourseIDs(_ context.Context) ([]uuid.UUID, error) {
	return m.courseIDs, nil
}

// ─── Mock application/analytics.LiveQueryRepo ────────────────────────────────

type mockLiveQueryRepo struct {
	// teacherCourses maps teacher_id → []course_id (the critical scoping data)
	teacherCourses map[uuid.UUID][]uuid.UUID
	courseTitles   map[uuid.UUID]string
	allCourseIDs   []uuid.UUID
}

func newMockLiveQueryRepo() *mockLiveQueryRepo {
	return &mockLiveQueryRepo{
		teacherCourses: make(map[uuid.UUID][]uuid.UUID),
		courseTitles:   make(map[uuid.UUID]string),
	}
}

func (m *mockLiveQueryRepo) CountCourses(_ context.Context) (all, free, paid int, err error) {
	return len(m.allCourseIDs), 0, 0, nil
}
func (m *mockLiveQueryRepo) GetModuleTitles(_ context.Context, _ uuid.UUID) (map[uuid.UUID]string, error) {
	return map[uuid.UUID]string{}, nil
}
func (m *mockLiveQueryRepo) GetQuizStatsForCourse(_ context.Context, _ uuid.UUID) (avgScore, passRate float64, totalAttempts int, err error) {
	return 0, 0, 0, nil
}
func (m *mockLiveQueryRepo) GetStudentProgressInCourse(_ context.Context, _ uuid.UUID, _, _ int) ([]StudentProgressEntry, int, error) {
	return []StudentProgressEntry{}, 0, nil
}
func (m *mockLiveQueryRepo) GetStudentPointsHistory30d(_ context.Context, _ uuid.UUID) ([]PointsHistoryEntry, error) {
	return []PointsHistoryEntry{}, nil
}
func (m *mockLiveQueryRepo) GetStudentCourseProgress(_ context.Context, _ uuid.UUID) ([]CourseProgressEntry, error) {
	return []CourseProgressEntry{}, nil
}
func (m *mockLiveQueryRepo) CountStudentsWithMoreTotalPoints(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockLiveQueryRepo) GetAdminStats(_ context.Context) (*AdminStatsResponse, error) {
	return &AdminStatsResponse{}, nil
}
func (m *mockLiveQueryRepo) ListCoursesAnalytics(_ context.Context) ([]CourseAnalyticsListEntry, error) {
	return []CourseAnalyticsListEntry{}, nil
}
func (m *mockLiveQueryRepo) ListStudentsAnalytics(_ context.Context) ([]StudentAnalyticsListEntry, error) {
	return []StudentAnalyticsListEntry{}, nil
}
func (m *mockLiveQueryRepo) GetStudentDashboardStats(_ context.Context, _ uuid.UUID) (*StudentDashboardStats, error) {
	return &StudentDashboardStats{}, nil
}
func (m *mockLiveQueryRepo) GetStudentDashboardEnrollments(_ context.Context, _ uuid.UUID) ([]StudentDashboardEnrollment, error) {
	return []StudentDashboardEnrollment{}, nil
}
func (m *mockLiveQueryRepo) GetStudentDashboardUpcomingSessions(_ context.Context, _ uuid.UUID) ([]UpcomingLiveSession, error) {
	return []UpcomingLiveSession{}, nil
}

// GetTeacherCourseIDs returns ONLY the courses owned by the given teacher.
// This is the critical scoping method for Property 57. (Requirement 23.5)
func (m *mockLiveQueryRepo) GetTeacherCourseIDs(_ context.Context, teacherID uuid.UUID) ([]uuid.UUID, error) {
	return m.teacherCourses[teacherID], nil
}

func (m *mockLiveQueryRepo) GetCourseTitles(_ context.Context, courseIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	result := make(map[uuid.UUID]string)
	for _, id := range courseIDs {
		if title, ok := m.courseTitles[id]; ok {
			result[id] = title
		}
	}
	return result, nil
}
func (m *mockLiveQueryRepo) GetCourseRevenue(_ context.Context, _ uuid.UUID) (float64, error) {
	return 0, nil
}
func (m *mockLiveQueryRepo) GetCourseCompletionRate(_ context.Context, _ uuid.UUID) (float64, error) {
	return 0, nil
}

// ─── Mock Cache ───────────────────────────────────────────────────────────────

type mockCache struct{}

func (m *mockCache) Get(_ context.Context, _ string) (string, error) { return "", nil }
func (m *mockCache) Set(_ context.Context, _ string, _ string, _ time.Duration) error {
	return nil
}

// ─── Property 57 ─────────────────────────────────────────────────────────────

// TestProperty57_TeacherAnalyticsScopedToOwnCourses verifies that
// GetTeacherAnalytics returns data exclusively for the authenticated teacher's
// own courses — no data from other teachers' courses is included.
//
// **Validates: Requirements 23.5**
func TestProperty57_TeacherAnalyticsScopedToOwnCourses(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx := context.Background()

		// Generate 2–4 teachers, each with 1–4 courses
		numTeachers := rapid.IntRange(2, 4).Draw(t, "num_teachers")
		coursesPerTeacher := rapid.IntRange(1, 4).Draw(t, "courses_per_teacher")

		liveRepo := newMockLiveQueryRepo()

		teacherIDs := make([]uuid.UUID, numTeachers)
		for i := range teacherIDs {
			teacherIDs[i] = uuid.New()
			courses := make([]uuid.UUID, coursesPerTeacher)
			for j := range courses {
				courses[j] = uuid.New()
				liveRepo.courseTitles[courses[j]] = "Course-" + courses[j].String()[:8]
				liveRepo.allCourseIDs = append(liveRepo.allCourseIDs, courses[j])
			}
			liveRepo.teacherCourses[teacherIDs[i]] = courses
		}

		analyticsRepo := newMockAnalyticsRepo(liveRepo.allCourseIDs)
		svc := NewService(analyticsRepo, liveRepo, &mockCache{})

		now := time.Now().UTC()

		for _, teacherID := range teacherIDs {
			ownCourseIDs := liveRepo.teacherCourses[teacherID]

			// Build sets for assertion
			ownCourseSet := make(map[uuid.UUID]bool, len(ownCourseIDs))
			for _, id := range ownCourseIDs {
				ownCourseSet[id] = true
			}

			otherCourseSet := make(map[uuid.UUID]bool)
			for _, otherID := range teacherIDs {
				if otherID == teacherID {
					continue
				}
				for _, id := range liveRepo.teacherCourses[otherID] {
					otherCourseSet[id] = true
				}
			}

			resp, err := svc.GetTeacherAnalytics(ctx, GetTeacherAnalyticsCommand{
				TeacherID: teacherID,
				From:      now.AddDate(0, 0, -30),
				To:        now,
			})
			if err != nil {
				t.Fatalf("GetTeacherAnalytics failed for teacher %v: %v", teacherID, err)
			}

			// Property: response is attributed to the correct teacher
			if resp.TeacherID != teacherID {
				t.Fatalf("response TeacherID %v != requesting teacher %v", resp.TeacherID, teacherID)
			}

			// Property: every course in the response belongs to this teacher (Requirement 23.5)
			for _, cs := range resp.Courses {
				if !ownCourseSet[cs.CourseID] {
					t.Fatalf(
						"teacher %v received analytics for course %v which does NOT belong to them (Requirement 23.5)",
						teacherID, cs.CourseID,
					)
				}
			}

			// Property: no course from another teacher appears in the response
			for _, cs := range resp.Courses {
				if otherCourseSet[cs.CourseID] {
					t.Fatalf(
						"teacher %v received analytics for course %v which belongs to another teacher (Requirement 23.5)",
						teacherID, cs.CourseID,
					)
				}
			}

			// Property: the response contains exactly the teacher's own courses
			if len(resp.Courses) != len(ownCourseIDs) {
				t.Fatalf(
					"teacher %v: expected %d courses in analytics, got %d",
					teacherID, len(ownCourseIDs), len(resp.Courses),
				)
			}

			// Property: all of the teacher's own courses are present
			responseCourseSet := make(map[uuid.UUID]bool, len(resp.Courses))
			for _, cs := range resp.Courses {
				responseCourseSet[cs.CourseID] = true
			}
			for _, ownID := range ownCourseIDs {
				if !responseCourseSet[ownID] {
					t.Fatalf(
						"teacher %v: own course %v is missing from analytics response",
						teacherID, ownID,
					)
				}
			}
		}
	})
}
