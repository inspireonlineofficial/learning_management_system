package middleware

import (
	"encoding/json"
	"lms-backend/pkg/apperrors"
	"net/http"
)

// writeError writes an error response in the standard format
func writeError(w http.ResponseWriter, appErr *apperrors.AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.HTTPStatus)

	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    appErr.Code,
			"message": appErr.Message,
		},
	}

	if len(appErr.Details) > 0 {
		response["error"].(map[string]interface{})["details"] = appErr.Details
	}

	json.NewEncoder(w).Encode(response)
}
