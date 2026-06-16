package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"lms-backend/internal/application/enrollments"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

type EnrollmentsHandler struct {
	service enrollments.Service
}

func NewEnrollmentsHandler(service enrollments.Service) *EnrollmentsHandler {
	return &EnrollmentsHandler{service: service}
}

// CreateEnrollment handles POST /v1/enrollments
//
// @Summary      Create enrollment
// @Description  Enrolls the authenticated student in a free course
// @Tags         enrollments
// @Accept       json
// @Produce      json
// @Param        body  body  object{course_id=string}  true  "Enrollment request"
// @Success      201  {object}  enrollments.EnrollmentResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/enrollments [post]
func (h *EnrollmentsHandler) CreateEnrollment(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	var req struct {
		CourseID string `json:"course_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	courseID, err := uuid.Parse(req.CourseID)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}

	cmd := enrollments.EnrollFreeCommand{
		StudentID: userID,
		CourseID:  courseID,
	}

	enrollment, err := h.service.EnrollFree(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(enrollment)
}

// ListStudentEnrollments handles GET /v1/student/enrollments
func (h *EnrollmentsHandler) ListStudentEnrollments(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	page := 1
	limit := 20
	if raw := r.URL.Query().Get("page"); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed > 0 {
			page = parsed
		}
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	enrollmentList, total, err := h.service.ListStudentEnrollments(r.Context(), userID, page, limit)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"data": enrollmentList,
		"meta": map[string]int{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + limit - 1) / limit,
		},
	})
}

// GetStreamingSignedURL handles GET /v1/stream/lessons/:lessonId/signed-url
//
// @Summary      Get streaming signed URL
// @Description  Returns a presigned URL for streaming a lesson video; requires the student to be enrolled in the course
// @Tags         enrollments
// @Produce      json
// @Param        lessonId  path  string  true  "Lesson ID"
// @Success      200  {object}  enrollments.StreamingSignedURLResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/stream/lessons/{lessonId}/signed-url [get]
func (h *EnrollmentsHandler) GetStreamingSignedURL(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	lessonIDStr := r.PathValue("lessonId")
	lessonID, err := uuid.Parse(lessonIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid lesson ID"))
		return
	}

	cmd := enrollments.GetStreamingSignedURLCommand{
		UserID:   userID,
		LessonID: lessonID,
	}

	result, err := h.service.GetStreamingSignedURL(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// UpdateLessonProgress handles POST /v1/enrollments/:courseId/lessons/:lessonId/progress
//
// @Summary      Update lesson progress
// @Description  Records or updates the authenticated student's watch progress for a lesson
// @Tags         enrollments
// @Accept       json
// @Produce      json
// @Param        courseId   path  string  true  "Course ID"
// @Param        lessonId   path  string  true  "Lesson ID"
// @Param        body       body  object{position_seconds=int,watched_percent=number,completed=bool}  true  "Progress update request"
// @Success      200  {object}  enrollments.LessonProgressResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/enrollments/{courseId}/lessons/{lessonId}/progress [post]
func (h *EnrollmentsHandler) UpdateLessonProgress(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	courseIDStr := r.PathValue("courseId")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}

	lessonIDStr := r.PathValue("lessonId")
	lessonID, err := uuid.Parse(lessonIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid lesson ID"))
		return
	}

	var req struct {
		PositionSeconds int     `json:"position_seconds"`
		WatchedPercent  float64 `json:"watched_percent"`
		Completed       bool    `json:"completed"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	// Get enrollment ID from student and course
	enrollment, err := h.service.GetEnrollment(r.Context(), userID, courseID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	cmd := enrollments.UpdateLessonProgressCommand{
		EnrollmentID:    enrollment.ID,
		LessonID:        lessonID,
		PositionSeconds: req.PositionSeconds,
		WatchedPercent:  req.WatchedPercent,
		Completed:       req.Completed,
	}

	progress, err := h.service.UpdateLessonProgress(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(progress)
}

// GetLessonProgress handles GET /v1/enrollments/:courseId/lessons/:lessonId/progress
func (h *EnrollmentsHandler) GetLessonProgress(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	courseID, err := uuid.Parse(r.PathValue("courseId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}
	lessonID, err := uuid.Parse(r.PathValue("lessonId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid lesson ID"))
		return
	}

	enrollment, err := h.service.GetEnrollment(r.Context(), userID, courseID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	progress, err := h.service.GetLessonProgress(r.Context(), enrollment.ID, lessonID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, progress)
}
