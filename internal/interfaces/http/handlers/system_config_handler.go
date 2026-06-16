package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	appsysconfig "lms-backend/internal/application/system_config"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// SystemConfigHandler handles HTTP requests for the system config bounded context.
type SystemConfigHandler struct {
	service appsysconfig.Service
}

// NewSystemConfigHandler creates a new SystemConfigHandler.
func NewSystemConfigHandler(service appsysconfig.Service) *SystemConfigHandler {
	return &SystemConfigHandler{service: service}
}

// GetSettings handles GET /v1/admin/system/settings
// Returns the current platform settings. Requirements: 25.1
//
// @Summary      Get system settings
// @Description  Returns the current platform settings
// @Tags         system-config
// @Produce      json
// @Success      200  {object}  system_config.SettingsResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/system/settings [get]
func (h *SystemConfigHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.GetSettings(r.Context())
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// UpdateSettings handles PATCH /v1/admin/system/settings
// Applies a partial update to platform settings. Requirements: 25.2
//
// @Summary      Update system settings
// @Description  Applies a partial update to platform settings
// @Tags         system-config
// @Accept       json
// @Produce      json
// @Param        body  body  object  true  "Settings update data"
// @Success      200  {object}  system_config.SettingsResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/system/settings [patch]
func (h *SystemConfigHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	actorID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	actorName := getActorNameFromContext(r)

	var req struct {
		PlatformName          *string          `json:"platform_name"`
		DefaultTimezone       *string          `json:"default_timezone"`
		OAuthProvidersEnabled []string         `json:"oauth_providers_enabled"`
		MaintenanceMode       *bool            `json:"maintenance_mode"`
		FeatureFlags          *json.RawMessage `json:"feature_flags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("VALIDATION_ERROR", "invalid request body"))
		return
	}

	cmd := appsysconfig.UpdateSettingsCommand{
		ActorID:               actorID,
		ActorName:             actorName,
		IPAddress:             r.RemoteAddr,
		PlatformName:          req.PlatformName,
		DefaultTimezone:       req.DefaultTimezone,
		OAuthProvidersEnabled: req.OAuthProvidersEnabled,
		MaintenanceMode:       req.MaintenanceMode,
	}
	if req.FeatureFlags != nil {
		cmd.FeatureFlags = *req.FeatureFlags
	}

	result, err := h.service.UpdateSettings(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// GetSettingsHistory handles GET /v1/admin/system/settings/history
// Returns paginated settings change history. Requirements: 25.3
//
// @Summary      Get settings history
// @Description  Returns paginated settings change history
// @Tags         system-config
// @Produce      json
// @Param        page   query  int  false  "Page number"     default(1)
// @Param        limit  query  int  false  "Items per page"  default(20)
// @Success      200  {object}  system_config.SettingsHistoryResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/system/settings/history [get]
func (h *SystemConfigHandler) GetSettingsHistory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := 1
	limit := 20
	if p := q.Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	result, err := h.service.GetSettingsHistory(r.Context(), appsysconfig.GetSettingsHistoryCommand{
		Page:  page,
		Limit: limit,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// RollbackSettings handles POST /v1/admin/system/settings/rollback/:historyId
// Restores settings to a historical snapshot. Requirements: 25.4
//
// @Summary      Rollback settings
// @Description  Restores platform settings to a historical snapshot
// @Tags         system-config
// @Produce      json
// @Param        historyId  path  string  true  "History entry ID"
// @Success      200  {object}  system_config.RollbackResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/system/settings/rollback/{historyId} [post]
func (h *SystemConfigHandler) RollbackSettings(w http.ResponseWriter, r *http.Request) {
	actorID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	actorName := getActorNameFromContext(r)

	historyIDStr := r.PathValue("historyId")
	historyID, err := uuid.Parse(historyIDStr)
	if err != nil {
		writeErrorResponse(w, apperrors.NewValidationError([]map[string]string{
			{"field": "historyId", "message": "must be a valid UUID"},
		}))
		return
	}

	result, err := h.service.RollbackSettings(r.Context(), appsysconfig.RollbackSettingsCommand{
		ActorID:   actorID,
		ActorName: actorName,
		IPAddress: r.RemoteAddr,
		HistoryID: historyID,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}
