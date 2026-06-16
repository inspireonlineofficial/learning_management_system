package notifications

import (
	"time"

	domainnotif "lms-backend/internal/domain/notifications"

	"github.com/google/uuid"
)

// NotificationResponse is the public-facing notification representation.
// Requirements: 22.2
type NotificationResponse struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

// NotificationListResponse wraps a paginated list of notifications with unread_count.
// Requirements: 22.2
type NotificationListResponse struct {
	Data        []*NotificationResponse `json:"data"`
	UnreadCount int                     `json:"unread_count"`
	Meta        map[string]any          `json:"meta"`
}

// MarkAllReadResponse is returned after marking all notifications as read.
// Requirements: 22.4
type MarkAllReadResponse struct {
	MarkedCount int `json:"marked_count"`
}

// BroadcastResponse is returned after sending a broadcast.
// Requirements: 22.6
type BroadcastResponse struct {
	RecipientCount int `json:"recipient_count"`
}

// TemplateResponse is the public-facing template representation.
type TemplateResponse struct {
	ID               uuid.UUID                       `json:"id"`
	Type             string                          `json:"type"`
	Channel          domainnotif.NotificationChannel `json:"channel" swaggertype:"string" enums:"in_app,email,both"`
	SubjectTemplate  *string                         `json:"subject_template,omitempty"`
	BodyTemplate     string                          `json:"body_template"`
	AllowedVariables []string                        `json:"allowed_variables"`
	UpdatedAt        time.Time                       `json:"updated_at"`
}

func toNotificationResponse(n *domainnotif.Notification) *NotificationResponse {
	return &NotificationResponse{
		ID:        n.ID,
		Type:      n.Type,
		Title:     n.Title,
		Body:      n.Body,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt,
	}
}

func toTemplateResponse(t *domainnotif.NotificationTemplate) *TemplateResponse {
	return &TemplateResponse{
		ID:               t.ID,
		Type:             t.Type,
		Channel:          t.Channel,
		SubjectTemplate:  t.SubjectTemplate,
		BodyTemplate:     t.BodyTemplate,
		AllowedVariables: t.AllowedVariables,
		UpdatedAt:        t.UpdatedAt,
	}
}
