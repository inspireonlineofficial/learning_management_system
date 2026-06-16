package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	appnotif "lms-backend/internal/application/notifications"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// NotificationsHandler handles HTTP requests for the notifications bounded context.
type NotificationsHandler struct {
	service appnotif.Service
}

// NewNotificationsHandler creates a new NotificationsHandler.
func NewNotificationsHandler(service appnotif.Service) *NotificationsHandler {
	return &NotificationsHandler{service: service}
}

// ─── Student / authenticated endpoints ───────────────────────────────────────

// ListNotifications handles GET /v1/notifications
// Returns a paginated list of notifications for the authenticated user.
// Requirements: 22.2
//
// @Summary      List notifications
// @Description  Returns a paginated list of notifications for the authenticated user
// @Tags         notifications
// @Produce      json
// @Param        page   query  int  false  "Page number"     default(1)
// @Param        limit  query  int  false  "Items per page"  default(20)
// @Success      200  {object}  notifications.NotificationListResponse
// @Failure      401  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/notifications [get]
func (h *NotificationsHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	q := r.URL.Query()
	page, limit := parseNotifPaginationParams(q)

	result, err := h.service.ListNotifications(r.Context(), appnotif.ListNotificationsCommand{
		UserID: userID,
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// MarkRead handles PATCH /v1/notifications/:notificationId/read
// Marks a single notification as read for the authenticated user.
// Requirements: 22.3
//
// @Summary      Mark notification as read
// @Description  Marks a single notification as read for the authenticated user
// @Tags         notifications
// @Produce      json
// @Param        notificationId  path  string  true  "Notification ID"
// @Success      204
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/notifications/{notificationId}/read [patch]
func (h *NotificationsHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	notifID, err := uuid.Parse(r.PathValue("notificationId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid notification ID"))
		return
	}

	if err := h.service.MarkRead(r.Context(), appnotif.MarkReadCommand{
		NotificationID: notifID,
		UserID:         userID,
	}); err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// MarkAllRead handles PATCH /v1/notifications/read-all
// Marks all notifications as read for the authenticated user.
// Requirements: 22.4
//
// @Summary      Mark all notifications as read
// @Description  Marks all notifications as read for the authenticated user
// @Tags         notifications
// @Produce      json
// @Success      200  {object}  notifications.MarkAllReadResponse
// @Failure      401  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/notifications/read-all [patch]
func (h *NotificationsHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	result, err := h.service.MarkAllRead(r.Context(), userID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// ─── Admin endpoints ──────────────────────────────────────────────────────────

// ListTemplates handles GET /v1/admin/notifications/templates
//
// @Summary      List notification templates
// @Description  Returns configurable notification templates
// @Tags         notifications
// @Produce      json
// @Success      200  {array}  notifications.TemplateResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/notifications/templates [get]
func (h *NotificationsHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.ListTemplates(r.Context())
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// UpdateTemplate handles PATCH /v1/admin/notifications/templates/:templateId
// Updates a notification template's subject and body.
// Requirements: 22.5
//
// @Summary      Update notification template
// @Description  Updates a notification template's subject and body
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Param        templateId  path  string  true  "Template ID"
// @Param        body        body  object  true  "Template update data"
// @Success      200  {object}  notifications.TemplateResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/notifications/templates/{templateId} [patch]
func (h *NotificationsHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	templateID, err := uuid.Parse(r.PathValue("templateId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid template ID"))
		return
	}

	var req struct {
		SubjectTemplate *string `json:"subject_template"`
		BodyTemplate    string  `json:"body_template"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_BODY", "invalid request body"))
		return
	}
	if req.BodyTemplate == "" {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("BODY_REQUIRED", "body_template is required"))
		return
	}

	result, err := h.service.UpdateTemplate(r.Context(), appnotif.UpdateTemplateCommand{
		TemplateID:      templateID,
		SubjectTemplate: req.SubjectTemplate,
		BodyTemplate:    req.BodyTemplate,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// SendBroadcast handles POST /v1/admin/notifications/broadcast
// Sends a broadcast notification to all users or a specific role.
// Requirements: 22.6
//
// @Summary      Send broadcast notification
// @Description  Sends a broadcast notification to all users or a specific role
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Param        body  body  object  true  "Broadcast data"
// @Success      200  {object}  notifications.BroadcastResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/notifications/broadcast [post]
func (h *NotificationsHandler) SendBroadcast(w http.ResponseWriter, r *http.Request) {
	actorID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	var req struct {
		TargetRole *string `json:"target_role"`
		Title      string  `json:"title"`
		Body       string  `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_BODY", "invalid request body"))
		return
	}
	if req.Title == "" || req.Body == "" {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("FIELDS_REQUIRED", "title and body are required"))
		return
	}

	result, err := h.service.SendBroadcast(r.Context(), appnotif.SendBroadcastCommand{
		ActorID:    actorID,
		ActorName:  getActorNameFromContext(r),
		TargetRole: req.TargetRole,
		Title:      req.Title,
		Body:       req.Body,
		IPAddress:  r.RemoteAddr,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// ListBroadcasts handles GET /v1/admin/notifications/broadcasts
// Returns persisted broadcast history from audit logs.
// Requirements: 22.6
func (h *NotificationsHandler) ListBroadcasts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, limit := parseNotifPaginationParams(q)

	result, err := h.service.ListBroadcasts(r.Context(), appnotif.ListBroadcastsCommand{
		Page:  page,
		Limit: limit,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func parseNotifPaginationParams(q interface{ Get(string) string }) (page, limit int) {
	page = 1
	limit = 20
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
	return
}
