package workers

import (
	"context"
	"encoding/json"
	"lms-backend/internal/domain/notifications"
	"lms-backend/pkg/logger"
)

// EmailService defines the interface for sending emails
type EmailService interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

// EmailJobHandler handles email sending jobs
type EmailJobHandler struct {
	emailService EmailService
}

// NewEmailJobHandler creates a new email job handler
func NewEmailJobHandler(emailService EmailService) *EmailJobHandler {
	return &EmailJobHandler{
		emailService: emailService,
	}
}

// Handle processes an email job
func (h *EmailJobHandler) Handle(ctx context.Context, job notifications.Job) error {
	var payload struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}

	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		logger.Error(ctx, "Failed to unmarshal email job payload", "error", err)
		return err
	}

	if err := h.emailService.SendEmail(ctx, payload.To, payload.Subject, payload.Body); err != nil {
		logger.Error(ctx, "Failed to send email", "to", payload.To, "error", err)
		return err
	}

	logger.Info(ctx, "Email sent successfully", "to", payload.To, "subject", payload.Subject)
	return nil
}
