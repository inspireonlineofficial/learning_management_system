package handlers

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"lms-backend/internal/application/courses"
	domainCourses "lms-backend/internal/domain/courses"
	"lms-backend/pkg/apperrors"
	"lms-backend/pkg/pagination"

	"github.com/google/uuid"
)

type CoursesHandler struct {
	service courses.Service
}

func NewCoursesHandler(service courses.Service) *CoursesHandler {
	return &CoursesHandler{service: service}
}

// Public endpoints

// ListPublishedCourses handles GET /v1/courses
//
// @Summary      List published courses
// @Description  Returns a paginated list of published courses with optional filtering by subject, level, price type, and search query
// @Tags         courses
// @Produce      json
// @Param        page        query  int     false  "Page number"         default(1)
// @Param        limit       query  int     false  "Items per page"      default(20)
// @Param        search      query  string  false  "Search query"
// @Param        subject     query  string  false  "Filter by subject"
// @Param        level       query  string  false  "Filter by level"
// @Param        price_type  query  string  false  "Filter by price type"
// @Param        min_price   query  number  false  "Minimum price"
// @Param        max_price   query  number  false  "Maximum price"
// @Param        sort_by     query  string  false  "Sort field"
// @Success      200  {object}  object{courses=[]courses.CourseResponse,meta=object}
// @Failure      400  {object}  ValidationErrorResponse
// @Router       /v1/courses [get]
func (h *CoursesHandler) ListPublishedCourses(w http.ResponseWriter, r *http.Request) {
	params := pagination.ParseParams(r)

	filters := domainCourses.CourseFilters{
		Search:    r.URL.Query().Get("search"),
		Subject:   r.URL.Query().Get("subject"),
		Level:     domainCourses.CourseLevel(r.URL.Query().Get("level")),
		PriceType: domainCourses.PriceType(r.URL.Query().Get("price_type")),
		SortBy:    r.URL.Query().Get("sort_by"),
	}

	if minPrice := r.URL.Query().Get("min_price"); minPrice != "" {
		if val, err := strconv.ParseFloat(minPrice, 64); err == nil {
			filters.MinPrice = &val
		}
	}

	if maxPrice := r.URL.Query().Get("max_price"); maxPrice != "" {
		if val, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			filters.MaxPrice = &val
		}
	}

	courseList, total, err := h.service.ListPublishedCourses(r.Context(), filters, params.Page, params.Limit)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	meta := pagination.Meta{
		Page:       params.Page,
		Limit:      params.Limit,
		Total:      total,
		TotalPages: (total + params.Limit - 1) / params.Limit,
	}

	response := map[string]interface{}{
		"courses": courseList,
		"meta":    meta,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetCourseDetail handles GET /v1/courses/{courseId}
//
// @Summary      Get course detail
// @Description  Returns detailed information about a published course including its module and lesson structure
// @Tags         courses
// @Produce      json
// @Param        courseId  path  string  true  "Course ID"
// @Success      200  {object}  courses.CourseDetailResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /v1/courses/{courseId} [get]
func (h *CoursesHandler) GetCourseDetail(w http.ResponseWriter, r *http.Request) {
	courseIDStr := r.PathValue("courseId")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}

	// Get student ID from context if authenticated
	var studentID *uuid.UUID
	if userID, err := getUserIDFromContext(r); err == nil {
		studentID = &userID
	}

	detail, err := h.service.GetCourseDetail(r.Context(), courseID, studentID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

// ListCourseReviews handles GET /v1/courses/{courseId}/reviews
//
// @Summary      List course reviews
// @Description  Returns a paginated list of reviews for a course along with rating distribution
// @Tags         courses
// @Produce      json
// @Param        courseId  path   string  true   "Course ID"
// @Param        page      query  int     false  "Page number"    default(1)
// @Param        limit     query  int     false  "Items per page" default(20)
// @Success      200  {object}  courses.CourseReviewsResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /v1/courses/{courseId}/reviews [get]
func (h *CoursesHandler) ListCourseReviews(w http.ResponseWriter, r *http.Request) {
	courseIDStr := r.PathValue("courseId")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}

	params := pagination.ParseParams(r)

	reviews, err := h.service.ListCourseReviews(r.Context(), courseID, params.Page, params.Limit)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reviews)
}

// UpsertCourseReview handles POST /v1/courses/{courseId}/reviews
//
// @Summary      Create or update course review
// @Description  Creates a new review or updates an existing review for a course by the authenticated student
// @Tags         courses
// @Accept       json
// @Produce      json
// @Param        courseId  path  string  true  "Course ID"
// @Param        body      body  object{rating=int,comment=string}  true  "Review request"
// @Success      201  {object}  courses.CourseReviewResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/courses/{courseId}/reviews [post]
func (h *CoursesHandler) UpsertCourseReview(w http.ResponseWriter, r *http.Request) {
	courseIDStr := r.PathValue("courseId")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	var req struct {
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	cmd := courses.UpsertCourseReviewCommand{
		CourseID:  courseID,
		StudentID: userID,
		Rating:    req.Rating,
		Comment:   req.Comment,
	}

	review, err := h.service.UpsertCourseReview(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(review)
}

// DeleteCourseReview handles DELETE /v1/courses/{courseId}/reviews/me
func (h *CoursesHandler) DeleteCourseReview(w http.ResponseWriter, r *http.Request) {
	courseID, err := uuid.Parse(r.PathValue("courseId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}
	if err := h.service.DeleteCourseReview(r.Context(), courses.DeleteCourseReviewCommand{CourseID: courseID, StudentID: userID}); err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListCourseComments handles GET /v1/courses/{courseId}/comments
func (h *CoursesHandler) ListCourseComments(w http.ResponseWriter, r *http.Request) {
	courseID, err := uuid.Parse(r.PathValue("courseId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}
	params := pagination.ParseParams(r)
	comments, err := h.service.ListCourseComments(r.Context(), courseID, params.Page, params.Limit)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

// CreateCourseComment handles POST /v1/courses/{courseId}/comments
func (h *CoursesHandler) CreateCourseComment(w http.ResponseWriter, r *http.Request) {
	courseID, err := uuid.Parse(r.PathValue("courseId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}
	role, err := getUserRoleFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	var req struct {
		ModuleID        *uuid.UUID `json:"module_id"`
		LessonID        *uuid.UUID `json:"lesson_id"`
		QuizID          *uuid.UUID `json:"quiz_id"`
		ParentCommentID *uuid.UUID `json:"parent_comment_id"`
		Content         string     `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}
	comment, err := h.service.CreateCourseComment(r.Context(), courses.CreateCourseCommentCommand{
		CourseID:        courseID,
		UserID:          userID,
		Role:            role,
		ModuleID:        req.ModuleID,
		LessonID:        req.LessonID,
		QuizID:          req.QuizID,
		ParentCommentID: req.ParentCommentID,
		Content:         req.Content,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comment)
}

// UpdateCourseComment handles PATCH /v1/courses/comments/{commentId}
func (h *CoursesHandler) UpdateCourseComment(w http.ResponseWriter, r *http.Request) {
	commentID, err := uuid.Parse(r.PathValue("commentId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid comment ID"))
		return
	}
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}
	role, err := getUserRoleFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	var req struct {
		Content  string `json:"content"`
		IsPinned *bool  `json:"is_pinned"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}
	comment, err := h.service.UpdateCourseComment(r.Context(), courses.UpdateCourseCommentCommand{
		CommentID: commentID,
		UserID:    userID,
		Role:      role,
		Content:   req.Content,
		IsPinned:  req.IsPinned,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comment)
}

// DeleteCourseComment handles DELETE /v1/courses/comments/{commentId}
func (h *CoursesHandler) DeleteCourseComment(w http.ResponseWriter, r *http.Request) {
	commentID, err := uuid.Parse(r.PathValue("commentId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid comment ID"))
		return
	}
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}
	role, err := getUserRoleFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	if err := h.service.DeleteCourseComment(r.Context(), courses.DeleteCourseCommentCommand{CommentID: commentID, UserID: userID, Role: role}); err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Teacher endpoints

// ListTeacherCourses handles GET /v1/teacher/courses
func (h *CoursesHandler) ListTeacherCourses(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	params := pagination.ParseParams(r)
	courseList, total, err := h.service.ListTeacherCourses(r.Context(), userID, params.Page, params.Limit)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	meta := pagination.Meta{
		Page:       params.Page,
		Limit:      params.Limit,
		Total:      total,
		TotalPages: (total + params.Limit - 1) / params.Limit,
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"courses": courseList,
		"meta":    meta,
	})
}

// CreateCourse handles POST /v1/teacher/courses
//
// @Summary      Create a course
// @Description  Creates a new course draft for the authenticated teacher
// @Tags         courses
// @Accept       json
// @Produce      json
// @Param        body  body  object{title=string,slug=string,short_description=string,description=string,subject=string,level=string,price_type=string,price=number,currency=string,prerequisites=string,thumbnail_url=string}  true  "Create course request"
// @Success      201  {object}  courses.CourseResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/teacher/courses [post]
func (h *CoursesHandler) CreateCourse(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	var req struct {
		Title                    string  `json:"title"`
		Slug                     string  `json:"slug"`
		Subtitle                 string  `json:"subtitle"`
		ShortDescription         string  `json:"short_description"`
		Description              string  `json:"description"`
		Subject                  string  `json:"subject"`
		Level                    string  `json:"level"`
		PriceType                string  `json:"price_type"`
		Price                    float64 `json:"price"`
		Currency                 string  `json:"currency"`
		Prerequisites            string  `json:"prerequisites"`
		Visibility               string  `json:"visibility"`
		LearningOutcomes         string  `json:"learning_outcomes"`
		Requirements             string  `json:"requirements"`
		TargetAudience           string  `json:"target_audience"`
		EstimatedDurationMinutes int     `json:"estimated_duration_minutes"`
		ThumbnailURL             string  `json:"thumbnail_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	cmd := courses.CreateCourseCommand{
		TeacherID:                userID,
		Title:                    req.Title,
		Slug:                     req.Slug,
		ShortDescription:         firstNonEmpty(req.ShortDescription, req.Subtitle),
		Description:              req.Description,
		Subject:                  req.Subject,
		Level:                    req.Level,
		PriceType:                req.PriceType,
		Price:                    req.Price,
		Currency:                 req.Currency,
		Prerequisites:            req.Prerequisites,
		Visibility:               req.Visibility,
		LearningOutcomes:         req.LearningOutcomes,
		Requirements:             req.Requirements,
		TargetAudience:           req.TargetAudience,
		EstimatedDurationMinutes: req.EstimatedDurationMinutes,
		ThumbnailURL:             req.ThumbnailURL,
	}

	course, err := h.service.CreateCourse(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(course)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func boolDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func optionalUUID(raw string) (*uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// UpdateCourse handles PATCH /v1/teacher/courses/{courseId}
//
// @Summary      Update a course
// @Description  Updates an existing course draft owned by the authenticated teacher
// @Tags         courses
// @Accept       json
// @Produce      json
// @Param        courseId  path  string  true  "Course ID"
// @Param        body      body  object{title=string,slug=string,short_description=string,description=string,subject=string,level=string,price_type=string,price=number,currency=string,prerequisites=string,thumbnail_url=string}  false  "Update course request"
// @Success      200  {object}  courses.CourseResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/teacher/courses/{courseId} [patch]
func (h *CoursesHandler) UpdateCourse(w http.ResponseWriter, r *http.Request) {
	courseIDStr := r.PathValue("courseId")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	var req struct {
		Title                    string  `json:"title"`
		Slug                     string  `json:"slug"`
		ShortDescription         string  `json:"short_description"`
		Description              string  `json:"description"`
		Subject                  string  `json:"subject"`
		Level                    string  `json:"level"`
		PriceType                string  `json:"price_type"`
		Price                    float64 `json:"price"`
		Currency                 string  `json:"currency"`
		Prerequisites            string  `json:"prerequisites"`
		Visibility               string  `json:"visibility"`
		LearningOutcomes         string  `json:"learning_outcomes"`
		Requirements             string  `json:"requirements"`
		TargetAudience           string  `json:"target_audience"`
		EstimatedDurationMinutes int     `json:"estimated_duration_minutes"`
		ThumbnailURL             string  `json:"thumbnail_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	cmd := courses.UpdateCourseCommand{
		CourseID:                 courseID,
		TeacherID:                userID,
		Title:                    req.Title,
		Slug:                     req.Slug,
		ShortDescription:         req.ShortDescription,
		Description:              req.Description,
		Subject:                  req.Subject,
		Level:                    req.Level,
		PriceType:                req.PriceType,
		Price:                    req.Price,
		Currency:                 req.Currency,
		Prerequisites:            req.Prerequisites,
		Visibility:               req.Visibility,
		LearningOutcomes:         req.LearningOutcomes,
		Requirements:             req.Requirements,
		TargetAudience:           req.TargetAudience,
		EstimatedDurationMinutes: req.EstimatedDurationMinutes,
		ThumbnailURL:             req.ThumbnailURL,
	}

	course, err := h.service.UpdateCourse(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(course)
}

// SubmitCourse handles POST /v1/teacher/courses/{courseId}/submit
//
// @Summary      Submit course for review
// @Description  Submits a course draft for admin review; the course must be complete before submission
// @Tags         courses
// @Produce      json
// @Param        courseId  path  string  true  "Course ID"
// @Success      204  "No Content"
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/teacher/courses/{courseId}/submit [post]
func (h *CoursesHandler) SubmitCourse(w http.ResponseWriter, r *http.Request) {
	courseIDStr := r.PathValue("courseId")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	cmd := courses.SubmitCourseCommand{
		CourseID:  courseID,
		TeacherID: userID,
	}

	err = h.service.SubmitCourse(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteCourse handles DELETE /v1/teacher/courses/{courseId}
func (h *CoursesHandler) DeleteCourse(w http.ResponseWriter, r *http.Request) {
	courseID, err := uuid.Parse(r.PathValue("courseId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	if err := h.service.DeleteCourse(r.Context(), courses.DeleteCourseCommand{
		CourseID:  courseID,
		TeacherID: userID,
	}); err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetTeacherCoursePreview handles GET /v1/teacher/courses/{courseId}/preview
//
// @Summary      Get teacher course preview
// @Description  Returns the full course detail including unpublished content for the owning teacher
// @Tags         courses
// @Produce      json
// @Param        courseId  path  string  true  "Course ID"
// @Success      200  {object}  courses.CourseDetailResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/teacher/courses/{courseId}/preview [get]
func (h *CoursesHandler) GetTeacherCoursePreview(w http.ResponseWriter, r *http.Request) {
	courseIDStr := r.PathValue("courseId")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	detail, err := h.service.GetTeacherCoursePreview(r.Context(), courseID, userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

// Teacher content builder endpoints

// CreateModule handles POST /v1/teacher/courses/{courseId}/modules
func (h *CoursesHandler) CreateModule(w http.ResponseWriter, r *http.Request) {
	courseID, err := uuid.Parse(r.PathValue("courseId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Position    int    `json:"position"`
		IsFree      *bool  `json:"is_free"`
		IsPublished *bool  `json:"is_published"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	result, err := h.service.CreateModule(r.Context(), courses.CreateModuleCommand{
		CourseID:    courseID,
		TeacherID:   userID,
		Title:       req.Title,
		Description: req.Description,
		Position:    req.Position,
		IsFree:      boolDefault(req.IsFree, true),
		IsPublished: boolDefault(req.IsPublished, true),
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// UpdateModule handles PATCH /v1/teacher/modules/{moduleId}
func (h *CoursesHandler) UpdateModule(w http.ResponseWriter, r *http.Request) {
	moduleID, err := uuid.Parse(r.PathValue("moduleId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid module ID"))
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Position    int    `json:"position"`
		IsFree      *bool  `json:"is_free"`
		IsPublished *bool  `json:"is_published"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	result, err := h.service.UpdateModule(r.Context(), courses.UpdateModuleCommand{
		ModuleID:    moduleID,
		TeacherID:   userID,
		Title:       req.Title,
		Description: req.Description,
		Position:    req.Position,
		IsFree:      boolDefault(req.IsFree, true),
		IsPublished: boolDefault(req.IsPublished, true),
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// DeleteModule handles DELETE /v1/teacher/modules/{moduleId}
func (h *CoursesHandler) DeleteModule(w http.ResponseWriter, r *http.Request) {
	moduleID, err := uuid.Parse(r.PathValue("moduleId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid module ID"))
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	if err := h.service.DeleteModule(r.Context(), courses.DeleteModuleCommand{
		ModuleID:  moduleID,
		TeacherID: userID,
	}); err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateChapter handles POST /v1/teacher/modules/{moduleId}/chapters
func (h *CoursesHandler) CreateChapter(w http.ResponseWriter, r *http.Request) {
	moduleID, err := uuid.Parse(r.PathValue("moduleId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid module ID"))
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	var req struct {
		Title    string `json:"title"`
		Position int    `json:"position"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	result, err := h.service.CreateChapter(r.Context(), courses.CreateChapterCommand{
		ModuleID:  moduleID,
		TeacherID: userID,
		Title:     req.Title,
		Position:  req.Position,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// UpdateChapter handles PATCH /v1/teacher/chapters/{chapterId}
func (h *CoursesHandler) UpdateChapter(w http.ResponseWriter, r *http.Request) {
	chapterID, err := uuid.Parse(r.PathValue("chapterId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid chapter ID"))
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	var req struct {
		Title    string `json:"title"`
		Position int    `json:"position"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	result, err := h.service.UpdateChapter(r.Context(), courses.UpdateChapterCommand{
		ChapterID: chapterID,
		TeacherID: userID,
		Title:     req.Title,
		Position:  req.Position,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// DeleteChapter handles DELETE /v1/teacher/chapters/{chapterId}
func (h *CoursesHandler) DeleteChapter(w http.ResponseWriter, r *http.Request) {
	chapterID, err := uuid.Parse(r.PathValue("chapterId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid chapter ID"))
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	if err := h.service.DeleteChapter(r.Context(), courses.DeleteChapterCommand{
		ChapterID: chapterID,
		TeacherID: userID,
	}); err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateLesson handles POST /v1/teacher/chapters/{chapterId}/lessons
func (h *CoursesHandler) CreateLesson(w http.ResponseWriter, r *http.Request) {
	chapterID, err := uuid.Parse(r.PathValue("chapterId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid chapter ID"))
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	var req struct {
		Title           string  `json:"title"`
		Description     string  `json:"description"`
		Type            string  `json:"type"`
		VideoID         *string `json:"video_id"`
		DurationSeconds int     `json:"duration_seconds"`
		IsFreePreview   bool    `json:"is_free_preview"`
		IsFree          *bool   `json:"is_free"`
		IsDownloadable  bool    `json:"is_downloadable"`
		Position        int     `json:"position"`
		Status          string  `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	videoID, err := parseOptionalUUID(req.VideoID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	result, err := h.service.CreateLesson(r.Context(), courses.CreateLessonCommand{
		ChapterID:       chapterID,
		TeacherID:       userID,
		Title:           req.Title,
		Description:     req.Description,
		Type:            req.Type,
		VideoID:         videoID,
		DurationSeconds: req.DurationSeconds,
		IsFreePreview:   req.IsFreePreview,
		IsFree:          boolDefault(req.IsFree, true),
		IsDownloadable:  req.IsDownloadable,
		Position:        req.Position,
		Status:          req.Status,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// UpdateLesson handles PATCH /v1/teacher/lessons/{lessonId}
func (h *CoursesHandler) UpdateLesson(w http.ResponseWriter, r *http.Request) {
	lessonID, err := uuid.Parse(r.PathValue("lessonId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid lesson ID"))
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	var req struct {
		Title           string  `json:"title"`
		Description     string  `json:"description"`
		Type            string  `json:"type"`
		VideoID         *string `json:"video_id"`
		DurationSeconds int     `json:"duration_seconds"`
		IsFreePreview   bool    `json:"is_free_preview"`
		IsFree          *bool   `json:"is_free"`
		IsDownloadable  bool    `json:"is_downloadable"`
		Position        int     `json:"position"`
		Status          string  `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	videoID, err := parseOptionalUUID(req.VideoID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	result, err := h.service.UpdateLesson(r.Context(), courses.UpdateLessonCommand{
		LessonID:        lessonID,
		TeacherID:       userID,
		Title:           req.Title,
		Description:     req.Description,
		Type:            req.Type,
		VideoID:         videoID,
		DurationSeconds: req.DurationSeconds,
		IsFreePreview:   req.IsFreePreview,
		IsFree:          boolDefault(req.IsFree, true),
		IsDownloadable:  req.IsDownloadable,
		Position:        req.Position,
		Status:          req.Status,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// DeleteLesson handles DELETE /v1/teacher/lessons/{lessonId}
func (h *CoursesHandler) DeleteLesson(w http.ResponseWriter, r *http.Request) {
	lessonID, err := uuid.Parse(r.PathValue("lessonId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid lesson ID"))
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	if err := h.service.DeleteLesson(r.Context(), courses.DeleteLessonCommand{
		LessonID:  lessonID,
		TeacherID: userID,
	}); err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateCourseNote handles POST /v1/teacher/courses/{courseId}/notes
func (h *CoursesHandler) CreateCourseNote(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uuid.UUID)
	courseID, err := uuid.Parse(r.PathValue("courseId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}

	var req struct {
		ModuleID    string `json:"module_id"`
		LessonID    string `json:"lesson_id"`
		Title       string `json:"title"`
		Content     string `json:"content"`
		FileURL     string `json:"file_url"`
		IsFree      *bool  `json:"is_free"`
		IsPublished *bool  `json:"is_published"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}
	moduleID, err := optionalUUID(req.ModuleID)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_MODULE_ID", "invalid module ID"))
		return
	}
	lessonID, err := optionalUUID(req.LessonID)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_LESSON_ID", "invalid lesson ID"))
		return
	}

	result, err := h.service.CreateCourseNote(r.Context(), courses.CreateCourseNoteCommand{
		CourseID:    courseID,
		TeacherID:   userID,
		ModuleID:    moduleID,
		LessonID:    lessonID,
		Title:       req.Title,
		Content:     req.Content,
		FileURL:     req.FileURL,
		IsFree:      boolDefault(req.IsFree, true),
		IsPublished: boolDefault(req.IsPublished, false),
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusCreated, result)
}

// UpdateCourseNote handles PATCH /v1/teacher/notes/{noteId}
func (h *CoursesHandler) UpdateCourseNote(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uuid.UUID)
	noteID, err := uuid.Parse(r.PathValue("noteId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid note ID"))
		return
	}

	var req struct {
		ModuleID    string `json:"module_id"`
		LessonID    string `json:"lesson_id"`
		Title       string `json:"title"`
		Content     string `json:"content"`
		FileURL     string `json:"file_url"`
		IsFree      *bool  `json:"is_free"`
		IsPublished *bool  `json:"is_published"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}
	moduleID, err := optionalUUID(req.ModuleID)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_MODULE_ID", "invalid module ID"))
		return
	}
	lessonID, err := optionalUUID(req.LessonID)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_LESSON_ID", "invalid lesson ID"))
		return
	}

	result, err := h.service.UpdateCourseNote(r.Context(), courses.UpdateCourseNoteCommand{
		NoteID:      noteID,
		TeacherID:   userID,
		ModuleID:    moduleID,
		LessonID:    lessonID,
		Title:       req.Title,
		Content:     req.Content,
		FileURL:     req.FileURL,
		IsFree:      boolDefault(req.IsFree, true),
		IsPublished: boolDefault(req.IsPublished, false),
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// DeleteCourseNote handles DELETE /v1/teacher/notes/{noteId}
func (h *CoursesHandler) DeleteCourseNote(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(uuid.UUID)
	noteID, err := uuid.Parse(r.PathValue("noteId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid note ID"))
		return
	}
	if err := h.service.DeleteCourseNote(r.Context(), courses.DeleteCourseNoteCommand{NoteID: noteID, TeacherID: userID}); err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ReorderContent handles PATCH /v1/teacher/content/reorder
func (h *CoursesHandler) ReorderContent(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	var req struct {
		Type      string         `json:"type"`
		ParentID  string         `json:"parent_id"`
		Positions map[string]int `json:"positions"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	parentID, err := uuid.Parse(req.ParentID)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid parent ID"))
		return
	}

	positions, err := parseUUIDPositionMap(req.Positions)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	if err := h.service.ReorderContent(r.Context(), courses.ReorderContentCommand{
		TeacherID: userID,
		Type:      req.Type,
		ParentID:  parentID,
		Positions: positions,
	}); err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Admin endpoints

// ListPendingCourses handles GET /v1/admin/courses
func (h *CoursesHandler) ListPendingCourses(w http.ResponseWriter, r *http.Request) {
	params := pagination.ParseParams(r)
	filters := domainCourses.CourseFilters{
		Search: r.URL.Query().Get("search"),
		Status: domainCourses.CourseStatus(r.URL.Query().Get("status")),
	}
	if teacherIDRaw := strings.TrimSpace(r.URL.Query().Get("teacher_id")); teacherIDRaw != "" {
		teacherID, err := uuid.Parse(teacherIDRaw)
		if err != nil {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_TEACHER_ID", "invalid teacher ID"))
			return
		}
		filters.TeacherID = &teacherID
	}

	courseList, total, err := h.service.ListAdminCourses(r.Context(), filters, params.Page, params.Limit)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	meta := pagination.Meta{
		Page:       params.Page,
		Limit:      params.Limit,
		Total:      total,
		TotalPages: (total + params.Limit - 1) / params.Limit,
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"courses": courseList,
		"meta":    meta,
	})
}

// GetAdminCourseDetail handles GET /v1/admin/courses/{courseId}
func (h *CoursesHandler) GetAdminCourseDetail(w http.ResponseWriter, r *http.Request) {
	courseIDStr := r.PathValue("courseId")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}

	detail, err := h.service.GetAdminCourseDetail(r.Context(), courseID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, detail)
}

// ReviewCourse handles POST /v1/admin/courses/{courseId}/review
//
// @Summary      Review a course submission
// @Description  Approves or rejects a course that has been submitted for review by a teacher
// @Tags         courses
// @Accept       json
// @Produce      json
// @Param        courseId  path  string  true  "Course ID"
// @Param        body      body  object{action=string,comment=string}  true  "Review action: 'approve' or 'reject'"
// @Success      204  "No Content"
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/courses/{courseId}/review [post]
func (h *CoursesHandler) ReviewCourse(w http.ResponseWriter, r *http.Request) {
	courseIDStr := r.PathValue("courseId")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	var req struct {
		Action  string `json:"action"` // "approve" or "reject"
		Comment string `json:"comment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	switch req.Action {
	case "approve":
		cmd := courses.ApproveCourseCommand{
			CourseID: courseID,
			AdminID:  userID,
		}
		err = h.service.ApproveCourse(r.Context(), cmd)
	case "reject":
		cmd := courses.RejectCourseCommand{
			CourseID: courseID,
			AdminID:  userID,
			Comment:  req.Comment,
		}
		err = h.service.RejectCourse(r.Context(), cmd)
	default:
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ACTION", "action must be 'approve' or 'reject'"))
		return
	}

	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Upload endpoints

// UploadVideo handles POST /v1/uploads/video
//
// @Summary      Upload a video
// @Description  Uploads a video file for use in course lessons; returns a video ID that can be polled for processing status
// @Tags         uploads
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "Video file"
// @Success      201  {object}  courses.VideoStatusResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/uploads/video [post]
func (h *CoursesHandler) UploadVideo(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxVideoUploadBytes+1024*1024)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_MULTIPART", "invalid multipart form"))
		return
	}
	courseID, err := uuid.Parse(r.FormValue("course_id"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "valid course_id is required"))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("FILE_REQUIRED", "file is required"))
		return
	}
	defer file.Close()
	magic, err := readMagic(file)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_FILE", "could not prepare file for upload"))
		return
	}

	cmd := courses.UploadVideoCommand{
		CourseID:   courseID,
		UploaderID: userID,
		FileName:   header.Filename,
		FileSize:   header.Size,
		MimeType:   header.Header.Get("Content-Type"),
		MagicBytes: magic,
		Reader:     file,
	}

	result, err := h.service.UploadVideo(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// GetVideoStatus handles GET /v1/uploads/video/{videoId}/status
//
// @Summary      Get video processing status
// @Description  Returns the current processing status of an uploaded video (processing, ready, or failed)
// @Tags         uploads
// @Produce      json
// @Param        videoId  path  string  true  "Video ID"
// @Success      200  {object}  courses.VideoStatusResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/uploads/video/{videoId}/status [get]
func (h *CoursesHandler) GetVideoStatus(w http.ResponseWriter, r *http.Request) {
	videoIDStr := r.PathValue("videoId")
	videoID, err := uuid.Parse(videoIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid video ID"))
		return
	}

	status, err := h.service.GetVideoStatus(r.Context(), videoID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// InitDirectVideoUpload handles POST /v1/uploads/video/init
//
// @Summary      Initialize a direct-to-storage video upload
// @Description  Returns a presigned PUT URL the client can use to upload the video bytes directly to RustFS. After the PUT completes, the client must call POST /v1/uploads/video/{videoId}/complete to flip the status to "ready".
// @Tags         uploads
// @Accept       json
// @Produce      json
// @Param        request  body  InitDirectVideoUploadRequest  true  "Upload init"
// @Success      200  {object}  courses.DirectUploadResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/uploads/video/init [post]
type InitDirectVideoUploadRequest struct {
	CourseID string `json:"course_id"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
	MimeType string `json:"mime_type"`
	// MagicBytes is the first 512 bytes of the file, base64 encoded. The
	// server uses it to validate the file type and to refuse executables.
	MagicBytes string `json:"magic_b64"`
}

func (h *CoursesHandler) InitDirectVideoUpload(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}
	var req InitDirectVideoUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_BODY", "invalid JSON body"))
		return
	}
	courseID, err := uuid.Parse(req.CourseID)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "valid course_id is required"))
		return
	}
	magic, err := decodeMagicB64(req.MagicBytes)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	resp, err := h.service.InitDirectUpload(r.Context(), courses.InitDirectUploadCommand{
		CourseID:   courseID,
		UploaderID: userID,
		FileName:   req.FileName,
		FileSize:   req.FileSize,
		MimeType:   req.MimeType,
		MagicBytes: magic,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// decodeMagicB64 decodes the magic-bytes header sent by the client. The
// payload is the first ~512 bytes of the file, base64 encoded so the JSON
// body stays small.
func decodeMagicB64(s string) ([]byte, error) {
	if s == "" {
		return nil, apperrors.NewSimpleValidationError("MAGIC_REQUIRED", "magic_b64 is required")
	}
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, apperrors.NewSimpleValidationError("INVALID_MAGIC", "magic_b64 is not valid base64")
	}
	return raw, nil
}

// CompleteVideoUpload handles POST /v1/uploads/video/{videoId}/complete
//
// @Summary      Complete a direct-to-storage video upload
// @Description  Called by the client after a presigned PUT to RustFS finishes. Verifies the object landed, then flips the video status to "ready" so playback can start.
// @Tags         uploads
// @Produce      json
// @Param        videoId  path  string  true  "Video ID"
// @Success      200  {object}  courses.VideoStatusResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/uploads/video/{videoId}/complete [post]
func (h *CoursesHandler) CompleteVideoUpload(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}
	videoIDStr := r.PathValue("videoId")
	videoID, err := uuid.Parse(videoIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid video ID"))
		return
	}
	status, err := h.service.CompleteVideoUpload(r.Context(), courses.CompleteVideoUploadCommand{
		VideoID:  videoID,
		Uploader: userID,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// InitMultipartVideoUpload handles POST /v1/uploads/video/multipart/init
//
// @Summary      Initialize a resumable multipart video upload
// @Description  Starts an S3 multipart upload and returns an upload id the client uses to upload individual chunks. Each chunk is uploaded independently, so a page refresh can resume from the last completed part.
// @Tags         uploads
// @Accept       json
// @Produce      json
// @Param        request  body  InitMultipartUploadRequest  true  "Multipart init"
// @Success      200  {object}  courses.MultipartInitResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/uploads/video/multipart/init [post]
type InitMultipartUploadRequest struct {
	CourseID  string `json:"course_id"`
	FileName  string `json:"file_name"`
	FileSize  int64  `json:"file_size"`
	MimeType  string `json:"mime_type"`
	MagicB64  string `json:"magic_b64"`
	ChunkSize int64  `json:"chunk_size"`
}

func (h *CoursesHandler) InitMultipartVideoUpload(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}
	var req InitMultipartUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_BODY", "invalid JSON body"))
		return
	}
	courseID, err := uuid.Parse(req.CourseID)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "valid course_id is required"))
		return
	}
	magic, err := decodeMagicB64(req.MagicB64)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	resp, err := h.service.InitMultipartUpload(r.Context(), courses.InitMultipartUploadCommand{
		CourseID:   courseID,
		UploaderID: userID,
		FileName:   req.FileName,
		FileSize:   req.FileSize,
		MimeType:   req.MimeType,
		MagicBytes: magic,
		ChunkSize:  req.ChunkSize,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// PresignUploadPart handles POST /v1/uploads/video/multipart/{videoId}/part
//
// @Summary      Get a presigned URL for one upload chunk
// @Description  Returns a presigned PUT URL for a single chunk. The client uploads the chunk directly to RustFS and reports the resulting ETag back via the complete endpoint.
// @Tags         uploads
// @Accept       json
// @Produce      json
// @Param        videoId  path  string  true  "Video ID"
// @Param        request  body  PresignUploadPartRequest  true  "Part presign request"
// @Success      200  {object}  courses.PresignUploadPartResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/uploads/video/multipart/{videoId}/part [post]
type PresignUploadPartRequest struct {
	UploadID   string `json:"upload_id"`
	PartNumber int    `json:"part_number"`
}

func (h *CoursesHandler) PresignUploadPart(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}
	videoID, err := uuid.Parse(r.PathValue("videoId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid video ID"))
		return
	}
	var req PresignUploadPartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_BODY", "invalid JSON body"))
		return
	}
	resp, err := h.service.PresignUploadPart(r.Context(), courses.PresignUploadPartCommand{
		VideoID:    videoID,
		Uploader:   userID,
		UploadID:   req.UploadID,
		PartNumber: req.PartNumber,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// CompleteMultipartVideoUpload handles POST /v1/uploads/video/multipart/{videoId}/complete
//
// @Summary      Complete a multipart video upload
// @Description  Submits the list of completed parts in order. The server calls S3 CompleteMultipartUpload, then enqueues the transcode job. The video row flips to "ready" only after the HLS transcoding worker finishes.
// @Tags         uploads
// @Accept       json
// @Produce      json
// @Param        videoId  path  string  true  "Video ID"
// @Param        request  body  CompleteMultipartUploadRequest  true  "Complete multipart"
// @Success      200  {object}  courses.VideoStatusResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/uploads/video/multipart/{videoId}/complete [post]
type CompleteMultipartUploadRequest struct {
	UploadID string `json:"upload_id"`
	Parts    []struct {
		PartNumber int    `json:"part_number"`
		ETag       string `json:"etag"`
	} `json:"parts"`
}

func (h *CoursesHandler) CompleteMultipartVideoUpload(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}
	videoID, err := uuid.Parse(r.PathValue("videoId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid video ID"))
		return
	}
	var req CompleteMultipartUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_BODY", "invalid JSON body"))
		return
	}
	parts := make([]courses.CompletedPart, len(req.Parts))
	for i, p := range req.Parts {
		parts[i] = courses.CompletedPart{PartNumber: p.PartNumber, ETag: p.ETag}
	}
	status, err := h.service.CompleteMultipartUpload(r.Context(), courses.CompleteMultipartUploadCommand{
		VideoID:  videoID,
		Uploader: userID,
		UploadID: req.UploadID,
		Parts:    parts,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// AbortMultipartVideoUpload handles POST /v1/uploads/video/multipart/{videoId}/abort
//
// @Summary      Abort a multipart video upload
// @Description  Cancels an in-progress upload and marks the video row as failed. Safe to call multiple times.
// @Tags         uploads
// @Accept       json
// @Produce      json
// @Param        videoId  path  string  true  "Video ID"
// @Param        request  body  AbortMultipartUploadRequest  true  "Abort multipart"
// @Success      204
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/uploads/video/multipart/{videoId}/abort [post]
type AbortMultipartUploadRequest struct {
	UploadID string `json:"upload_id"`
}

func (h *CoursesHandler) AbortMultipartVideoUpload(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}
	videoID, err := uuid.Parse(r.PathValue("videoId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid video ID"))
		return
	}
	var req AbortMultipartUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_BODY", "invalid JSON body"))
		return
	}
	if err := h.service.AbortMultipartUpload(r.Context(), courses.AbortMultipartUploadCommand{
		VideoID:  videoID,
		Uploader: userID,
		UploadID: req.UploadID,
	}); err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UploadFile handles POST /v1/uploads/file
//
// @Summary      Upload a file
// @Description  Uploads a downloadable file (e.g. PDF) for use as a lesson attachment; returns a presigned download URL
// @Tags         uploads
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "File to upload"
// @Success      201  {object}  courses.FileUploadResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/uploads/file [post]
func (h *CoursesHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFileUploadBytes+1024*1024)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_MULTIPART", "invalid multipart form"))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("FILE_REQUIRED", "file is required"))
		return
	}
	defer file.Close()
	magic, err := readMagic(file)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_FILE", "could not prepare file for upload"))
		return
	}

	cmd := courses.UploadFileCommand{
		UploaderID: userID,
		FileName:   header.Filename,
		FileSize:   header.Size,
		MimeType:   header.Header.Get("Content-Type"),
		MagicBytes: magic,
		Reader:     file,
	}

	result, err := h.service.UploadFile(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

const (
	maxVideoUploadBytes int64 = 2 * 1024 * 1024 * 1024
	maxFileUploadBytes  int64 = 50 * 1024 * 1024
)

func readMagic(r io.Reader) ([]byte, error) {
	buf := make([]byte, 512)
	n, err := io.ReadFull(r, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, apperrors.NewSimpleValidationError("INVALID_FILE", "could not read file")
	}
	return buf[:n], nil
}

func parseOptionalUUID(value *string) (*uuid.UUID, error) {
	if value == nil || *value == "" {
		return nil, nil
	}

	parsed, err := uuid.Parse(*value)
	if err != nil {
		return nil, apperrors.NewSimpleValidationError("INVALID_ID", "invalid UUID")
	}

	return &parsed, nil
}

func parseUUIDPositionMap(raw map[string]int) (map[uuid.UUID]int, error) {
	positions := make(map[uuid.UUID]int, len(raw))
	for key, position := range raw {
		id, err := uuid.Parse(key)
		if err != nil {
			return nil, apperrors.NewSimpleValidationError("INVALID_ID", "invalid content ID in positions map")
		}
		positions[id] = position
	}

	return positions, nil
}
