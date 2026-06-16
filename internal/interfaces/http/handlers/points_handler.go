package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"lms-backend/internal/application/points"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// PointsHandler handles HTTP requests for the points & gamification engine.
type PointsHandler struct {
	service points.Service
}

// NewPointsHandler creates a new PointsHandler.
func NewPointsHandler(service points.Service) *PointsHandler {
	return &PointsHandler{service: service}
}

// GetStudentPoints handles GET /v1/student/points
//
// @Summary      Get student points summary
// @Description  Returns total_points, points_today, points_this_week, daily_breakdown_today, global_rank, and weekly_rank for the authenticated student
// @Tags         points
// @Produce      json
// @Success      200  {object}  points.StudentPointsResponse
// @Failure      401  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/student/points [get]
func (h *PointsHandler) GetStudentPoints(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	period := r.URL.Query().Get("period")
	if period != "7d" && period != "30d" {
		period = "30d"
	}

	cmd := points.GetStudentPointsCommand{
		StudentID: userID,
		Period:    period,
	}

	result, err := h.service.GetStudentPoints(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// GetPointsHistory handles GET /v1/student/points/history
//
// @Summary      Get points history
// @Description  Returns a paginated log of point-earning events for the authenticated student
// @Tags         points
// @Produce      json
// @Param        page   query  int  false  "Page number"  default(1)
// @Param        limit  query  int  false  "Items per page (max 100)"  default(20)  maximum(100)
// @Success      200  {object}  points.PointsHistoryResponse
// @Failure      401  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/student/points/history [get]
func (h *PointsHandler) GetPointsHistory(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
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

	cmd := points.GetPointsHistoryCommand{
		StudentID: userID,
		Page:      page,
		Limit:     limit,
	}

	result, err := h.service.GetPointsHistory(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// GetLeaderboard handles GET /v1/leaderboard
//
// @Summary      Get leaderboard
// @Description  Returns ranked leaderboard entries for the specified period. Opted-out students are shown as Anonymous to other users.
// @Tags         points
// @Produce      json
// @Param        period     query  string  false  "Leaderboard period: weekly or alltime"  Enums(weekly, alltime)  default(alltime)
// @Param        page       query  int     false  "Page number"  default(1)
// @Param        limit      query  int     false  "Items per page (max 100)"  default(20)  maximum(100)
// @Param        course_id  query  string  false  "Filter by course ID (UUID)"
// @Success      200  {object}  points.LeaderboardResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/leaderboard [get]
func (h *PointsHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	period := r.URL.Query().Get("period")
	if period != "weekly" && period != "alltime" {
		period = "alltime"
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	var courseID *uuid.UUID
	if cid := r.URL.Query().Get("course_id"); cid != "" {
		parsed, err := uuid.Parse(cid)
		if err != nil {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid course_id"))
			return
		}
		courseID = &parsed
	}

	cmd := points.GetLeaderboardCommand{
		RequesterID: userID,
		Period:      period,
		Limit:       limit,
		CourseID:    courseID,
	}

	result, err := h.service.GetLeaderboard(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// UpdatePointsConfig handles PATCH /v1/admin/points/config
//
// @Summary      Update points configuration
// @Description  Admin updates the platform-wide points configuration for video completions, quiz passes, and perfect score bonuses
// @Tags         points
// @Accept       json
// @Produce      json
// @Param        body  body  object{points_per_video=int,points_per_quiz_pass=int,bonus_points_perfect_score=int}  false  "Points configuration update"
// @Success      200  {object}  points.PointsConfigResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/points/config [patch]
func (h *PointsHandler) UpdatePointsConfig(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	var req struct {
		PointsPerVideo          *int `json:"points_per_video"`
		PointsPerQuizPass       *int `json:"points_per_quiz_pass"`
		BonusPointsPerfectScore *int `json:"bonus_points_perfect_score"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	// Extract actor name from context (set by auth middleware)
	actorName, _ := r.Context().Value("user_name").(string)
	ipAddress := r.RemoteAddr

	cmd := points.UpdatePointsConfigCommand{
		ActorID:                 userID,
		ActorName:               actorName,
		PointsPerVideo:          req.PointsPerVideo,
		PointsPerQuizPass:       req.PointsPerQuizPass,
		BonusPointsPerfectScore: req.BonusPointsPerfectScore,
		IPAddress:               ipAddress,
	}

	result, err := h.service.UpdatePointsConfig(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// GetPointsConfig handles GET /v1/admin/points/config.
func (h *PointsHandler) GetPointsConfig(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.GetPointsConfig(r.Context())
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// ToggleLeaderboardOptOut handles PATCH /v1/student/leaderboard/opt-out
//
// @Summary      Toggle leaderboard opt-out
// @Description  Allows the authenticated student to opt in or out of the public leaderboard
// @Tags         points
// @Accept       json
// @Produce      json
// @Param        body  body  object{opt_out=bool}  true  "Opt-out toggle request"
// @Success      200  {object}  points.ToggleLeaderboardOptOutResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/student/leaderboard/opt-out [patch]
func (h *PointsHandler) ToggleLeaderboardOptOut(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	var req struct {
		OptOut bool `json:"opt_out"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	cmd := points.ToggleLeaderboardOptOutCommand{
		StudentID: userID,
		OptOut:    req.OptOut,
	}

	result, err := h.service.ToggleLeaderboardOptOut(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}
