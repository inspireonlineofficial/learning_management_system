package handlers

import (
	"net/http"
	"strconv"
	"time"

	appaudit "lms-backend/internal/application/audit"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// AuditHandler handles HTTP requests for audit log queries.
type AuditHandler struct {
	service appaudit.Service
}

// NewAuditHandler creates a new AuditHandler.
func NewAuditHandler(service appaudit.Service) *AuditHandler {
	return &AuditHandler{service: service}
}

// ListAuditLogs handles GET /v1/admin/audit-logs
// Returns paginated, filterable audit log entries. Requirements: 9.5
//
// @Summary      List audit logs
// @Description  Returns paginated, filterable audit log entries
// @Tags         audit
// @Produce      json
// @Param        page         query  int     false  "Page number"    default(1)
// @Param        limit        query  int     false  "Items per page" default(20)
// @Param        actor_id     query  string  false  "Filter by actor UUID"
// @Param        action       query  string  false  "Filter by action name"
// @Param        target_type  query  string  false  "Filter by target type"
// @Param        target_id    query  string  false  "Filter by target UUID"
// @Param        from_date    query  string  false  "Start date (YYYY-MM-DD)"
// @Param        to_date      query  string  false  "End date (YYYY-MM-DD)"
// @Success      200  {object}  audit.ListAuditLogsResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/audit-logs [get]
func (h *AuditHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
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

	cmd := appaudit.ListAuditLogsCommand{Page: page, Limit: limit}

	if v := q.Get("actor_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_PARAM", "actor_id must be a valid UUID"))
			return
		}
		cmd.ActorID = &id
	}
	if v := q.Get("action"); v != "" {
		cmd.Action = &v
	}
	if v := q.Get("target_type"); v != "" {
		cmd.TargetType = &v
	}
	if v := q.Get("target_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_PARAM", "target_id must be a valid UUID"))
			return
		}
		cmd.TargetID = &id
	}
	if v := q.Get("from_date"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_DATE", "from_date must be YYYY-MM-DD"))
			return
		}
		cmd.FromDate = &t
	}
	if v := q.Get("to_date"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_DATE", "to_date must be YYYY-MM-DD"))
			return
		}
		cmd.ToDate = &t
	}

	result, err := h.service.ListAuditLogs(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}
