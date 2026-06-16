package system_config

import (
	"encoding/json"

	"github.com/google/uuid"
)

// GetSettingsCommand retrieves the current system settings.
// Requirements: 25.1
type GetSettingsCommand struct {
	// No fields — singleton fetch.
}

// UpdateSettingsCommand updates one or more system settings fields.
// Only non-nil fields are applied. Requirements: 25.2
type UpdateSettingsCommand struct {
	ActorID               uuid.UUID
	ActorName             string
	IPAddress             string
	PlatformName          *string
	DefaultTimezone       *string
	OAuthProvidersEnabled []string
	MaintenanceMode       *bool
	FeatureFlags          json.RawMessage
}

// GetSettingsHistoryCommand retrieves paginated settings history.
// Requirements: 25.3
type GetSettingsHistoryCommand struct {
	Page  int
	Limit int
}

// RollbackSettingsCommand restores settings to a historical snapshot.
// Requirements: 25.4
type RollbackSettingsCommand struct {
	ActorID   uuid.UUID
	ActorName string
	IPAddress string
	HistoryID uuid.UUID
}
