package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
)

// MaintenanceModeChecker is a function that returns true when maintenance mode is active.
type MaintenanceModeChecker func() bool

// MaintenanceMode middleware returns HTTP 503 for all non-admin endpoints when
// maintenance mode is enabled. Admin endpoints (/v1/admin/*) are always allowed through.
// Requirements: 25.5
func MaintenanceMode(checker MaintenanceModeChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !checker() {
				next.ServeHTTP(w, r)
				return
			}

			// Admin endpoints are exempt from maintenance mode
			if strings.HasPrefix(r.URL.Path, "/v1/admin/") {
				next.ServeHTTP(w, r)
				return
			}

			// Health check is always available
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "3600")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "MAINTENANCE_MODE",
					"message": "The platform is currently under maintenance. Please try again later.",
					"details": []interface{}{},
				},
			})
		})
	}
}
