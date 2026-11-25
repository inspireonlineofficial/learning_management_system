package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Audit regression: short admin-user paths used to panic with a negative
// slice index before authentication ran.
func TestAudit_AdminUserRoutesDoNotPanicOnShortPaths(t *testing.T) {
	s := &Server{}
	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{http.MethodPost, "/v1/admin/users/abc", http.StatusNotFound},
		{http.MethodPost, "/v1/admin/users/a", http.StatusNotFound},
		{http.MethodPatch, "/v1/admin/users/x", http.StatusUnauthorized},
		{http.MethodPost, "/v1/admin/users/123/impersonate", http.StatusUnauthorized},
		{http.MethodDelete, "/v1/admin/users/123", http.StatusNotFound},
	} {
		rec := httptest.NewRecorder()
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("%s %s panicked: %v", tc.method, tc.path, r)
				}
			}()
			s.handleAdminUserRoutes(rec, httptest.NewRequest(tc.method, tc.path, nil))
		}()
		if rec.Code != tc.want {
			t.Errorf("%s %s: got %d, want %d", tc.method, tc.path, rec.Code, tc.want)
		}
	}
}
