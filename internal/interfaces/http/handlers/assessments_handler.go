package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"lms-backend/internal/application/assessments"
	"lms-backend/pkg/apperrors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

// Service is an alias for the assessments service interface
type Service = assessments.Service

// AssessmentsHandler handles HTTP requests for assessments
type AssessmentsHandler struct {
	service Service
}

const (
	maxAssignmentFiles              = 5
	maxAssignmentFileBytes    int64 = 50 * 1024 * 1024
	maxAssignmentRequestBytes       = maxAssignmentFiles*maxAssignmentFileBytes + 1024*1024
)

// NewAssessmentsHandler creates a new assessments handler
func NewAssessmentsHandler(service Service) *AssessmentsHandler {
	return &AssessmentsHandler{
		service: service,
	}
}

// CreateQuiz handles POST /v1/teacher/courses/:courseId/quizzes
//
// @Summary      Create a quiz
// @Description  Creates a new quiz for the specified course
// @Tags         assessments
// @Accept       json
// @Produce      json
// @Param        courseId  path      string                           true  "Course ID"
// @Param        body      body      assessments.CreateQuizCommand    true  "Quiz creation request"
// @Success      201  {object}  assessments.QuizTeacherResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/teacher/courses/{courseId}/quizzes [post]
func (h *AssessmentsHandler) CreateQuiz(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("user_id").(uuid.UUID)

	// Extract courseId from URL
	courseIDStr := r.PathValue("courseId")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}

	// Parse request body
	var req struct {
		LessonID                   *string                             `json:"lesson_id"`
		Title                      string                              `json:"title"`
		TimeLimitSeconds           int                                 `json:"time_limit_seconds"`
		MaxAttempts                int                                 `json:"max_attempts"`
		PassingScorePercent        float64                             `json:"passing_score_percent"`
		ShuffleQuestions           bool                                `json:"shuffle_questions"`
		ShowAnswersAfterSubmission bool                                `json:"show_answers_after_submission"`
		Questions                  []assessments.CreateQuestionCommand `json:"questions"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	// Parse optional lesson_id
	var lessonID *uuid.UUID
	if req.LessonID != nil && *req.LessonID != "" {
		parsed, err := uuid.Parse(*req.LessonID)
		if err != nil {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid lesson ID"))
			return
		}
		lessonID = &parsed
	}

	// Build command
	cmd := assessments.CreateQuizCommand{
		CourseID:                   courseID,
		TeacherID:                  userID,
		LessonID:                   lessonID,
		Title:                      req.Title,
		TimeLimitSeconds:           req.TimeLimitSeconds,
		MaxAttempts:                req.MaxAttempts,
		PassingScorePercent:        req.PassingScorePercent,
		ShuffleQuestions:           req.ShuffleQuestions,
		ShowAnswersAfterSubmission: req.ShowAnswersAfterSubmission,
		Questions:                  req.Questions,
	}

	// Execute use case
	result, err := h.service.CreateQuiz(ctx, cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, result)
}

// GetTeacherQuizzes handles GET /v1/teacher/courses/:courseId/quizzes
//
// @Summary      List teacher quizzes
// @Description  Returns all quizzes for the specified course (teacher view with questions)
// @Tags         assessments
// @Produce      json
// @Param        courseId  path      string  true  "Course ID"
// @Success      200  {array}   assessments.QuizTeacherResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/teacher/courses/{courseId}/quizzes [get]
func (h *AssessmentsHandler) GetTeacherQuizzes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("user_id").(uuid.UUID)

	// Extract courseId from URL
	courseIDStr := r.PathValue("courseId")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}

	// Execute use case
	result, err := h.service.GetTeacherQuizzes(ctx, courseID, userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// GetTeacherQuiz handles GET /v1/teacher/quizzes/:quizId.
func (h *AssessmentsHandler) GetTeacherQuiz(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	quizID, err := uuid.Parse(r.PathValue("quizId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid quiz ID"))
		return
	}
	result, err := h.service.GetTeacherQuiz(r.Context(), quizID, userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// UpdateTeacherQuiz handles PATCH /v1/teacher/quizzes/:quizId.
func (h *AssessmentsHandler) UpdateTeacherQuiz(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	quizID, err := uuid.Parse(r.PathValue("quizId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid quiz ID"))
		return
	}

	var req struct {
		LessonID                   *string                             `json:"lesson_id"`
		Title                      string                              `json:"title"`
		TimeLimitSeconds           int                                 `json:"time_limit_seconds"`
		MaxAttempts                int                                 `json:"max_attempts"`
		PassingScorePercent        float64                             `json:"passing_score_percent"`
		ShuffleQuestions           bool                                `json:"shuffle_questions"`
		ShowAnswersAfterSubmission bool                                `json:"show_answers_after_submission"`
		Questions                  []assessments.CreateQuestionCommand `json:"questions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	var lessonID *uuid.UUID
	if req.LessonID != nil && *req.LessonID != "" {
		parsed, err := uuid.Parse(*req.LessonID)
		if err != nil {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid lesson ID"))
			return
		}
		lessonID = &parsed
	}

	result, err := h.service.UpdateQuiz(r.Context(), assessments.UpdateQuizCommand{
		QuizID:                     quizID,
		TeacherID:                  userID,
		LessonID:                   lessonID,
		Title:                      req.Title,
		TimeLimitSeconds:           req.TimeLimitSeconds,
		MaxAttempts:                req.MaxAttempts,
		PassingScorePercent:        req.PassingScorePercent,
		ShuffleQuestions:           req.ShuffleQuestions,
		ShowAnswersAfterSubmission: req.ShowAnswersAfterSubmission,
		Questions:                  req.Questions,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// DeleteTeacherQuiz handles DELETE /v1/teacher/quizzes/:quizId.
func (h *AssessmentsHandler) DeleteTeacherQuiz(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	quizID, err := uuid.Parse(r.PathValue("quizId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid quiz ID"))
		return
	}
	if err := h.service.DeleteQuiz(r.Context(), quizID, userID); err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// CreateTeacherQuestion handles POST /v1/teacher/quizzes/:quizId/questions.
func (h *AssessmentsHandler) CreateTeacherQuestion(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	quizID, err := uuid.Parse(r.PathValue("quizId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid quiz ID"))
		return
	}
	question, err := decodeTeacherQuestionRequest(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	result, err := h.service.CreateQuestion(r.Context(), assessments.CreateStandaloneQuestionCommand{
		QuizID:    quizID,
		TeacherID: userID,
		Question:  question,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusCreated, result)
}

// UpdateTeacherQuestion handles PATCH /v1/teacher/questions/:questionId.
func (h *AssessmentsHandler) UpdateTeacherQuestion(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	questionID, err := uuid.Parse(r.PathValue("questionId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid question ID"))
		return
	}
	question, err := decodeTeacherQuestionRequest(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	result, err := h.service.UpdateQuestion(r.Context(), assessments.UpdateQuestionCommand{
		QuestionID: questionID,
		TeacherID:  userID,
		Question:   question,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// DeleteTeacherQuestion handles DELETE /v1/teacher/questions/:questionId.
func (h *AssessmentsHandler) DeleteTeacherQuestion(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	questionID, err := uuid.Parse(r.PathValue("questionId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid question ID"))
		return
	}
	if err := h.service.DeleteQuestion(r.Context(), questionID, userID); err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

// ListStudentQuizzes handles GET /v1/student/assessments.
func (h *AssessmentsHandler) ListStudentQuizzes(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	result, err := h.service.ListStudentQuizzes(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"quizzes": result,
	})
}

// GetStudentQuizDetail handles GET /v1/student/assessments/:quizId.
func (h *AssessmentsHandler) GetStudentQuizDetail(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	quizID, err := uuid.Parse(r.PathValue("quizId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid quiz ID"))
		return
	}

	result, err := h.service.GetStudentQuizDetail(r.Context(), quizID, userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// GetStudentQuizAttemptResult handles GET /v1/student/assessments/:quizId/attempts/:attemptId.
func (h *AssessmentsHandler) GetStudentQuizAttemptResult(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	quizID, err := uuid.Parse(r.PathValue("quizId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid quiz ID"))
		return
	}

	attemptID, err := uuid.Parse(r.PathValue("attemptId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid attempt ID"))
		return
	}

	result, err := h.service.GetStudentQuizAttemptResult(r.Context(), quizID, attemptID, userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// GetStudentAttempt handles GET /v1/student/assessments/attempts/:attemptId.
func (h *AssessmentsHandler) GetStudentAttempt(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	attemptID, err := uuid.Parse(r.PathValue("attemptId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid attempt ID"))
		return
	}

	result, err := h.service.GetStudentAttempt(r.Context(), attemptID, userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// CreateAssignment handles POST /v1/teacher/courses/:courseId/assignments
//
// @Summary      Create an assignment
// @Description  Creates a new assignment for the specified course
// @Tags         assessments
// @Accept       json
// @Produce      json
// @Param        courseId  path      string                                 true  "Course ID"
// @Param        body      body      assessments.CreateAssignmentCommand    true  "Assignment creation request"
// @Success      201  {object}  assessments.AssignmentResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/teacher/courses/{courseId}/assignments [post]
func (h *AssessmentsHandler) CreateAssignment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("user_id").(uuid.UUID)

	// Extract courseId from URL
	courseIDStr := r.PathValue("courseId")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course ID"))
		return
	}

	// Parse request body
	var req struct {
		Title               string  `json:"title"`
		Description         string  `json:"description"`
		DueAt               string  `json:"due_at"`
		SubmissionType      string  `json:"submission_type"`
		MaxFileSizeMB       int     `json:"max_file_size_mb"`
		AllowLateSubmission bool    `json:"allow_late_submission"`
		TotalMarks          float64 `json:"total_marks"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	// Build command
	cmd := assessments.CreateAssignmentCommand{
		CourseID:            courseID,
		TeacherID:           userID,
		Title:               req.Title,
		Description:         req.Description,
		DueAt:               req.DueAt,
		SubmissionType:      req.SubmissionType,
		MaxFileSizeMB:       req.MaxFileSizeMB,
		AllowLateSubmission: req.AllowLateSubmission,
		TotalMarks:          req.TotalMarks,
	}

	// Execute use case
	result, err := h.service.CreateAssignment(ctx, cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, result)
}

// ListTeacherCourseAssignments handles GET /v1/teacher/courses/:courseId/assignments.
func (h *AssessmentsHandler) ListTeacherCourseAssignments(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.service.ListTeacherCourseAssignments(r.Context(), courseID, userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"assignments": result,
	})
}

// GetTeacherAssignment handles GET /v1/teacher/assignments/:assignmentId.
func (h *AssessmentsHandler) GetTeacherAssignment(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	assignmentID, err := uuid.Parse(r.PathValue("assignmentId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid assignment ID"))
		return
	}

	result, err := h.service.GetTeacherAssignment(r.Context(), assignmentID, userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// UpdateTeacherAssignment handles PATCH /v1/teacher/assignments/:assignmentId.
func (h *AssessmentsHandler) UpdateTeacherAssignment(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	assignmentID, err := uuid.Parse(r.PathValue("assignmentId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid assignment ID"))
		return
	}

	var req struct {
		Title               string  `json:"title"`
		Description         string  `json:"description"`
		DueAt               string  `json:"due_at"`
		SubmissionType      string  `json:"submission_type"`
		MaxFileSizeMB       int     `json:"max_file_size_mb"`
		AllowLateSubmission bool    `json:"allow_late_submission"`
		TotalMarks          float64 `json:"total_marks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	result, err := h.service.UpdateAssignment(r.Context(), assessments.UpdateAssignmentCommand{
		AssignmentID:        assignmentID,
		TeacherID:           userID,
		Title:               req.Title,
		Description:         req.Description,
		DueAt:               req.DueAt,
		SubmissionType:      req.SubmissionType,
		MaxFileSizeMB:       req.MaxFileSizeMB,
		AllowLateSubmission: req.AllowLateSubmission,
		TotalMarks:          req.TotalMarks,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// ListStudentAssignments handles GET /v1/student/assignments.
func (h *AssessmentsHandler) ListStudentAssignments(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	result, err := h.service.ListStudentAssignments(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"assignments": result,
	})
}

// GetStudentAssignmentDetail handles GET /v1/student/assignments/:assignmentId.
func (h *AssessmentsHandler) GetStudentAssignmentDetail(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	assignmentID, err := uuid.Parse(r.PathValue("assignmentId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid assignment ID"))
		return
	}

	result, err := h.service.GetStudentAssignmentDetail(r.Context(), assignmentID, userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// StartQuizAttempt handles POST /v1/quizzes/:quizId/attempts
//
// @Summary      Start a quiz attempt
// @Description  Starts a new quiz attempt for the authenticated student
// @Tags         assessments
// @Accept       json
// @Produce      json
// @Param        quizId  path      string  true  "Quiz ID"
// @Success      201  {object}  assessments.QuizAttemptResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/quizzes/{quizId}/attempts [post]
func (h *AssessmentsHandler) StartQuizAttempt(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("user_id").(uuid.UUID)

	// Extract quizId from URL
	quizIDStr := r.PathValue("quizId")
	quizID, err := uuid.Parse(quizIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid quiz ID"))
		return
	}

	// Build command
	cmd := assessments.StartAttemptCommand{
		QuizID:    quizID,
		StudentID: userID,
	}

	// Execute use case
	result, err := h.service.StartAttempt(ctx, cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, result)
}

// SubmitQuizAttempt handles POST /v1/quizzes/:quizId/attempts/:attemptId/submit
//
// @Summary      Submit a quiz attempt
// @Description  Submits answers for an in-progress quiz attempt and returns the scored result
// @Tags         assessments
// @Accept       json
// @Produce      json
// @Param        quizId     path      string                              true  "Quiz ID"
// @Param        attemptId  path      string                              true  "Attempt ID"
// @Param        body       body      assessments.SubmitAttemptCommand    true  "Quiz answers"
// @Success      200  {object}  assessments.SubmitAttemptResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/quizzes/{quizId}/attempts/{attemptId}/submit [post]
func (h *AssessmentsHandler) SubmitQuizAttempt(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("user_id").(uuid.UUID)

	// Extract quizId and attemptId from URL
	quizIDStr := r.PathValue("quizId")
	_, err := uuid.Parse(quizIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid quiz ID"))
		return
	}

	attemptIDStr := r.PathValue("attemptId")
	attemptID, err := uuid.Parse(attemptIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid attempt ID"))
		return
	}

	// Parse request body
	var req struct {
		Answers []assessments.QuizAnswerCommand `json:"answers"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	// Build command
	cmd := assessments.SubmitAttemptCommand{
		AttemptID: attemptID,
		StudentID: userID,
		Answers:   req.Answers,
	}

	// Execute use case
	result, err := h.service.SubmitAttempt(ctx, cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// SaveQuizAttemptAnswers handles PATCH /v1/quizzes/:quizId/attempts/:attemptId.
func (h *AssessmentsHandler) SaveQuizAttemptAnswers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("user_id").(uuid.UUID)

	if _, err := uuid.Parse(r.PathValue("quizId")); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid quiz ID"))
		return
	}
	attemptID, err := uuid.Parse(r.PathValue("attemptId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid attempt ID"))
		return
	}

	var req struct {
		Answers map[string]interface{} `json:"answers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}
	if req.Answers == nil {
		req.Answers = map[string]interface{}{}
	}

	result, err := h.service.SaveAttemptAnswers(ctx, assessments.SaveAttemptAnswersCommand{
		AttemptID: attemptID,
		StudentID: userID,
		Answers:   req.Answers,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// SubmitAssignment handles POST /v1/assignments/:assignmentId/submissions
//
// @Summary      Submit an assignment
// @Description  Submits text content and/or files for the specified assignment
// @Tags         assessments
// @Accept       multipart/form-data
// @Produce      json
// @Param        assignmentId  path       string  true   "Assignment ID"
// @Param        text_content  formData   string  false  "Text content of the submission"
// @Param        is_draft      formData   string  false  "Submit as draft (true/false)"
// @Param        files         formData   file    false  "Submission files (max 5)"
// @Success      201  {object}  assessments.AssignmentSubmissionResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/assignments/{assignmentId}/submissions [post]
func (h *AssessmentsHandler) SubmitAssignment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("user_id").(uuid.UUID)

	// Extract assignmentId from URL
	assignmentIDStr := r.PathValue("assignmentId")
	assignmentID, err := uuid.Parse(assignmentIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid assignment ID"))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAssignmentRequestBytes)

	// Parse multipart form (for file uploads)
	err = r.ParseMultipartForm(8 << 20)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_FORM", "failed to parse form data"))
		return
	}

	// Extract text content
	textContent := r.FormValue("text_content")
	isDraftStr := r.FormValue("is_draft")
	isDraft := isDraftStr == "true"

	// Extract files
	var files []assessments.SubmissionFileCommand
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		fileHeaders := r.MultipartForm.File["files"]
		if len(fileHeaders) > maxAssignmentFiles {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("TOO_MANY_FILES", "maximum 5 files allowed"))
			return
		}

		for _, fileHeader := range fileHeaders {
			if fileHeader.Size <= 0 || fileHeader.Size > maxAssignmentFileBytes {
				writeErrorResponse(w, apperrors.NewSimpleValidationError("FILE_TOO_LARGE", "each file must be between 1 byte and 50 MB"))
				return
			}

			file, err := fileHeader.Open()
			if err != nil {
				writeErrorResponse(w, apperrors.NewSimpleValidationError("FILE_READ_ERROR", "failed to read file"))
				return
			}
			defer file.Close()

			content, err := io.ReadAll(file)
			if err != nil {
				writeErrorResponse(w, apperrors.NewSimpleValidationError("FILE_READ_ERROR", "failed to read file content"))
				return
			}
			if !isAllowedAssignmentFile(fileHeader.Header.Get("Content-Type"), content) {
				writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_FILE_TYPE", "file type is not allowed"))
				return
			}

			files = append(files, assessments.SubmissionFileCommand{
				OriginalFilename: fileHeader.Filename,
				MimeType:         http.DetectContentType(firstBytes(content, 512)),
				SizeBytes:        fileHeader.Size,
				Content:          content,
			})
		}
	}

	// Build command
	cmd := assessments.SubmitAssignmentCommand{
		AssignmentID: assignmentID,
		StudentID:    userID,
		TextContent:  textContent,
		Files:        files,
		IsDraft:      isDraft,
	}

	// Execute use case
	result, err := h.service.SubmitAssignment(ctx, cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, result)
}

func isAllowedAssignmentFile(declared string, content []byte) bool {
	detected := http.DetectContentType(firstBytes(content, 512))
	allowed := map[string]bool{
		"application/pdf":           true,
		"image/jpeg":                true,
		"image/png":                 true,
		"image/webp":                true,
		"text/plain":                true,
		"text/plain; charset=utf-8": true,
	}
	if !allowed[detected] {
		return false
	}
	return declared == "" ||
		declared == "application/octet-stream" ||
		declared == detected ||
		bytes.HasPrefix([]byte(declared), []byte(detected))
}

func firstBytes(content []byte, max int) []byte {
	if len(content) <= max {
		return content
	}
	return content[:max]
}

// ListTeacherAssignmentSubmissions handles GET /v1/teacher/assignments/:assignmentId/submissions.
func (h *AssessmentsHandler) ListTeacherAssignmentSubmissions(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	assignmentID, err := uuid.Parse(r.PathValue("assignmentId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid assignment ID"))
		return
	}

	page := 1
	limit := 20
	if raw := r.URL.Query().Get("page"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	result, err := h.service.ListTeacherAssignmentSubmissions(r.Context(), assignmentID, userID, page, limit)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// GetTeacherAssignmentSubmission handles GET /v1/teacher/assignments/:assignmentId/submissions/:submissionId.
func (h *AssessmentsHandler) GetTeacherAssignmentSubmission(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	assignmentID, err := uuid.Parse(r.PathValue("assignmentId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid assignment ID"))
		return
	}

	submissionID, err := uuid.Parse(r.PathValue("submissionId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid submission ID"))
		return
	}

	result, err := h.service.GetTeacherAssignmentSubmission(r.Context(), assignmentID, submissionID, userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// GradeSubmission handles POST /v1/teacher/assignments/:assignmentId/submissions/:submissionId/grade
//
// @Summary      Grade an assignment submission
// @Description  Records a grade and optional feedback for a student's assignment submission
// @Tags         assessments
// @Accept       json
// @Produce      json
// @Param        assignmentId  path      string                               true  "Assignment ID"
// @Param        submissionId  path      string                               true  "Submission ID"
// @Param        body          body      assessments.GradeSubmissionCommand   true  "Grade request"
// @Success      200  {object}  assessments.AssignmentSubmissionResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/teacher/assignments/{assignmentId}/submissions/{submissionId}/grade [post]
func (h *AssessmentsHandler) GradeSubmission(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("user_id").(uuid.UUID)

	// Extract assignmentId and submissionId from URL
	assignmentIDStr := r.PathValue("assignmentId")
	_, err := uuid.Parse(assignmentIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid assignment ID"))
		return
	}

	submissionIDStr := r.PathValue("submissionId")
	submissionID, err := uuid.Parse(submissionIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid submission ID"))
		return
	}

	// Parse request body
	var req struct {
		Score             float64 `json:"score"`
		Feedback          string  `json:"feedback"`
		RevisionRequested bool    `json:"revision_requested"`
		RevisionNotes     string  `json:"revision_notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	// Build command
	cmd := assessments.GradeSubmissionCommand{
		SubmissionID:      submissionID,
		GradedBy:          userID,
		Score:             req.Score,
		Feedback:          req.Feedback,
		RevisionRequested: req.RevisionRequested,
		RevisionNotes:     req.RevisionNotes,
	}

	// Execute use case
	result, err := h.service.GradeSubmission(ctx, cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

func decodeTeacherQuestionRequest(r *http.Request) (assessments.CreateQuestionCommand, error) {
	var req struct {
		Body        string `json:"body"`
		Prompt      string `json:"prompt"`
		Type        string `json:"type"`
		Position    int    `json:"position"`
		Explanation string `json:"explanation"`
		Options     []struct {
			Body      string `json:"body"`
			Text      string `json:"text"`
			IsCorrect bool   `json:"is_correct"`
			Position  int    `json:"position"`
		} `json:"options"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return assessments.CreateQuestionCommand{}, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body")
	}

	body := req.Body
	if body == "" {
		body = req.Prompt
	}
	questionType := req.Type
	switch questionType {
	case "single_choice":
		questionType = "single"
	case "multi_select":
		questionType = "multiple"
	}

	options := make([]assessments.CreateQuestionOptionCommand, 0, len(req.Options))
	for i, opt := range req.Options {
		optionBody := opt.Body
		if optionBody == "" {
			optionBody = opt.Text
		}
		position := opt.Position
		if position <= 0 {
			position = i + 1
		}
		options = append(options, assessments.CreateQuestionOptionCommand{
			Body:      optionBody,
			IsCorrect: opt.IsCorrect,
			Position:  position,
		})
	}

	return assessments.CreateQuestionCommand{
		Body:        body,
		Type:        questionType,
		Position:    req.Position,
		Explanation: req.Explanation,
		Options:     options,
	}, nil
}
