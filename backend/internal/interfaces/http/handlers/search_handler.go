package handlers

import (
	"net/http"
	"strconv"

	appsearch "lms-backend/internal/application/search"
	"lms-backend/pkg/apperrors"
)

// SearchHandler handles GET /v1/search.
type SearchHandler struct {
	service appsearch.Service
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(service appsearch.Service) *SearchHandler {
	return &SearchHandler{service: service}
}

// Search handles GET /v1/search
// Requirements: 26.1–26.5
//
// @Summary      Search
// @Description  Full-text search across courses, lessons, forum posts, and books; optionally uses auth context for enrollment-aware metadata
// @Tags         search
// @Produce      json
// @Param        q      query  string  true   "Search query (min 2 chars)"
// @Param        type   query  string  false  "Filter by type (courses, lessons, forum, books)"
// @Param        limit  query  int     false  "Max results per type"
// @Success      200  {object}  search.SearchResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Router       /v1/search [get]
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	query := q.Get("q")
	if len(query) < 2 {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("VALIDATION_ERROR", "query must be at least 2 characters"))
		return
	}

	limit := 10
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	cmd := appsearch.SearchCommand{
		Query: query,
		Type:  q.Get("type"),
		Limit: limit,
	}

	// Attach user ID if authenticated (optional — enrollment-aware metadata). Requirements: 26.4
	if userID, err := getUserIDFromContext(r); err == nil {
		cmd.UserID = &userID
	}

	result, err := h.service.Search(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}
