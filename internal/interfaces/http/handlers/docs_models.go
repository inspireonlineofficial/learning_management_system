package handlers

// ErrorResponse is the standard error envelope returned for 401, 403, 404, and 5xx responses.
// swagger:model ErrorResponse
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// ValidationErrorResponse is returned for HTTP 400 validation failures.
// swagger:model ValidationErrorResponse
type ValidationErrorResponse struct {
	Error struct {
		Code    string             `json:"code"`
		Message string             `json:"message"`
		Details []ValidationDetail `json:"details"`
	} `json:"error"`
}

// ValidationDetail describes a single field validation error.
// swagger:model ValidationDetail
type ValidationDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// PaginationMeta contains pagination metadata for list responses.
// swagger:model PaginationMeta
type PaginationMeta struct {
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
}
