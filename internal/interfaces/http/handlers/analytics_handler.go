package handlers

import (
	"net/http"
	"strconv"
	"time"

	"lms-backend/internal/application/analytics"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// AnalyticsHandler handles HTTP requests for the analytics bounded context.
type AnalyticsHandler struct {
	service analytics.Service
}

// NewAnalyticsHandler creates a new AnalyticsHandler.
func NewAnalyticsHandler(service analytics.Service) *AnalyticsHandler {
	return &AnalyticsHandler{service: service}
}

// GetAdminOverview handles GET /v1/admin/analytics/overview
// Requirement 23.1
//
// @Summary      Get admin analytics overview
// @Description  Returns platform-wide analytics including course counts, enrollments, revenue, and daily active users
// @Tags         analytics
// @Produce      json
// @Param        from  query  string  false  "Start date (YYYY-MM-DD)"
// @Param        to    query  string  false  "End date (YYYY-MM-DD)"
// @Success      200  {object}  analytics.AdminOverviewResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/analytics/overview [get]
func (h *AnalyticsHandler) GetAdminOverview(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseDateRange(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	cmd := analytics.GetAdminOverviewCommand{From: from, To: to}
	result, err := h.service.GetAdminOverview(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// GetCourseAnalytics handles GET /v1/admin/analytics/courses/:courseId
// Requirement 23.2
//
// @Summary      Get course analytics
// @Description  Returns per-course analytics including module completion rates, quiz stats, and enrollment over time
// @Tags         analytics
// @Produce      json
// @Param        courseId  path   string  true   "Course ID"
// @Param        from      query  string  false  "Start date (YYYY-MM-DD)"
// @Param        to        query  string  false  "End date (YYYY-MM-DD)"
// @Success      200  {object}  analytics.CourseAnalyticsResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/analytics/courses/{courseId} [get]
func (h *AnalyticsHandler) GetCourseAnalytics(w http.ResponseWriter, r *http.Request) {
	courseID, err := parseUUIDParam(r, "courseId")
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	from, to, err := parseDateRange(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	cmd := analytics.GetCourseAnalyticsCommand{CourseID: courseID, From: from, To: to}
	result, err := h.service.GetCourseAnalytics(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// GetCourseStudents handles GET /v1/admin/analytics/courses/:courseId/students
// Requirement 23.3
//
// @Summary      Get course students
// @Description  Returns a paginated list of students with their progress within a specific course
// @Tags         analytics
// @Produce      json
// @Param        courseId  path   string  true   "Course ID"
// @Param        page      query  int     false  "Page number"     default(1)
// @Param        limit     query  int     false  "Items per page"  default(20)
// @Success      200  {object}  analytics.CourseStudentsResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/analytics/courses/{courseId}/students [get]
func (h *AnalyticsHandler) GetCourseStudents(w http.ResponseWriter, r *http.Request) {
	courseID, err := parseUUIDParam(r, "courseId")
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	page := 1
	limit := 20
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	cmd := analytics.GetCourseStudentsCommand{CourseID: courseID, Page: page, Limit: limit}
	result, err := h.service.GetCourseStudents(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// GetStudentAnalytics handles GET /v1/admin/analytics/students/:studentId
// Requirement 23.4
//
// @Summary      Get student analytics
// @Description  Returns analytics for a specific student including points history, course progress, and global rank
// @Tags         analytics
// @Produce      json
// @Param        studentId  path   string  true   "Student ID"
// @Param        from       query  string  false  "Start date (YYYY-MM-DD)"
// @Param        to         query  string  false  "End date (YYYY-MM-DD)"
// @Success      200  {object}  analytics.StudentAnalyticsResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/analytics/students/{studentId} [get]
func (h *AnalyticsHandler) GetStudentAnalytics(w http.ResponseWriter, r *http.Request) {
	studentID, err := parseUUIDParam(r, "studentId")
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	cmd := analytics.GetStudentAnalyticsCommand{StudentID: studentID}
	result, err := h.service.GetStudentAnalytics(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// GetTeacherAnalytics handles GET /v1/teacher/analytics
// Scoped exclusively to the authenticated teacher's own courses. Requirement 23.5
//
// @Summary      Get teacher analytics
// @Description  Returns analytics scoped exclusively to the authenticated teacher's own courses
// @Tags         analytics
// @Produce      json
// @Param        from  query  string  false  "Start date (YYYY-MM-DD)"
// @Param        to    query  string  false  "End date (YYYY-MM-DD)"
// @Success      200  {object}  analytics.TeacherAnalyticsResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/teacher/analytics [get]
func (h *AnalyticsHandler) GetTeacherAnalytics(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	from, to, err := parseDateRange(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	cmd := analytics.GetTeacherAnalyticsCommand{TeacherID: userID, From: from, To: to}
	result, err := h.service.GetTeacherAnalytics(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// GetTeacherStudentAnalytics handles GET /v1/teacher/analytics/students/:studentId.
func (h *AnalyticsHandler) GetTeacherStudentAnalytics(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	studentID, err := parseUUIDParam(r, "studentId")
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	cmd := analytics.GetTeacherStudentAnalyticsCommand{TeacherID: userID, StudentID: studentID}
	result, err := h.service.GetTeacherStudentAnalytics(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// parseDateRange parses optional `from` and `to` query params (YYYY-MM-DD).
// Defaults to the last 30 days if not provided.
func parseDateRange(r *http.Request) (from, to time.Time, err error) {
	now := time.Now().UTC()
	to = now
	from = now.AddDate(0, 0, -30)

	if f := r.URL.Query().Get("from"); f != "" {
		parsed, parseErr := time.Parse("2006-01-02", f)
		if parseErr != nil {
			return from, to, apperrors.NewSimpleValidationError("INVALID_DATE", "from must be in YYYY-MM-DD format")
		}
		from = parsed
	}
	if t := r.URL.Query().Get("to"); t != "" {
		parsed, parseErr := time.Parse("2006-01-02", t)
		if parseErr != nil {
			return from, to, apperrors.NewSimpleValidationError("INVALID_DATE", "to must be in YYYY-MM-DD format")
		}
		to = parsed
	}
	return from, to, nil
}

// parseUUIDParam extracts a UUID path parameter by name.
func parseUUIDParam(r *http.Request, name string) (uuid.UUID, error) {
	raw := r.PathValue(name)
	if raw == "" {
		return uuid.Nil, apperrors.NewSimpleValidationError("MISSING_PARAM", name+" is required")
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apperrors.NewSimpleValidationError("INVALID_ID", "invalid "+name)
	}
	return id, nil
}

// GetAdminStats handles GET /v1/admin/stats
func (h *AnalyticsHandler) GetAdminStats(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.GetAdminStats(r.Context())
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// ListCoursesAnalytics handles GET /v1/admin/analytics/courses
func (h *AnalyticsHandler) ListCoursesAnalytics(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.ListCoursesAnalytics(r.Context())
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// ListStudentsAnalytics handles GET /v1/admin/analytics/students
func (h *AnalyticsHandler) ListStudentsAnalytics(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.ListStudentsAnalytics(r.Context())
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// GetStudentDashboard handles GET /v1/student/dashboard
func (h *AnalyticsHandler) GetStudentDashboard(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	result, err := h.service.GetStudentDashboard(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}
