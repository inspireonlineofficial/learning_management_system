package handlers

import (
	"encoding/json"
	"fmt"
	"lms-backend/internal/application/users"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// UsersHandler handles user management HTTP requests
type UsersHandler struct {
	usersService users.Service
}

// NewUsersHandler creates a new users handler
func NewUsersHandler(usersService users.Service) *UsersHandler {
	return &UsersHandler{
		usersService: usersService,
	}
}

// SubmitStudentProfile handles PUT /v1/onboarding/student-profile
func (h *UsersHandler) SubmitStudentProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", nil)
		return
	}

	var req struct {
		SchoolName      string  `json:"school_name"`
		ClassGrade      string  `json:"class_grade"`
		RollNumber      string  `json:"roll_number"`
		DateOfBirth     string  `json:"date_of_birth"`
		Gender          *string `json:"gender,omitempty"`
		GuardianName    *string `json:"guardian_name,omitempty"`
		GuardianContact *string `json:"guardian_contact,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	// Parse date of birth
	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_DATE_FORMAT", "Date of birth must be in YYYY-MM-DD format", nil)
		return
	}

	cmd := users.SubmitStudentProfileCommand{
		SchoolName:      req.SchoolName,
		ClassGrade:      req.ClassGrade,
		RollNumber:      req.RollNumber,
		DateOfBirth:     dob,
		Gender:          req.Gender,
		GuardianName:    req.GuardianName,
		GuardianContact: req.GuardianContact,
	}

	result, err := h.usersService.SubmitStudentProfile(r.Context(), userID, cmd)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetStudentProfile handles GET /v1/onboarding/student-profile
func (h *UsersHandler) GetStudentProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", nil)
		return
	}

	result, err := h.usersService.GetStudentProfile(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ListUsers handles GET /v1/admin/users
func (h *UsersHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	filters := users.ListUsersFilters{
		Page:  1,
		Limit: 20,
	}

	if page := query.Get("page"); page != "" {
		var p int
		if _, err := fmt.Sscanf(page, "%d", &p); err == nil && p > 0 {
			filters.Page = p
		}
	}

	if limit := query.Get("limit"); limit != "" {
		var l int
		if _, err := fmt.Sscanf(limit, "%d", &l); err == nil && l > 0 && l <= 100 {
			filters.Limit = l
		}
	}

	if role := query.Get("role"); role != "" {
		filters.Role = &role
	}

	if status := query.Get("status"); status != "" {
		filters.Status = &status
	}

	if search := query.Get("search"); search != "" {
		filters.Search = &search
	}

	if fromDate := query.Get("from_date"); fromDate != "" {
		if t, err := time.Parse("2006-01-02", fromDate); err == nil {
			filters.FromDate = &t
		}
	}

	if toDate := query.Get("to_date"); toDate != "" {
		if t, err := time.Parse("2006-01-02", toDate); err == nil {
			filters.ToDate = &t
		}
	}

	result, err := h.usersService.ListUsers(r.Context(), filters)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// CreateUser handles POST /v1/admin/users
func (h *UsersHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	var req struct {
		FullName string `json:"full_name"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	cmd := users.CreateUserCommand{
		FullName: req.FullName,
		Email:    req.Email,
		Role:     req.Role,
	}

	result, err := h.usersService.CreateUser(r.Context(), cmd)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// GetUser handles GET /v1/admin/users/:userId
func (h *UsersHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// Extract user ID from path
	path := strings.TrimPrefix(r.URL.Path, "/v1/admin/users/")
	userIDStr := strings.Split(path, "/")[0]

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID", nil)
		return
	}

	result, err := h.usersService.GetUser(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// UpdateUser handles PATCH /v1/admin/users/:userId
func (h *UsersHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// Get actor ID from context
	actorID, err := getUserIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", nil)
		return
	}

	// Extract target user ID from path
	path := strings.TrimPrefix(r.URL.Path, "/v1/admin/users/")
	targetUserIDStr := strings.Split(path, "/")[0]

	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID", nil)
		return
	}

	var req struct {
		Role   *string `json:"role,omitempty"`
		Status *string `json:"status,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	cmd := users.UpdateUserCommand{
		Role:   req.Role,
		Status: req.Status,
	}

	result, err := h.usersService.UpdateUser(r.Context(), actorID, targetUserID, cmd)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// UpdateStudentProfile handles PATCH /v1/admin/users/:userId/student-profile
func (h *UsersHandler) UpdateStudentProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// Get actor ID from context
	actorID, err := getUserIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", nil)
		return
	}

	// Extract target user ID from path
	path := strings.TrimPrefix(r.URL.Path, "/v1/admin/users/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		writeError(w, http.StatusBadRequest, "INVALID_PATH", "Invalid path", nil)
		return
	}
	targetUserIDStr := parts[0]

	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID", nil)
		return
	}

	var req struct {
		SchoolName      *string `json:"school_name,omitempty"`
		ClassGrade      *string `json:"class_grade,omitempty"`
		RollNumber      *string `json:"roll_number,omitempty"`
		DateOfBirth     *string `json:"date_of_birth,omitempty"`
		Gender          *string `json:"gender,omitempty"`
		GuardianName    *string `json:"guardian_name,omitempty"`
		GuardianContact *string `json:"guardian_contact,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	cmd := users.UpdateStudentProfileCommand{
		SchoolName:      req.SchoolName,
		ClassGrade:      req.ClassGrade,
		RollNumber:      req.RollNumber,
		Gender:          req.Gender,
		GuardianName:    req.GuardianName,
		GuardianContact: req.GuardianContact,
	}

	// Parse date of birth if provided
	if req.DateOfBirth != nil {
		dob, err := time.Parse("2006-01-02", *req.DateOfBirth)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_DATE_FORMAT", "Date of birth must be in YYYY-MM-DD format", nil)
			return
		}
		cmd.DateOfBirth = &dob
	}

	if err := h.usersService.UpdateStudentProfile(r.Context(), actorID, targetUserID, cmd); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Student profile updated successfully"})
}

// ForcePasswordReset handles POST /v1/admin/users/:userId/force-password-reset
func (h *UsersHandler) ForcePasswordReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed", nil)
		return
	}

	// Get actor ID from context
	actorID, err := getUserIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", nil)
		return
	}

	// Extract target user ID from path
	path := strings.TrimPrefix(r.URL.Path, "/v1/admin/users/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		writeError(w, http.StatusBadRequest, "INVALID_PATH", "Invalid path", nil)
		return
	}
	targetUserIDStr := parts[0]

	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID", nil)
		return
	}

	if err := h.usersService.ForcePasswordReset(r.Context(), actorID, targetUserID); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Password reset email sent successfully"})
}
