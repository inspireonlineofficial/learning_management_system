package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domainaudit "lms-backend/internal/domain/audit"
	domainnotif "lms-backend/internal/domain/notifications"
	"lms-backend/pkg/apperrors"
	"lms-backend/pkg/logger"
	"lms-backend/pkg/pagination"

	"github.com/google/uuid"
)

// Service defines the interface for all notification use cases.
type Service interface {
	// Delivery (Requirements: 22.1, 22.7)
	SendNotification(ctx context.Context, cmd SendNotificationCommand) error
	SendBroadcast(ctx context.Context, cmd SendBroadcastCommand) (*BroadcastResponse, error)
	ListBroadcasts(ctx context.Context, cmd ListBroadcastsCommand) (*BroadcastHistoryResponse, error)

	// Management (Requirements: 22.2–22.5)
	ListNotifications(ctx context.Context, cmd ListNotificationsCommand) (*NotificationListResponse, error)
	MarkRead(ctx context.Context, cmd MarkReadCommand) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) (*MarkAllReadResponse, error)
	ListTemplates(ctx context.Context) ([]TemplateResponse, error)
	UpdateTemplate(ctx context.Context, cmd UpdateTemplateCommand) (*TemplateResponse, error)
}

// AuditLogger records privileged admin actions.
type AuditLogger interface {
	LogAction(ctx context.Context, actorID uuid.UUID, actorName, action, targetType string, targetID uuid.UUID, metadata map[string]interface{}, ipAddress string) error
}

type service struct {
	notifRepo    domainnotif.NotificationRepository
	templateRepo domainnotif.NotificationTemplateRepository
	jobQueue     domainnotif.JobQueue
	audit        AuditLogger
	auditRepo    domainaudit.AuditLogRepository
}

// NewService creates a new notifications service.
func NewService(
	notifRepo domainnotif.NotificationRepository,
	templateRepo domainnotif.NotificationTemplateRepository,
	jobQueue domainnotif.JobQueue,
	audit AuditLogger,
	auditRepo domainaudit.AuditLogRepository,
) Service {
	return &service{
		notifRepo:    notifRepo,
		templateRepo: templateRepo,
		jobQueue:     jobQueue,
		audit:        audit,
		auditRepo:    auditRepo,
	}
}

// ─── Delivery use cases ───────────────────────────────────────────────────────

// SendNotification inserts an in-app notification and optionally enqueues an email job.
// Requirements: 22.1, 22.7
func (s *service) SendNotification(ctx context.Context, cmd SendNotificationCommand) error {
	n := &domainnotif.Notification{
		ID:        uuid.New(),
		UserID:    cmd.UserID,
		Type:      cmd.Type,
		Title:     cmd.Title,
		Body:      cmd.Body,
		IsRead:    false,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.notifRepo.Create(ctx, n); err != nil {
		logger.Error(ctx, "Failed to create notification", "user_id", cmd.UserID, "type", cmd.Type, "error", err)
		return apperrors.NewInternalError("NOTIFICATION_FAILED", "failed to create notification")
	}

	// Enqueue email if the template channel includes email and an email address is provided
	if cmd.Email != nil && *cmd.Email != "" {
		tmpl, err := s.templateRepo.FindByType(ctx, cmd.Type)
		if err == nil && tmpl != nil && (tmpl.Channel == domainnotif.ChannelEmail || tmpl.Channel == domainnotif.ChannelBoth) {
			s.enqueueEmailJob(ctx, *cmd.Email, cmd.Title, cmd.Body)
		}
	}

	return nil
}

// SendBroadcast fans out a notification to all users of the target role (or all users).
// Records audit log with action broadcast_sent.
// Requirements: 22.6, 22.7
func (s *service) SendBroadcast(ctx context.Context, cmd SendBroadcastCommand) (*BroadcastResponse, error) {
	if cmd.TargetRole != nil {
		role := strings.TrimSpace(*cmd.TargetRole)
		if role != "student" && role != "teacher" && role != "admin" {
			return nil, apperrors.NewSimpleValidationError("INVALID_ROLE", "target_role must be student, teacher, or admin")
		}
		cmd.TargetRole = &role
	}

	cmd.Title = strings.TrimSpace(cmd.Title)
	cmd.Body = strings.TrimSpace(cmd.Body)
	if cmd.Title == "" || cmd.Body == "" {
		return nil, apperrors.NewSimpleValidationError("FIELDS_REQUIRED", "title and body are required")
	}

	type broadcastPayload struct {
		TargetRole *string `json:"target_role,omitempty"`
		Title      string  `json:"title"`
		Body       string  `json:"body"`
	}

	payload, err := json.Marshal(broadcastPayload{
		TargetRole: cmd.TargetRole,
		Title:      cmd.Title,
		Body:       cmd.Body,
	})
	if err != nil {
		return nil, apperrors.NewInternalError("BROADCAST_FAILED", "failed to marshal broadcast payload")
	}

	// Enqueue async broadcast job — fan-out happens in the worker
	job := domainnotif.Job{
		Type:    "broadcast_notification",
		Payload: json.RawMessage(payload),
	}
	if err := s.jobQueue.Enqueue(ctx, job); err != nil {
		return nil, apperrors.NewInternalError("BROADCAST_FAILED", "failed to enqueue broadcast job")
	}

	// Count recipients for the response
	recipientCount, err := s.countRecipients(ctx, cmd.TargetRole)
	if err != nil {
		// Non-fatal — return 0 if count fails
		logger.Error(ctx, "Failed to count broadcast recipients", "error", err)
		recipientCount = 0
	}

	// Record audit log
	if s.audit != nil {
		meta := map[string]interface{}{
			"title":           cmd.Title,
			"body":            cmd.Body,
			"recipient_count": recipientCount,
		}
		if cmd.TargetRole != nil {
			meta["target_role"] = *cmd.TargetRole
		}
		_ = s.audit.LogAction(ctx, cmd.ActorID, cmd.ActorName, "broadcast_sent",
			"notification", uuid.Nil, meta, cmd.IPAddress)
	}

	return &BroadcastResponse{RecipientCount: recipientCount}, nil
}

// ListBroadcasts returns persisted admin broadcast history from audit logs.
// Requirements: 22.6
func (s *service) ListBroadcasts(ctx context.Context, cmd ListBroadcastsCommand) (*BroadcastHistoryResponse, error) {
	if s.auditRepo == nil {
		return &BroadcastHistoryResponse{Items: []BroadcastHistoryItemResponse{}, Meta: map[string]any{"page": 1, "limit": 20, "total": 0, "total_pages": 0}}, nil
	}
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.Limit < 1 || cmd.Limit > 100 {
		cmd.Limit = 20
	}

	action := "broadcast_sent"
	targetType := "notification"
	logs, total, err := s.auditRepo.List(ctx, domainaudit.AuditLogFilter{
		Action:     &action,
		TargetType: &targetType,
	}, cmd.Page, cmd.Limit)
	if err != nil {
		return nil, apperrors.NewInternalError("LIST_BROADCASTS_FAILED", "failed to list broadcast history")
	}

	items := make([]BroadcastHistoryItemResponse, 0, len(logs))
	for _, log := range logs {
		meta := map[string]any{}
		if len(log.Metadata) > 0 {
			_ = json.Unmarshal(log.Metadata, &meta)
		}
		audience := "all"
		if role, ok := meta["target_role"].(string); ok && role != "" {
			switch role {
			case "student":
				audience = "students"
			case "teacher":
				audience = "teachers"
			default:
				audience = role
			}
		}
		title, _ := meta["title"].(string)
		body, _ := meta["body"].(string)
		sentCount := intFromMetadata(meta["recipient_count"])

		items = append(items, BroadcastHistoryItemResponse{
			ID:        log.ID,
			Audience:  audience,
			Title:     title,
			Body:      body,
			SentCount: sentCount,
			CreatedAt: log.CreatedAt,
			Status:    "sent",
		})
	}

	meta := pagination.NewMeta(total, cmd.Page, cmd.Limit)
	return &BroadcastHistoryResponse{
		Items: items,
		Meta: map[string]any{
			"page":        meta.Page,
			"limit":       meta.Limit,
			"total":       meta.Total,
			"total_pages": meta.TotalPages,
		},
	}, nil
}

// ─── Management use cases ─────────────────────────────────────────────────────

// ListNotifications returns paginated notifications with unread_count.
// Requirements: 22.2
func (s *service) ListNotifications(ctx context.Context, cmd ListNotificationsCommand) (*NotificationListResponse, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.Limit < 1 || cmd.Limit > 100 {
		cmd.Limit = 20
	}

	items, total, err := s.notifRepo.ListByUserID(ctx, cmd.UserID, cmd.Page, cmd.Limit)
	if err != nil {
		return nil, apperrors.NewInternalError("LIST_NOTIFICATIONS_FAILED", "failed to list notifications")
	}

	unreadCount, err := s.notifRepo.CountUnread(ctx, cmd.UserID)
	if err != nil {
		unreadCount = 0
	}

	data := make([]*NotificationResponse, 0, len(items))
	for _, n := range items {
		data = append(data, toNotificationResponse(n))
	}

	meta := pagination.NewMeta(total, cmd.Page, cmd.Limit)
	return &NotificationListResponse{
		Data:        data,
		UnreadCount: unreadCount,
		Meta: map[string]any{
			"page":        meta.Page,
			"limit":       meta.Limit,
			"total":       meta.Total,
			"total_pages": meta.TotalPages,
		},
	}, nil
}

// MarkRead marks a single notification as read.
// Requirements: 22.3
func (s *service) MarkRead(ctx context.Context, cmd MarkReadCommand) error {
	n, err := s.notifRepo.FindByID(ctx, cmd.NotificationID)
	if err != nil || n == nil {
		return apperrors.NewNotFoundError("NOTIFICATION_NOT_FOUND", "notification not found")
	}
	if n.UserID != cmd.UserID {
		return apperrors.NewForbiddenError("FORBIDDEN", "cannot mark another user's notification as read")
	}
	if err := s.notifRepo.MarkRead(ctx, cmd.NotificationID); err != nil {
		return apperrors.NewInternalError("MARK_READ_FAILED", "failed to mark notification as read")
	}
	return nil
}

// MarkAllRead marks all unread notifications for a user as read.
// Requirements: 22.4
func (s *service) MarkAllRead(ctx context.Context, userID uuid.UUID) (*MarkAllReadResponse, error) {
	count, err := s.notifRepo.MarkAllRead(ctx, userID)
	if err != nil {
		return nil, apperrors.NewInternalError("MARK_ALL_READ_FAILED", "failed to mark all notifications as read")
	}
	return &MarkAllReadResponse{MarkedCount: count}, nil
}

// ListTemplates returns the configurable notification templates.
func (s *service) ListTemplates(ctx context.Context) ([]TemplateResponse, error) {
	templates, err := s.templateRepo.List(ctx)
	if err != nil {
		return nil, apperrors.NewInternalError("LIST_TEMPLATES_FAILED", "failed to list notification templates")
	}

	responses := make([]TemplateResponse, 0, len(templates))
	for _, template := range templates {
		if template != nil {
			responses = append(responses, *toTemplateResponse(template))
		}
	}
	return responses, nil
}

// UpdateTemplate updates a notification template's subject and body.
// Validates that all template variables in the body are in the allowed_variables list.
// Requirements: 22.5
func (s *service) UpdateTemplate(ctx context.Context, cmd UpdateTemplateCommand) (*TemplateResponse, error) {
	tmpl, err := s.templateRepo.FindByID(ctx, cmd.TemplateID)
	if err != nil || tmpl == nil {
		return nil, apperrors.NewNotFoundError("TEMPLATE_NOT_FOUND", "notification template not found")
	}

	// Validate template variables in body against allowed list
	if err := validateTemplateVariables(cmd.BodyTemplate, tmpl.AllowedVariables); err != nil {
		return nil, err
	}
	if cmd.SubjectTemplate != nil {
		if err := validateTemplateVariables(*cmd.SubjectTemplate, tmpl.AllowedVariables); err != nil {
			return nil, err
		}
	}

	tmpl.BodyTemplate = cmd.BodyTemplate
	if cmd.SubjectTemplate != nil {
		tmpl.SubjectTemplate = cmd.SubjectTemplate
	}
	tmpl.UpdatedAt = time.Now().UTC()

	if err := s.templateRepo.Update(ctx, tmpl); err != nil {
		return nil, apperrors.NewInternalError("UPDATE_TEMPLATE_FAILED", "failed to update notification template")
	}

	return toTemplateResponse(tmpl), nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// enqueueEmailJob enqueues a send_email job for async delivery.
func (s *service) enqueueEmailJob(ctx context.Context, to, subject, body string) {
	type emailPayload struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	data, err := json.Marshal(emailPayload{To: to, Subject: subject, Body: body})
	if err != nil {
		return
	}
	job := domainnotif.Job{
		Type:    "send_email",
		Payload: json.RawMessage(data),
	}
	if err := s.jobQueue.Enqueue(ctx, job); err != nil {
		logger.Error(ctx, "Failed to enqueue email job", "to", to, "error", err)
	}
}

// countRecipients returns the number of users matching the target role.
func (s *service) countRecipients(ctx context.Context, role *string) (int, error) {
	ids, err := s.notifRepo.ListAllUserIDs(ctx, role)
	if err != nil {
		return 0, err
	}
	return len(ids), nil
}

// validateTemplateVariables checks that all {{var}} placeholders in the template
// are present in the allowed list.
func validateTemplateVariables(template string, allowed []string) error {
	allowedSet := make(map[string]bool, len(allowed))
	for _, v := range allowed {
		allowedSet[v] = true
	}

	// Extract {{variable}} patterns
	remaining := template
	for {
		start := strings.Index(remaining, "{{")
		if start == -1 {
			break
		}
		end := strings.Index(remaining[start:], "}}")
		if end == -1 {
			break
		}
		varName := strings.TrimSpace(remaining[start+2 : start+end])
		if !allowedSet[varName] {
			return apperrors.NewSimpleValidationError("INVALID_TEMPLATE_VARIABLE",
				fmt.Sprintf("template variable '{{%s}}' is not in the allowed variables list", varName))
		}
		remaining = remaining[start+end+2:]
	}
	return nil
}

func intFromMetadata(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		var parsed int
		if _, err := fmt.Sscanf(v, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return 0
}
