package email

import (
	"bytes"
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

// Service implements email sending
type Service struct {
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	smtpFrom     string
}

// NewService creates a new email service
func NewService(host, port, username, password, from string) *Service {
	return &Service{
		smtpHost:     host,
		smtpPort:     port,
		smtpUsername: username,
		smtpPassword: password,
		smtpFrom:     from,
	}
}

// SendOTP sends an OTP email
func (s *Service) SendOTP(ctx context.Context, email, otp string, expiresAt time.Time) error {
	body := fmt.Sprintf("Your Inspire verification code is %s.\n\nThis code expires at %s.", otp, expiresAt.Format(time.RFC3339))
	return s.send(ctx, email, "Your Inspire verification code", body)
}

// SendPasswordReset sends a password reset email
func (s *Service) SendPasswordReset(ctx context.Context, email, resetLink string) error {
	body := fmt.Sprintf("Use this link to reset your Inspire password:\n\n%s\n\nIf you did not request this, you can ignore this email.", resetLink)
	return s.send(ctx, email, "Reset your Inspire password", body)
}

// SendWelcome sends a welcome email
func (s *Service) SendWelcome(ctx context.Context, email, name string) error {
	body := fmt.Sprintf("Hi %s,\n\nWelcome to Inspire.", name)
	return s.send(ctx, email, "Welcome to Inspire", body)
}

// SendWelcomeEmail sends a welcome email with temporary credentials
func (s *Service) SendWelcomeEmail(ctx context.Context, email, fullName, tempPassword string) error {
	body := fmt.Sprintf("Hi %s,\n\nAn Inspire account has been created for you.\n\nTemporary password: %s\n\nPlease sign in and change this password.", fullName, tempPassword)
	return s.send(ctx, email, "Your Inspire account is ready", body)
}

// SendPasswordResetEmail sends a password reset email with reset token
func (s *Service) SendPasswordResetEmail(ctx context.Context, email, fullName, resetToken string) error {
	body := fmt.Sprintf("Hi %s,\n\nUse this reset token to reset your Inspire password:\n\n%s\n\nIf you did not request this, you can ignore this email.", fullName, resetToken)
	return s.send(ctx, email, "Reset your Inspire password", body)
}

func (s *Service) send(ctx context.Context, to, subject, body string) error {
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("email recipient is required")
	}
	if strings.TrimSpace(s.smtpHost) == "" || strings.TrimSpace(s.smtpFrom) == "" {
		fmt.Printf("Email delivery not configured; would send %q to %s\n", subject, to)
		return nil
	}

	port := strings.TrimSpace(s.smtpPort)
	if port == "" {
		port = "587"
	}
	addr := strings.TrimSpace(s.smtpHost) + ":" + port

	var msg bytes.Buffer
	msg.WriteString("From: " + sanitizeHeader(s.smtpFrom) + "\r\n")
	msg.WriteString("To: " + sanitizeHeader(to) + "\r\n")
	msg.WriteString("Subject: " + sanitizeHeader(subject) + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	var auth smtp.Auth
	if strings.TrimSpace(s.smtpUsername) != "" || strings.TrimSpace(s.smtpPassword) != "" {
		auth = smtp.PlainAuth("", s.smtpUsername, s.smtpPassword, s.smtpHost)
	}

	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(addr, auth, s.smtpFrom, []string{to}, msg.Bytes())
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("send email: %w", err)
		}
		return nil
	}
}

func sanitizeHeader(value string) string {
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	return value
}
