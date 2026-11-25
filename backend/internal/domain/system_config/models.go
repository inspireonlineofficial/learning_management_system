package system_config

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SystemSetting is the singleton aggregate root for platform-wide configuration.
// id is always 1 (singleton pattern). Requirements: 25.1, 25.3
type SystemSetting struct {
	ID                    int             `json:"id"`
	PlatformName          string          `json:"platform_name"`
	DefaultTimezone       string          `json:"default_timezone"`
	OAuthProvidersEnabled []string        `json:"oauth_providers_enabled"`
	MaintenanceMode       bool            `json:"maintenance_mode"`
	FeatureFlags          json.RawMessage `json:"feature_flags"`
	UpdatedAt             time.Time       `json:"updated_at"`
	UpdatedBy             *uuid.UUID      `json:"updated_by"`
}

// SystemSettingHistory is an append-only record of every settings change.
// diff contains only the changed fields; snapshot contains the full state after the change.
// Requirements: 25.3, 25.4
type SystemSettingHistory struct {
	ID        uuid.UUID       `json:"id"`
	ChangedBy uuid.UUID       `json:"changed_by"`
	Diff      json.RawMessage `json:"diff"`
	Snapshot  json.RawMessage `json:"snapshot"`
	ChangedAt time.Time       `json:"changed_at"`
}
