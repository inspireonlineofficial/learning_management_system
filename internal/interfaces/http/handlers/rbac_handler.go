package handlers

import (
	"encoding/json"
	"net/http"

	apprbac "lms-backend/internal/application/rbac"
	"lms-backend/pkg/apperrors"
)

// RBACHandler handles HTTP requests for RBAC management.
type RBACHandler struct {
	service apprbac.Service
}

// NewRBACHandler creates a new RBACHandler.
func NewRBACHandler(service apprbac.Service) *RBACHandler {
	return &RBACHandler{service: service}
}

// ListRoles handles GET /v1/admin/rbac/roles
// Returns all roles with their current permission sets. Requirements: 9.2
//
// @Summary      List roles
// @Description  Returns all roles with their current permission sets
// @Tags         rbac
// @Produce      json
// @Success      200  {object}  rbac.ListRolesResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/rbac/roles [get]
func (h *RBACHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.ListRoles(r.Context())
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// UpdateRolePermissions handles PATCH /v1/admin/rbac/roles/:roleId
// Replaces the permission set for a role. Requirements: 9.3, 9.4
//
// @Summary      Update role permissions
// @Description  Replaces the permission set for the specified role
// @Tags         rbac
// @Accept       json
// @Produce      json
// @Param        roleId  path  string  true  "Role ID"
// @Param        body    body  object  true  "Permissions payload"
// @Success      200  {object}  rbac.UpdateRolePermissionsResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/rbac/roles/{roleId} [patch]
func (h *RBACHandler) UpdateRolePermissions(w http.ResponseWriter, r *http.Request) {
	actorID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}
	actorName := getActorNameFromContext(r)
	roleID := r.PathValue("roleId")
	if roleID == "" {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("MISSING_PARAM", "roleId is required"))
		return
	}

	var req struct {
		Permissions []apprbac.PermissionInput `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("VALIDATION_ERROR", "invalid request body"))
		return
	}

	result, err := h.service.UpdateRolePermissions(r.Context(), apprbac.UpdateRolePermissionsCommand{
		ActorID:     actorID,
		ActorName:   actorName,
		IPAddress:   r.RemoteAddr,
		RoleID:      roleID,
		Permissions: req.Permissions,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}
