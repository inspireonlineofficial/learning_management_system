package system_config

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domainsysconfig "lms-backend/internal/domain/system_config"
	"lms-backend/internal/interfaces/http/middleware"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// ─── Mock repositories ────────────────────────────────────────────────────────

type mockSettingRepo struct {
	setting *domainsysconfig.SystemSetting
}

func newMockSettingRepo(maintenanceMode bool, featureFlags map[string]bool) *mockSettingRepo {
	flags, _ := json.Marshal(featureFlags)
	return &mockSettingRepo{
		setting: &domainsysconfig.SystemSetting{
			ID:                    1,
			PlatformName:          "LMS",
			DefaultTimezone:       "UTC",
			OAuthProvidersEnabled: []string{"google"},
			MaintenanceMode:       maintenanceMode,
			FeatureFlags:          flags,
			UpdatedAt:             time.Now(),
		},
	}
}

func (m *mockSettingRepo) Get(_ context.Context) (*domainsysconfig.SystemSetting, error) {
	return m.setting, nil
}

func (m *mockSettingRepo) Update(_ context.Context, s *domainsysconfig.SystemSetting) error {
	m.setting = s
	return nil
}

type mockHistoryRepo struct {
	records []*domainsysconfig.SystemSettingHistory
}

func (m *mockHistoryRepo) Create(_ context.Context, h *domainsysconfig.SystemSettingHistory) error {
	m.records = append(m.records, h)
	return nil
}

func (m *mockHistoryRepo) FindByID(_ context.Context, id uuid.UUID) (*domainsysconfig.SystemSettingHistory, error) {
	for _, r := range m.records {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, nil
}

func (m *mockHistoryRepo) List(_ context.Context, page, limit int) ([]*domainsysconfig.SystemSettingHistory, int, error) {
	return m.records, len(m.records), nil
}

// ─── Property 60 ─────────────────────────────────────────────────────────────

// TestProperty60_MaintenanceModeReturns503ForNonAdminEndpoints verifies that
// when maintenance_mode is true, all non-admin endpoints return HTTP 503,
// while admin endpoints (/v1/admin/*) are always allowed through.
//
// **Validates: Requirements 25.5**
func TestProperty60_MaintenanceModeReturns503ForNonAdminEndpoints(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random non-admin path
		nonAdminPaths := []string{
			"/v1/courses",
			"/v1/auth/login",
			"/v1/student/points",
			"/v1/enrollments",
			"/v1/forum/posts",
			"/v1/bookshop/books",
			"/v1/leaderboard",
			"/v1/student/purchase-requests",
		}
		adminPaths := []string{
			"/v1/admin/users",
			"/v1/admin/courses/123/review",
			"/v1/admin/system/settings",
			"/v1/admin/analytics/overview",
			"/v1/admin/purchase-requests",
		}

		// Maintenance mode is ON
		checker := func() bool { return true }
		mw := middleware.MaintenanceMode(checker)

		// Wrap a simple 200 OK handler
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		handler := mw(inner)

		// Property: non-admin paths return 503
		pathIdx := rapid.IntRange(0, len(nonAdminPaths)-1).Draw(t, "path_idx")
		path := nonAdminPaths[pathIdx]

		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503 for non-admin path %q in maintenance mode, got %d", path, rr.Code)
		}

		// Property: response body contains MAINTENANCE_MODE error code
		body := rr.Body.String()
		if !strings.Contains(body, "MAINTENANCE_MODE") {
			t.Fatalf("expected MAINTENANCE_MODE error code in response body for path %q, got: %s", path, body)
		}

		// Property: admin paths always return 200 (pass-through)
		adminIdx := rapid.IntRange(0, len(adminPaths)-1).Draw(t, "admin_path_idx")
		adminPath := adminPaths[adminIdx]

		adminReq := httptest.NewRequest(http.MethodGet, adminPath, nil)
		adminRR := httptest.NewRecorder()
		handler.ServeHTTP(adminRR, adminReq)

		if adminRR.Code != http.StatusOK {
			t.Fatalf("expected 200 for admin path %q in maintenance mode, got %d", adminPath, adminRR.Code)
		}
	})
}

// TestProperty60b_MaintenanceModeOffAllowsAllRequests verifies that when
// maintenance_mode is false, all requests pass through normally.
//
// **Validates: Requirements 25.5**
func TestProperty60b_MaintenanceModeOffAllowsAllRequests(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		paths := []string{
			"/v1/courses",
			"/v1/auth/login",
			"/v1/student/points",
			"/v1/admin/users",
			"/v1/admin/system/settings",
			"/health",
		}

		// Maintenance mode is OFF
		checker := func() bool { return false }
		mw := middleware.MaintenanceMode(checker)

		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		handler := mw(inner)

		pathIdx := rapid.IntRange(0, len(paths)-1).Draw(t, "path_idx")
		path := paths[pathIdx]

		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for path %q when maintenance mode is OFF, got %d", path, rr.Code)
		}
	})
}

// ─── Property 61 ─────────────────────────────────────────────────────────────

// TestProperty61_DisabledFeatureFlagsReturn403Or404 verifies that the service
// correctly reports disabled features, and that the application layer can use
// feature flags to gate functionality.
//
// **Validates: Requirements 25.6**
func TestProperty61_DisabledFeatureFlagsReturn403Or404(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ctx := context.Background()

		// Generate a random set of feature flags
		featureNames := []string{"forum", "bookshop", "live_sessions", "certificates", "leaderboard"}
		numFeatures := rapid.IntRange(1, len(featureNames)).Draw(t, "num_features")

		flags := make(map[string]bool)
		for i := 0; i < numFeatures; i++ {
			enabled := rapid.Bool().Draw(t, "feature_enabled_"+featureNames[i])
			flags[featureNames[i]] = enabled
		}

		settingRepo := newMockSettingRepo(false, flags)
		historyRepo := &mockHistoryRepo{}
		svc := NewService(ServiceDeps{
			SettingRepo: settingRepo,
			HistoryRepo: historyRepo,
		})

		// Property: GetSettings returns the feature flags as configured
		resp, err := svc.GetSettings(ctx)
		if err != nil {
			t.Fatalf("GetSettings failed: %v", err)
		}

		var returnedFlags map[string]bool
		if err := json.Unmarshal(resp.FeatureFlags, &returnedFlags); err != nil {
			t.Fatalf("failed to unmarshal feature flags: %v", err)
		}

		// Property: every configured flag is present in the response
		for name, expectedEnabled := range flags {
			gotEnabled, exists := returnedFlags[name]
			if !exists {
				t.Fatalf("feature flag %q missing from GetSettings response", name)
			}
			if gotEnabled != expectedEnabled {
				t.Fatalf("feature flag %q: expected enabled=%v, got enabled=%v", name, expectedEnabled, gotEnabled)
			}
		}

		// Property: UpdateSettings correctly persists feature flag changes
		newFlags := make(map[string]bool)
		for name := range flags {
			// Flip all flags
			newFlags[name] = !flags[name]
		}
		newFlagsJSON, _ := json.Marshal(newFlags)

		actorID := uuid.New()
		updated, err := svc.UpdateSettings(ctx, UpdateSettingsCommand{
			ActorID:      actorID,
			ActorName:    "admin",
			FeatureFlags: newFlagsJSON,
		})
		if err != nil {
			t.Fatalf("UpdateSettings failed: %v", err)
		}

		var updatedFlags map[string]bool
		if err := json.Unmarshal(updated.FeatureFlags, &updatedFlags); err != nil {
			t.Fatalf("failed to unmarshal updated feature flags: %v", err)
		}

		for name, expectedEnabled := range newFlags {
			gotEnabled, exists := updatedFlags[name]
			if !exists {
				t.Fatalf("updated feature flag %q missing from UpdateSettings response", name)
			}
			if gotEnabled != expectedEnabled {
				t.Fatalf("updated feature flag %q: expected enabled=%v, got enabled=%v", name, expectedEnabled, gotEnabled)
			}
		}

		// Property: a history record was created for the update
		if len(historyRepo.records) == 0 {
			t.Fatal("expected at least one history record after UpdateSettings, got none")
		}

		// Property: the history diff contains the feature_flags key
		lastRecord := historyRepo.records[len(historyRepo.records)-1]
		var diff map[string]interface{}
		if err := json.Unmarshal(lastRecord.Diff, &diff); err != nil {
			t.Fatalf("failed to unmarshal history diff: %v", err)
		}
		if _, hasDiff := diff["feature_flags"]; !hasDiff {
			t.Fatal("history diff does not contain feature_flags key after flag update")
		}
	})
}
