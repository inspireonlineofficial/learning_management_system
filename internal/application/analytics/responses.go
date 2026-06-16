package analytics

import (
	"time"

	"github.com/google/uuid"
)

// AdminOverviewResponse is the response for GET /v1/admin/analytics/overview.
// Requirement 23.1
type AdminOverviewResponse struct {
	TotalCourses      CourseCounts     `json:"total_courses"`
	TotalStudents     int              `json:"total_students"`
	ActiveStudents30d int              `json:"active_students_30d"`
	TotalEnrollments  EnrollmentCounts `json:"total_enrollments"`
	Revenue           RevenueSummary   `json:"revenue"`
	DailyActiveUsers  []DAUEntry       `json:"daily_active_users"`
}

// CourseCounts breaks down course counts by price type.
type CourseCounts struct {
	All  int `json:"all"`
	Free int `json:"free"`
	Paid int `json:"paid"`
}

// EnrollmentCounts breaks down enrollment counts by type.
type EnrollmentCounts struct {
	All  int `json:"all"`
	Free int `json:"free"`
	Paid int `json:"paid"`
}

// RevenueSummary provides revenue totals for different periods.
type RevenueSummary struct {
	ThisMonth float64 `json:"this_month"`
	ThisYear  float64 `json:"this_year"`
	AllTime   float64 `json:"all_time"`
}

// DAUEntry is a single day's active user count.
type DAUEntry struct {
	Date        string `json:"date"`
	ActiveUsers int    `json:"active_users"`
}

// CourseAnalyticsResponse is the response for GET /v1/admin/analytics/courses/:courseId.
// Requirement 23.2
type CourseAnalyticsResponse struct {
	CourseID           uuid.UUID              `json:"course_id"`
	ModuleCompletions  []ModuleCompletionRate `json:"module_completions"`
	DropOffModule      *ModuleDropOff         `json:"drop_off_module,omitempty"`
	QuizStats          QuizStats              `json:"quiz_stats"`
	EnrollmentOverTime []EnrollmentEntry      `json:"enrollment_over_time"`
}

// ModuleCompletionRate holds completion data for a single module.
type ModuleCompletionRate struct {
	ModuleID          uuid.UUID `json:"module_id"`
	ModuleTitle       string    `json:"module_title"`
	TotalStudents     int       `json:"total_students"`
	CompletedStudents int       `json:"completed_students"`
	CompletionRate    float64   `json:"completion_rate"`
}

// ModuleDropOff identifies the module with the highest drop-off.
type ModuleDropOff struct {
	ModuleID    uuid.UUID `json:"module_id"`
	ModuleTitle string    `json:"module_title"`
	DropOffRate float64   `json:"drop_off_rate"`
}

// QuizStats aggregates quiz performance for a course.
type QuizStats struct {
	AverageScore  float64 `json:"average_score"`
	PassRate      float64 `json:"pass_rate"`
	TotalAttempts int     `json:"total_attempts"`
}

// EnrollmentEntry is a single day's enrollment count for a course.
type EnrollmentEntry struct {
	Date          string `json:"date"`
	TotalEnrolled int    `json:"total_enrolled"`
}

// CourseStudentsResponse is the response for GET /v1/admin/analytics/courses/:courseId/students.
// Requirement 23.3
type CourseStudentsResponse struct {
	Students []StudentProgressEntry `json:"students"`
	Meta     PaginationMeta         `json:"meta"`
}

// StudentProgressEntry holds per-student progress within a course.
type StudentProgressEntry struct {
	StudentID              uuid.UUID  `json:"student_id"`
	StudentName            string     `json:"student_name"`
	OverallProgressPercent float64    `json:"overall_progress_percent"`
	ModulesCompleted       int        `json:"modules_completed"`
	ModulesInProgress      int        `json:"modules_in_progress"`
	LastActiveAt           *time.Time `json:"last_active_at"`
}

// StudentAnalyticsResponse is the response for GET /v1/admin/analytics/students/:studentId.
// Requirement 23.4
type StudentAnalyticsResponse struct {
	StudentID      uuid.UUID             `json:"student_id"`
	PointsHistory  []PointsHistoryEntry  `json:"points_history_30d"`
	CourseProgress []CourseProgressEntry `json:"course_progress"`
	GlobalRank     int                   `json:"global_rank"`
}

// PointsHistoryEntry is a single day's points total for a student.
type PointsHistoryEntry struct {
	Date   string `json:"date"`
	Points int    `json:"points"`
}

// CourseProgressEntry holds a student's progress in a single course.
type CourseProgressEntry struct {
	CourseID        uuid.UUID `json:"course_id"`
	CourseTitle     string    `json:"course_title"`
	ProgressPercent float64   `json:"progress_percent"`
	EnrolledAt      time.Time `json:"enrolled_at"`
}

// TeacherAnalyticsResponse is the response for GET /v1/teacher/analytics.
// Scoped exclusively to the authenticated teacher's own courses. Requirement 23.5
type TeacherAnalyticsResponse struct {
	TeacherID    uuid.UUID            `json:"teacher_id"`
	Courses      []TeacherCourseStats `json:"courses"`
	TotalRevenue float64              `json:"total_revenue"`
}

// TeacherCourseStats holds analytics for a single course owned by the teacher.
type TeacherCourseStats struct {
	CourseID         uuid.UUID `json:"course_id"`
	CourseTitle      string    `json:"course_title"`
	TotalEnrolled    int       `json:"total_enrolled"`
	CompletionRate   float64   `json:"completion_rate"`
	AverageQuizScore float64   `json:"average_quiz_score"`
	Revenue          float64   `json:"revenue"`
}

// PaginationMeta holds pagination metadata.
type PaginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// AdminStatsResponse is the response for GET /v1/admin/stats.
type AdminStatsResponse struct {
	TotalUsers       int     `json:"total_users"`
	ActiveUsers30d   int     `json:"active_users_30d"`
	TotalCourses     int     `json:"total_courses"`
	PublishedCourses int     `json:"published_courses"`
	TotalEnrollments int     `json:"total_enrollments"`
	TotalRevenue     float64 `json:"total_revenue"`
}

// CourseAnalyticsListEntry represents a single course in the analytics list.
type CourseAnalyticsListEntry struct {
	CourseID    uuid.UUID `json:"course_id"`
	Enrolled    int       `json:"enrolled"`
	Completed   int       `json:"completed"`
	AvgProgress float64   `json:"avg_progress"`
	Revenue     float64   `json:"revenue"`
	Rating      float64   `json:"rating"`
}

// CourseAnalyticsListResponse wraps the list of course analytics.
type CourseAnalyticsListResponse struct {
	Items []CourseAnalyticsListEntry `json:"items"`
}

// StudentAnalyticsListEntry represents a single student in the analytics list.
type StudentAnalyticsListEntry struct {
	StudentID       uuid.UUID `json:"student_id"`
	EnrolledCourses int       `json:"enrolled_courses"`
	HoursLearned    float64   `json:"hours_learned"`
	AvgScore        float64   `json:"avg_score"`
	Streak          int       `json:"streak"`
	Certificates    int       `json:"certificates"`
}

// StudentAnalyticsListResponse wraps the list of student analytics.
type StudentAnalyticsListResponse struct {
	Items []StudentAnalyticsListEntry `json:"items"`
}

// StudentDashboardResponse is the response for GET /v1/student/dashboard.
type StudentDashboardResponse struct {
	Stats              StudentDashboardStats        `json:"stats"`
	ContinueLearning   []StudentDashboardEnrollment `json:"continue_learning"`
	UpcomingLive       []UpcomingLiveSession        `json:"upcoming_live"`
	RecentAchievements []RecentAchievement          `json:"recent_achievements"`
}

// StudentDashboardStats holds summary stats for the student dashboard.
type StudentDashboardStats struct {
	EnrolledCourses  int     `json:"enrolled_courses"`
	CompletedCourses int     `json:"completed_courses"`
	HoursLearned     float64 `json:"hours_learned"`
	Points           int     `json:"points"`
	StreakDays       int     `json:"streak_days"`
}

// StudentDashboardEnrollment represents a student's active enrollment on dashboard.
type StudentDashboardEnrollment struct {
	ID              uuid.UUID              `json:"id"`
	EnrolledAt      time.Time              `json:"enrolled_at"`
	ProgressPercent float64                `json:"progress_percent"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	Course          StudentDashboardCourse `json:"course"`
}

// StudentDashboardCourse represents course details for student dashboard enrollment.
type StudentDashboardCourse struct {
	ID       uuid.UUID                 `json:"id"`
	Title    string                    `json:"title"`
	CoverURL string                    `json:"cover_url,omitempty"`
	Category *StudentDashboardCategory `json:"category,omitempty"`
}

// StudentDashboardCategory represents a category/subject.
type StudentDashboardCategory struct {
	Name string `json:"name"`
}

// UpcomingLiveSession represents a scheduled live session for a student.
type UpcomingLiveSession struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	StartsAt    time.Time `json:"starts_at"`
	CourseTitle string    `json:"course_title"`
}

// RecentAchievement represents an award or certificate earned.
type RecentAchievement struct {
	ID       uuid.UUID `json:"id"`
	Title    string    `json:"title"`
	Icon     string    `json:"icon,omitempty"`
	EarnedAt time.Time `json:"earned_at"`
}
