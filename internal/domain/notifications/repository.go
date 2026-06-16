package notifications

import (
	"context"

	"github.com/google/uuid"
)

// NotificationRepository defines the persistence interface for Notification.
// Requirements: 22.2, 22.3, 22.4
type NotificationRepository interface {
	Create(ctx context.Context, n *Notification) error
	FindByID(ctx context.Context, id uuid.UUID) (*Notification, error)
	// ListByUserID returns paginated notifications for a user, newest first.
	ListByUserID(ctx context.Context, userID uuid.UUID, page, limit int) ([]*Notification, int, error)
	// CountUnread returns the number of unread notifications for a user.
	CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
	// MarkRead marks a single notification as read.
	MarkRead(ctx context.Context, id uuid.UUID) error
	// MarkAllRead marks all unread notifications for a user as read; returns count updated.
	MarkAllRead(ctx context.Context, userID uuid.UUID) (int, error)
	// ListAllUserIDs returns all distinct user IDs for broadcast fan-out.
	ListAllUserIDs(ctx context.Context, role *string) ([]uuid.UUID, error)
}

// NotificationTemplateRepository defines the persistence interface for NotificationTemplate.
// Requirements: 22.5
type NotificationTemplateRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*NotificationTemplate, error)
	FindByType(ctx context.Context, notifType string) (*NotificationTemplate, error)
	Update(ctx context.Context, t *NotificationTemplate) error
	List(ctx context.Context) ([]*NotificationTemplate, error)
}
