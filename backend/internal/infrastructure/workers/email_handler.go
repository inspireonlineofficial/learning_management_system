package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"lms-backend/internal/domain/notifications"
	"lms-backend/internal/infrastructure/email"
	"lms-backend/pkg/logger"
)

// EmailService defines the interface for sending emails
type EmailService interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

// MessageSender can send HTML + text messages.
type MessageSender interface {
	Send(ctx context.Context, m email.Message) error
}

// EmailJobHandler handles "send_email" jobs. Transient SMTP failures are
// returned so the worker retries; an invalid or refused recipient and a
// missing SMTP configuration are permanent (the job is dead-lettered).
type EmailJobHandler struct {
	emailService EmailService
}

// NewEmailJobHandler creates a new email job handler
func NewEmailJobHandler(emailService EmailService) *EmailJobHandler {
	return &EmailJobHandler{emailService: emailService}
}

// Handle processes an email job
func (h *EmailJobHandler) Handle(ctx context.Context, job notifications.Job) error {
	var payload struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
		HTML    string `json:"html"`
	}
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return Permanent(fmt.Errorf("email job payload: %w", err))
	}

	var err error
	if sender, ok := h.emailService.(MessageSender); ok {
		err = sender.Send(ctx, email.Message{To: payload.To, Subject: payload.Subject, Text: payload.Body, HTML: payload.HTML})
	} else {
		err = h.emailService.SendEmail(ctx, payload.To, payload.Subject, payload.Body)
	}
	if err != nil {
		var recipient email.RecipientError
		if errors.As(err, &recipient) || errors.Is(err, email.ErrNotConfigured) {
			logger.Error(ctx, "Email not deliverable", "subject", payload.Subject, "error", err)
			return Permanent(err)
		}
		return err
	}

	logger.Info(ctx, "Email sent", "subject", payload.Subject)
	return nil
}
