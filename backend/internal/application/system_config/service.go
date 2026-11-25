package system_config

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	domainsysconfig "lms-backend/internal/domain/system_config"
	"lms-backend/pkg/apperrors"
	"lms-backend/pkg/pagination"

	"github.com/google/uuid"
)

// AuditLogger defines the subset of audit logging needed by this service.
type AuditLogger interface {
	LogAction(ctx context.Context, actorID uuid.UUID, actorName, action, targetType string, targetID uuid.UUID, metadata map[string]interface{}, ipAddress string) error
}

// MaintenanceModeChecker is a function type that returns whether maintenance mode is active.
// It is set by the service and read by the MaintenanceMode middleware.
type MaintenanceModeChecker func() bool

// Service defines the interface for all system config use cases.
type Service interface {
	// GetSettings returns the current platform settings. Requirements: 25.1
	GetSettings(ctx context.Context) (*SettingsResponse, error)

	// UpdateSettings applies a partial update, records a diff, and appends history.
	// Records audit log entry with action "config_changed". Requirements: 25.2
	UpdateSettings(ctx context.Context, cmd UpdateSettingsCommand) (*SettingsResponse, error)

	// GetSettingsHistory returns paginated change history. Requirements: 25.3
	GetSettingsHistory(ctx context.Context, cmd GetSettingsHistoryCommand) (*SettingsHistoryResponse, error)

	// RollbackSettings restores settings to a historical snapshot. Requirements: 25.4
	RollbackSettings(ctx context.Context, cmd RollbackSettingsCommand) (*RollbackResponse, error)

	// IsMaintenanceMode returns true when maintenance_mode is enabled.
	// Used by the MaintenanceMode middleware. Requirements: 25.5
	IsMaintenanceMode(ctx context.Context) bool
}

// ServiceDeps groups all dependencies for the system config service.
type ServiceDeps struct {
	SettingRepo domainsysconfig.SystemSettingRepository
	HistoryRepo domainsysconfig.SystemSettingHistoryRepository
	AuditLogger AuditLogger
}

type service struct {
	settingRepo domainsysconfig.SystemSettingRepository
	historyRepo domainsysconfig.SystemSettingHistoryRepository
	auditLogger AuditLogger
}

// NewService creates a new system config service.
func NewService(deps ServiceDeps) Service {
	return &service{
		settingRepo: deps.SettingRepo,
		historyRepo: deps.HistoryRepo,
		auditLogger: deps.AuditLogger,
	}
}

// GetSettings returns the current platform settings.
// Requirements: 25.1
func (s *service) GetSettings(ctx context.Context) (*SettingsResponse, error) {
	setting, err := s.settingRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}
	return toSettingsResponse(setting), nil
}

// UpdateSettings applies a partial update, computes a field-level diff, appends history,
// and records an audit log entry with action "config_changed".
// Requirements: 25.2
func (s *service) UpdateSettings(ctx context.Context, cmd UpdateSettingsCommand) (*SettingsResponse, error) {
	current, err := s.settingRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current settings: %w", err)
	}

	// Build diff map — only changed fields
	diff := make(map[string]interface{})

	if cmd.PlatformName != nil && *cmd.PlatformName != current.PlatformName {
		diff["platform_name"] = map[string]interface{}{"from": current.PlatformName, "to": *cmd.PlatformName}
		current.PlatformName = *cmd.PlatformName
	}
	if cmd.DefaultTimezone != nil && *cmd.DefaultTimezone != current.DefaultTimezone {
		diff["default_timezone"] = map[string]interface{}{"from": current.DefaultTimezone, "to": *cmd.DefaultTimezone}
		current.DefaultTimezone = *cmd.DefaultTimezone
	}
	if cmd.OAuthProvidersEnabled != nil {
		diff["oauth_providers_enabled"] = map[string]interface{}{"from": current.OAuthProvidersEnabled, "to": cmd.OAuthProvidersEnabled}
		current.OAuthProvidersEnabled = cmd.OAuthProvidersEnabled
	}
	if cmd.MaintenanceMode != nil && *cmd.MaintenanceMode != current.MaintenanceMode {
		diff["maintenance_mode"] = map[string]interface{}{"from": current.MaintenanceMode, "to": *cmd.MaintenanceMode}
		current.MaintenanceMode = *cmd.MaintenanceMode
	}
	if cmd.FeatureFlags != nil {
		diff["feature_flags"] = map[string]interface{}{"from": string(current.FeatureFlags), "to": string(cmd.FeatureFlags)}
		current.FeatureFlags = cmd.FeatureFlags
	}

	if len(diff) == 0 {
		// Nothing changed — return current state without writing
		return toSettingsResponse(current), nil
	}

	current.UpdatedAt = time.Now()
	current.UpdatedBy = &cmd.ActorID

	if err := s.settingRepo.Update(ctx, current); err != nil {
		return nil, fmt.Errorf("update settings: %w", err)
	}

	// Append history record
	diffJSON, _ := json.Marshal(diff)
	snapshotJSON, _ := json.Marshal(current)
	history := &domainsysconfig.SystemSettingHistory{
		ID:        uuid.New(),
		ChangedBy: cmd.ActorID,
		Diff:      diffJSON,
		Snapshot:  snapshotJSON,
		ChangedAt: current.UpdatedAt,
	}
	if err := s.historyRepo.Create(ctx, history); err != nil {
		// Non-fatal: settings are already saved; log the error but don't fail the request
		_ = err
	}

	// Audit log — action "config_changed" (Requirement 9.4)
	if s.auditLogger != nil {
		_ = s.auditLogger.LogAction(ctx, cmd.ActorID, cmd.ActorName, "config_changed", "system_settings", uuid.Nil, map[string]interface{}{"diff": diff}, cmd.IPAddress)
	}

	return toSettingsResponse(current), nil
}

// GetSettingsHistory returns paginated settings change history.
// Requirements: 25.3
func (s *service) GetSettingsHistory(ctx context.Context, cmd GetSettingsHistoryCommand) (*SettingsHistoryResponse, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.Limit < 1 || cmd.Limit > 100 {
		cmd.Limit = 20
	}

	records, total, err := s.historyRepo.List(ctx, cmd.Page, cmd.Limit)
	if err != nil {
		return nil, fmt.Errorf("list settings history: %w", err)
	}

	items := make([]*SettingsHistoryItem, 0, len(records))
	for _, r := range records {
		items = append(items, &SettingsHistoryItem{
			ID:        r.ID,
			ChangedBy: r.ChangedBy,
			Diff:      r.Diff,
			Snapshot:  r.Snapshot,
			ChangedAt: r.ChangedAt,
		})
	}

	return &SettingsHistoryResponse{
		Data: items,
		Meta: pagination.NewMeta(total, cmd.Page, cmd.Limit),
	}, nil
}

// RollbackSettings restores settings to the snapshot from a history record.
// Records a new history entry and audit log for the rollback action.
// Requirements: 25.4
func (s *service) RollbackSettings(ctx context.Context, cmd RollbackSettingsCommand) (*RollbackResponse, error) {
	historyRecord, err := s.historyRepo.FindByID(ctx, cmd.HistoryID)
	if err != nil || historyRecord == nil {
		return nil, apperrors.NewNotFoundError("HISTORY_NOT_FOUND", "settings history record not found")
	}

	// Deserialise the snapshot into a SystemSetting
	var restored domainsysconfig.SystemSetting
	if err := json.Unmarshal(historyRecord.Snapshot, &restored); err != nil {
		return nil, fmt.Errorf("deserialise snapshot: %w", err)
	}

	// Get current settings to compute rollback diff
	current, err := s.settingRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current settings: %w", err)
	}

	now := time.Now()
	restored.UpdatedAt = now
	restored.UpdatedBy = &cmd.ActorID

	if err := s.settingRepo.Update(ctx, &restored); err != nil {
		return nil, fmt.Errorf("rollback settings: %w", err)
	}

	// Append history for the rollback itself
	rollbackDiff := map[string]interface{}{
		"rollback_from_history_id":  cmd.HistoryID,
		"previous_maintenance_mode": current.MaintenanceMode,
		"restored_maintenance_mode": restored.MaintenanceMode,
	}
	diffJSON, _ := json.Marshal(rollbackDiff)
	snapshotJSON, _ := json.Marshal(restored)
	rollbackHistory := &domainsysconfig.SystemSettingHistory{
		ID:        uuid.New(),
		ChangedBy: cmd.ActorID,
		Diff:      diffJSON,
		Snapshot:  snapshotJSON,
		ChangedAt: now,
	}
	_ = s.historyRepo.Create(ctx, rollbackHistory)

	// Audit log
	if s.auditLogger != nil {
		_ = s.auditLogger.LogAction(ctx, cmd.ActorID, cmd.ActorName, "config_changed", "system_settings", cmd.HistoryID, map[string]interface{}{"action": "rollback", "history_id": cmd.HistoryID}, cmd.IPAddress)
	}

	return &RollbackResponse{
		RestoredFromHistoryID: cmd.HistoryID,
		Settings:              toSettingsResponse(&restored),
	}, nil
}

// IsMaintenanceMode returns true when maintenance_mode is currently enabled.
// Called by the MaintenanceMode middleware on every non-admin request. Requirements: 25.5
func (s *service) IsMaintenanceMode(ctx context.Context) bool {
	setting, err := s.settingRepo.Get(ctx)
	if err != nil {
		return false
	}
	return setting.MaintenanceMode
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func toSettingsResponse(s *domainsysconfig.SystemSetting) *SettingsResponse {
	flags := s.FeatureFlags
	if flags == nil {
		flags = json.RawMessage("{}")
	}
	return &SettingsResponse{
		PlatformName:          s.PlatformName,
		DefaultTimezone:       s.DefaultTimezone,
		OAuthProvidersEnabled: s.OAuthProvidersEnabled,
		MaintenanceMode:       s.MaintenanceMode,
		FeatureFlags:          flags,
		UpdatedAt:             s.UpdatedAt,
		UpdatedBy:             s.UpdatedBy,
	}
}
