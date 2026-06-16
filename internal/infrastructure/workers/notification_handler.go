package workers

import (
	"context"
	"encoding/json"
	"lms-backend/internal/domain/notifications"
	"lms-backend/pkg/logger"
)

// NotificationInserter inserts a single in-app notification record.
type NotificationInserter interface {
	InsertNotification(ctx context.Context, userID, notifType, title, body string) error
}

// BroadcastFanOut fans out a broadcast notification to all target users.
type BroadcastFanOut interface {
	FanOut(ctx context.Context, role *string, notifType, title, body string) error
}

// SendNotificationHandler handles send_notification jobs.
// Requirements: 22.7
type SendNotificationHandler struct {
	inserter NotificationInserter
}

// NewSendNotificationHandler creates a new SendNotificationHandler.
func NewSendNotificationHandler(inserter NotificationInserter) *SendNotificationHandler {
	return &SendNotificationHandler{inserter: inserter}
}

// Handle processes a send_notification job.
func (h *SendNotificationHandler) Handle(ctx context.Context, job *notifications.Job) error {
	var payload struct {
		UserID string `json:"user_id"`
		Type   string `json:"type"`
		Title  string `json:"title"`
		Body   string `json:"body"`
	}
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		logger.Error(ctx, "Failed to unmarshal send_notification payload", "error", err)
		return err
	}
	return h.inserter.InsertNotification(ctx, payload.UserID, payload.Type, payload.Title, payload.Body)
}

// BroadcastNotificationJobHandler handles broadcast_notification jobs.
// Requirements: 22.6, 22.7
type BroadcastNotificationJobHandler struct {
	fanOut BroadcastFanOut
}

// NewBroadcastNotificationJobHandler creates a new BroadcastNotificationJobHandler.
func NewBroadcastNotificationJobHandler(fanOut BroadcastFanOut) *BroadcastNotificationJobHandler {
	return &BroadcastNotificationJobHandler{fanOut: fanOut}
}

// Handle processes a broadcast_notification job.
func (h *BroadcastNotificationJobHandler) Handle(ctx context.Context, job *notifications.Job) error {
	var payload struct {
		TargetRole *string `json:"target_role,omitempty"`
		Title      string  `json:"title"`
		Body       string  `json:"body"`
	}
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		logger.Error(ctx, "Failed to unmarshal broadcast_notification payload", "error", err)
		return err
	}

	logger.Info(ctx, "Processing broadcast notification", "target_role", payload.TargetRole)
	return h.fanOut.FanOut(ctx, payload.TargetRole, "broadcast", payload.Title, payload.Body)
}
