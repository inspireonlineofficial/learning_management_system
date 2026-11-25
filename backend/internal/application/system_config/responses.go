package system_config

import (
	"encoding/json"
	"time"

	"lms-backend/pkg/pagination"

	"github.com/google/uuid"
)

// SettingsResponse is the public representation of the current system settings.
// Requirements: 25.1
type SettingsResponse struct {
	PlatformName          string          `json:"platform_name"`
	DefaultTimezone       string          `json:"default_timezone"`
	OAuthProvidersEnabled []string        `json:"oauth_providers_enabled"`
	MaintenanceMode       bool            `json:"maintenance_mode"`
	FeatureFlags          json.RawMessage `json:"feature_flags" swaggertype:"object"`
	UpdatedAt             time.Time       `json:"updated_at"`
	UpdatedBy             *uuid.UUID      `json:"updated_by,omitempty"`
}

// SettingsHistoryItem represents a single entry in the settings change history.
// Requirements: 25.3
type SettingsHistoryItem struct {
	ID        uuid.UUID       `json:"id"`
	ChangedBy uuid.UUID       `json:"changed_by"`
	Diff      json.RawMessage `json:"diff" swaggertype:"object"`
	Snapshot  json.RawMessage `json:"snapshot" swaggertype:"object"`
	ChangedAt time.Time       `json:"changed_at"`
}

// SettingsHistoryResponse wraps a paginated list of history items.
type SettingsHistoryResponse struct {
	Data []*SettingsHistoryItem `json:"data"`
	Meta pagination.Meta        `json:"meta"`
}

// RollbackResponse confirms a successful settings rollback.
// Requirements: 25.4
type RollbackResponse struct {
	RestoredFromHistoryID uuid.UUID         `json:"restored_from_history_id"`
	Settings              *SettingsResponse `json:"settings"`
}
