package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditLog represents an immutable audit log entry. Requirements: 9.4, 9.5
type AuditLog struct {
	ID         uuid.UUID       `json:"id"`
	ActorID    uuid.UUID       `json:"actor_id"`
	ActorName  string          `json:"actor_name"`
	Action     string          `json:"action"`
	TargetType *string         `json:"target_type,omitempty"`
	TargetID   *uuid.UUID      `json:"target_id,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	IPAddress  *string         `json:"ip_address,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// AuditLogFilter defines optional filters for listing audit logs.
type AuditLogFilter struct {
	ActorID    *uuid.UUID
	Action     *string
	TargetType *string
	TargetID   *uuid.UUID
	FromDate   *time.Time
	ToDate     *time.Time
}

// AuditLogRepository defines the read-only query port for audit logs.
// The DB user has no UPDATE/DELETE on audit_logs (Requirements: 9.6).
type AuditLogRepository interface {
	// List returns paginated, filterable audit log entries.
	List(ctx context.Context, filter AuditLogFilter, page, limit int) ([]AuditLog, int, error)
}
