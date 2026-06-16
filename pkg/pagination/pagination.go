package pagination

import (
	"net/http"
	"strconv"
)

// Params holds pagination parameters
type Params struct {
	Page  int
	Limit int
}

// Meta holds pagination metadata for responses
type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// ParseParams extracts pagination parameters from HTTP request
// Defaults: page=1, limit=20, max limit=100
func ParseParams(r *http.Request) Params {
	page := parseIntParam(r.URL.Query().Get("page"), 1)
	if page < 1 {
		page = 1
	}

	limit := parseIntParam(r.URL.Query().Get("limit"), 20)
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return Params{
		Page:  page,
		Limit: limit,
	}
}

// NewMeta creates pagination metadata
func NewMeta(total, page, limit int) Meta {
	totalPages := total / limit
	if total%limit > 0 {
		totalPages++
	}

	return Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}

func parseIntParam(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return val
}
