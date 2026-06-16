package notifications

import "github.com/google/uuid"

// SendNotificationCommand holds data for sending a single notification.
// Requirements: 22.1, 22.7
type SendNotificationCommand struct {
	UserID uuid.UUID
	Type   string
	Title  string
	Body   string
	// Email is optional; if set and the template channel includes email, an email is also sent.
	Email *string
}

// SendBroadcastCommand holds data for sending a broadcast notification.
// Requirements: 22.6
type SendBroadcastCommand struct {
	ActorID   uuid.UUID
	ActorName string
	// TargetRole is optional; nil means all users.
	TargetRole *string
	Title      string
	Body       string
	IPAddress  string
}

// ListNotificationsCommand holds pagination for listing notifications.
// Requirements: 22.2
type ListNotificationsCommand struct {
	UserID uuid.UUID
	Page   int
	Limit  int
}

// MarkReadCommand marks a single notification as read.
// Requirements: 22.3
type MarkReadCommand struct {
	NotificationID uuid.UUID
	UserID         uuid.UUID
}

// UpdateTemplateCommand updates a notification template.
// Requirements: 22.5
type UpdateTemplateCommand struct {
	TemplateID      uuid.UUID
	SubjectTemplate *string
	BodyTemplate    string
}
