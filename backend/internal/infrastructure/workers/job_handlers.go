package workers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"lms-backend/internal/domain/notifications"
	"lms-backend/pkg/logger"

	"github.com/google/uuid"
)

// Before the November 2026 audit only "transcode_video" had a registered
// handler. Every other job type that services enqueue (in-app notifications,
// broadcasts, emails, moderation notices, certificate PDFs) was popped from the
// queue and discarded with "no handler registered", and nothing consumed the
// separate notifications queue at all. These handlers close that gap.

// JobHandlerFunc adapts a function to JobHandler.
type JobHandlerFunc func(ctx context.Context, job notifications.Job) error

// Handle implements JobHandler.
func (f JobHandlerFunc) Handle(ctx context.Context, job notifications.Job) error { return f(ctx, job) }

// NotificationCreator persists in-app notifications.
type NotificationCreator interface {
	Create(ctx context.Context, n *notifications.Notification) error
}

func insertNotification(ctx context.Context, repo NotificationCreator, userID uuid.UUID, notifType, title, body string) error {
	return repo.Create(ctx, &notifications.Notification{
		// Same ID on every retry of the job: the insert is ignored the
		// second time, so a retried job never notifies anyone twice.
		ID:        notifications.DedupeID(ctx, "notification", userID.String(), notifType),
		UserID:    userID,
		Type:      notifType,
		Title:     title,
		Body:      body,
		CreatedAt: time.Now().UTC(),
	})
}

// NewSendNotificationJobHandler handles "send_notification" jobs.
func NewSendNotificationJobHandler(repo NotificationCreator) JobHandler {
	return JobHandlerFunc(func(ctx context.Context, job notifications.Job) error {
		var p struct {
			UserID string `json:"user_id"`
			Type   string `json:"type"`
			Title  string `json:"title"`
			Body   string `json:"body"`
		}
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return fmt.Errorf("send_notification payload: %w", err)
		}
		userID, err := uuid.Parse(p.UserID)
		if err != nil {
			return fmt.Errorf("send_notification user_id: %w", err)
		}
		return insertNotification(ctx, repo, userID, p.Type, p.Title, p.Body)
	})
}

// NewBroadcastJobHandler handles "broadcast_notification" jobs.
func NewBroadcastJobHandler(fanOut BroadcastFanOut) JobHandler {
	h := NewBroadcastNotificationJobHandler(fanOut)
	return JobHandlerFunc(func(ctx context.Context, job notifications.Job) error {
		return h.Handle(ctx, &job)
	})
}

// NewForumPostRejectedJobHandler notifies the author of a rejected post.
func NewForumPostRejectedJobHandler(repo NotificationCreator) JobHandler {
	return JobHandlerFunc(func(ctx context.Context, job notifications.Job) error {
		var p struct {
			AuthorID uuid.UUID `json:"author_id"`
			Reason   string    `json:"reason"`
		}
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return fmt.Errorf("forum_post_rejected payload: %w", err)
		}
		body := "Your forum post was not approved."
		if p.Reason != "" {
			body += " Reason: " + p.Reason
		}
		return insertNotification(ctx, repo, p.AuthorID, "forum_post_rejected", "Forum post not approved", body)
	})
}

// NewContentRemovedJobHandler notifies the author when moderators remove a
// flagged post or comment.
func NewContentRemovedJobHandler(db *sql.DB, repo NotificationCreator) JobHandler {
	return JobHandlerFunc(func(ctx context.Context, job notifications.Job) error {
		var p struct {
			TargetType string    `json:"target_type"`
			TargetID   uuid.UUID `json:"target_id"`
		}
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return fmt.Errorf("content_removed payload: %w", err)
		}
		table := "forum_posts"
		label := "post"
		if p.TargetType == "comment" {
			table = "forum_comments"
			label = "comment"
		}
		var authorID uuid.UUID
		// table is chosen from a fixed set above, never from input.
		if err := db.QueryRowContext(ctx, "SELECT author_id FROM "+table+" WHERE id = $1", p.TargetID).Scan(&authorID); err != nil {
			logger.Warn(ctx, "content_removed: author lookup failed", "target_id", p.TargetID, "error", err)
			return nil // nothing to notify; do not retry forever
		}
		return insertNotification(ctx, repo, authorID, "content_removed", "Content removed",
			fmt.Sprintf("A forum %s you wrote was removed by a moderator for violating community guidelines.", label))
	})
}

// NewSendEmailJobHandler handles "send_email" jobs.
func NewSendEmailJobHandler(emailService EmailService) JobHandler {
	return NewEmailJobHandler(emailService)
}

// CertificatePDFGenerator renders certificate PDFs.
type CertificatePDFGenerator interface {
	GenerateByID(ctx context.Context, certificateID uuid.UUID) error
}

// CertificatePDFFunc adapts a function to CertificatePDFGenerator.
type CertificatePDFFunc func(ctx context.Context, certificateID uuid.UUID) error

// GenerateByID implements CertificatePDFGenerator.
func (f CertificatePDFFunc) GenerateByID(ctx context.Context, id uuid.UUID) error { return f(ctx, id) }

// NewGenerateCertificateJobHandler handles "generate_certificate" jobs.
func NewGenerateCertificateJobHandler(gen CertificatePDFGenerator) JobHandler {
	return JobHandlerFunc(func(ctx context.Context, job notifications.Job) error {
		var p struct {
			CertificateID uuid.UUID `json:"CertificateID"`
		}
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return fmt.Errorf("generate_certificate payload: %w", err)
		}
		return gen.GenerateByID(ctx, p.CertificateID)
	})
}
