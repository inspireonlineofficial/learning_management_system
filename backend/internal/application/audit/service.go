package audit

import (
	"context"
	"fmt"
	"time"

	domainaudit "lms-backend/internal/domain/audit"
	"lms-backend/pkg/pagination"

	"github.com/google/uuid"
)

// ListAuditLogsCommand defines filters and pagination for listing audit logs.
// Requirements: 9.5
type ListAuditLogsCommand struct {
	ActorID    *uuid.UUID
	Action     *string
	TargetType *string
	TargetID   *uuid.UUID
	FromDate   *time.Time
	ToDate     *time.Time
	Page       int
	Limit      int
}

// AuditLogResponse is the public representation of an audit log entry.
type AuditLogResponse struct {
	ID         uuid.UUID   `json:"id"`
	ActorID    uuid.UUID   `json:"actor_id"`
	ActorName  string      `json:"actor_name"`
	Action     string      `json:"action"`
	TargetType *string     `json:"target_type,omitempty"`
	TargetID   *uuid.UUID  `json:"target_id,omitempty"`
	Metadata   interface{} `json:"metadata,omitempty"`
	IPAddress  *string     `json:"ip_address,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
}

// ListAuditLogsResponse wraps paginated audit log entries.
type ListAuditLogsResponse struct {
	Data []AuditLogResponse `json:"data"`
	Meta pagination.Meta    `json:"meta"`
}

// Service defines the audit log query use cases.
type Service interface {
	// ListAuditLogs returns paginated, filterable audit log entries. Requirements: 9.5
	ListAuditLogs(ctx context.Context, cmd ListAuditLogsCommand) (*ListAuditLogsResponse, error)
}

type service struct {
	repo domainaudit.AuditLogRepository
}

// NewService creates a new audit service.
func NewService(repo domainaudit.AuditLogRepository) Service {
	return &service{repo: repo}
}

// ListAuditLogs returns paginated, filterable audit log entries.
// Requirements: 9.5
func (s *service) ListAuditLogs(ctx context.Context, cmd ListAuditLogsCommand) (*ListAuditLogsResponse, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.Limit < 1 || cmd.Limit > 100 {
		cmd.Limit = 20
	}

	filter := domainaudit.AuditLogFilter{
		ActorID:    cmd.ActorID,
		Action:     cmd.Action,
		TargetType: cmd.TargetType,
		TargetID:   cmd.TargetID,
		FromDate:   cmd.FromDate,
		ToDate:     cmd.ToDate,
	}

	logs, total, err := s.repo.List(ctx, filter, cmd.Page, cmd.Limit)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}

	items := make([]AuditLogResponse, 0, len(logs))
	for _, l := range logs {
		items = append(items, AuditLogResponse{
			ID:         l.ID,
			ActorID:    l.ActorID,
			ActorName:  l.ActorName,
			Action:     l.Action,
			TargetType: l.TargetType,
			TargetID:   l.TargetID,
			Metadata:   l.Metadata,
			IPAddress:  l.IPAddress,
			CreatedAt:  l.CreatedAt,
		})
	}

	return &ListAuditLogsResponse{
		Data: items,
		Meta: pagination.NewMeta(total, cmd.Page, cmd.Limit),
	}, nil
}
