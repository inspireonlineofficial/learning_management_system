package system_config

import (
	"context"

	"github.com/google/uuid"
)

// SystemSettingRepository defines the persistence port for the singleton system settings.
// Requirements: 25.1, 25.3
type SystemSettingRepository interface {
	// Get returns the current system settings (singleton, id=1).
	Get(ctx context.Context) (*SystemSetting, error)

	// Update persists changes to the singleton settings record.
	Update(ctx context.Context, setting *SystemSetting) error
}

// SystemSettingHistoryRepository defines the persistence port for the append-only settings history.
// No Update or Delete methods — history is immutable. Requirements: 25.3, 25.4
type SystemSettingHistoryRepository interface {
	// Create appends a new history record.
	Create(ctx context.Context, history *SystemSettingHistory) error

	// FindByID returns a single history record by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*SystemSettingHistory, error)

	// List returns paginated history records ordered by changed_at DESC.
	List(ctx context.Context, page, limit int) ([]*SystemSettingHistory, int, error)
}
