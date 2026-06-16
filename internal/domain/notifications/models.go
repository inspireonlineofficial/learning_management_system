package notifications

import (
	"time"

	"github.com/google/uuid"
)

// NotificationChannel defines the delivery channel for a notification.
type NotificationChannel string

const (
	ChannelInApp NotificationChannel = "in_app"
	ChannelEmail NotificationChannel = "email"
	ChannelBoth  NotificationChannel = "both"
)

// Notification is an entity representing a single in-app notification for a user.
// Requirements: 22.1, 22.2
type Notification struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

// NotificationTemplate is an entity defining the subject/body template for a notification type.
// Requirements: 22.5
type NotificationTemplate struct {
	ID               uuid.UUID           `json:"id"`
	Type             string              `json:"type"`
	Channel          NotificationChannel `json:"channel"`
	SubjectTemplate  *string             `json:"subject_template,omitempty"`
	BodyTemplate     string              `json:"body_template"`
	AllowedVariables []string            `json:"allowed_variables"`
	UpdatedAt        time.Time           `json:"updated_at"`
}
