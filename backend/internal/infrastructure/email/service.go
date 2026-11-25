// Package email sends transactional email over SMTP.
//
// It works with any SMTP relay (Amazon SES, Mailgun, Postmark, Brevo, Gmail
// Workspace, a self-hosted Postfix) and with the bundled development mail
// server (cmd/mailsink). Configuration (environment):
//
//	SMTP_HOST, SMTP_PORT        relay address (port 587 STARTTLS, 465 TLS)
//	SMTP_TLS                    "starttls" (default), "tls" (implicit, port 465
//	                            default) or "none" (only for a local dev server)
//	SMTP_USERNAME, SMTP_PASSWORD
//	SMTP_FROM, SMTP_FROM_NAME   envelope sender and display name
//
// With no SMTP_HOST, sending fails with ErrNotConfigured. It used to print
// "would send" and report success, so OTPs and password resets silently
// never arrived.
package email

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

// ErrNotConfigured is returned when no SMTP server is configured.
var ErrNotConfigured = errors.New("email delivery is not configured (set SMTP_HOST)")

// Message is one email. HTML is optional; Text is always sent.
type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

// Config holds SMTP settings.
type Config struct {
	Host     string
	Port     string
	TLSMode  string // "starttls", "tls" or "none"
	Username string
	Password string
	From     string
	FromName string
	// Timeout bounds one delivery (dial + conversation). Default 20s.
	Timeout time.Duration
}

// Service sends email over SMTP.
type Service struct {
	cfg Config
}

// NewService keeps the original constructor signature.
func NewService(host, port, username, password, from string) *Service {
	return NewServiceWithConfig(Config{Host: host, Port: port, Username: username, Password: password, From: from})
}

// NewServiceWithConfig creates a Service from a full configuration.
func NewServiceWithConfig(cfg Config) *Service {
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.Port = strings.TrimSpace(cfg.Port)
	cfg.TLSMode = strings.ToLower(strings.TrimSpace(cfg.TLSMode))
	if cfg.TLSMode == "" {
		if cfg.Port == "465" {
			cfg.TLSMode = "tls"
		} else {
			cfg.TLSMode = "starttls"
		}
	}
	if cfg.Port == "" {
		if cfg.TLSMode == "tls" {
			cfg.Port = "465"
		} else {
			cfg.Port = "587"
		}
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 20 * time.Second
	}
	return &Service{cfg: cfg}
}

// Configured reports whether an SMTP server is set.
func (s *Service) Configured() bool {
	return s.cfg.Host != "" && strings.TrimSpace(s.cfg.From) != ""
}

// Describe returns a log-safe summary (no credentials).
func (s *Service) Describe() string {
	if !s.Configured() {
		return "not configured"
	}
	return fmt.Sprintf("smtp://%s:%s (tls=%s, auth=%t)", s.cfg.Host, s.cfg.Port, s.cfg.TLSMode, s.cfg.Username != "")
}

func (s *Service) SendOTP(ctx context.Context, email, otp string, expiresAt time.Time) error {
	body := fmt.Sprintf("Your Inspire verification code is %s.\n\nThis code expires at %s.", otp, expiresAt.Format(time.RFC3339))
	return s.SendEmail(ctx, email, "Your Inspire verification code", body)
}

func (s *Service) SendPasswordReset(ctx context.Context, email, resetLink string) error {
	body := fmt.Sprintf("Use this link to reset your Inspire password:\n\n%s\n\nIf you did not request this, you can ignore this email.", resetLink)
	return s.SendEmail(ctx, email, "Reset your Inspire password", body)
}

func (s *Service) SendWelcome(ctx context.Context, email, name string) error {
	body := fmt.Sprintf("Hi %s,\n\nWelcome to Inspire.", name)
	return s.SendEmail(ctx, email, "Welcome to Inspire", body)
}

func (s *Service) SendWelcomeEmail(ctx context.Context, email, fullName, tempPassword string) error {
	body := fmt.Sprintf("Hi %s,\n\nAn Inspire account has been created for you.\n\nTemporary password: %s\n\nPlease sign in and change this password.", fullName, tempPassword)
	return s.SendEmail(ctx, email, "Your Inspire account is ready", body)
}

func (s *Service) SendPasswordResetEmail(ctx context.Context, email, fullName, resetToken string) error {
	body := fmt.Sprintf("Hi %s,\n\nUse this reset token to reset your Inspire password:\n\n%s\n\nIf you did not request this, you can ignore this email.", fullName, resetToken)
	return s.SendEmail(ctx, email, "Reset your Inspire password", body)
}

// SendEmail sends a plain-text email.
func (s *Service) SendEmail(ctx context.Context, to, subject, body string) error {
	return s.Send(ctx, Message{To: to, Subject: subject, Text: body})
}

// RecipientError is a recipient address that is invalid or that the server
// refused permanently (5xx); retrying will not help.
type RecipientError struct{ Err error }

func (e RecipientError) Error() string { return e.Err.Error() }
func (e RecipientError) Unwrap() error { return e.Err }

// Send delivers one message.
func (s *Service) Send(ctx context.Context, m Message) error {
	to, err := mail.ParseAddress(strings.TrimSpace(m.To))
	if err != nil {
		return RecipientError{Err: fmt.Errorf("invalid recipient: %w", err)}
	}
	if !s.Configured() {
		return ErrNotConfigured
	}
	from, err := mail.ParseAddress(strings.TrimSpace(s.cfg.From))
	if err != nil {
		return fmt.Errorf("invalid SMTP_FROM: %w", err)
	}
	if s.cfg.FromName != "" {
		from.Name = s.cfg.FromName
	}
	data, err := buildMessage(from, to, m)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(s.cfg.Timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- s.deliver(ctx, deadline, from.Address, to.Address, data) }()
	select {
	case <-ctx.Done():
		return fmt.Errorf("send email: %w", ctx.Err())
	case err := <-done:
		return err
	}
}

// Verify connects, negotiates TLS and logs in, then quits without sending
// anything. It reports exactly which step failed.
func (s *Service) Verify(ctx context.Context) error {
	if !s.Configured() {
		return ErrNotConfigured
	}
	deadline := time.Now().Add(s.cfg.Timeout)
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- s.deliver(ctx, deadline, "", "", nil) }()
	select {
	case <-ctx.Done():
		return fmt.Errorf("verify SMTP: %w", ctx.Err())
	case err := <-done:
		return err
	}
}

// deliver runs one SMTP session. With an empty sender it stops after
// authentication (used by Verify).
func (s *Service) deliver(ctx context.Context, deadline time.Time, from, to string, data []byte) error {
	addr := net.JoinHostPort(s.cfg.Host, s.cfg.Port)
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	tlsConfig := &tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}

	var conn net.Conn
	var err error
	if s.cfg.TLSMode == "tls" {
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: tlsConfig}).DialContext(ctx, "tcp", addr)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("connect to SMTP server: %w", err)
	}
	_ = conn.SetDeadline(deadline)
	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("SMTP greeting: %w", err)
	}
	defer client.Close()

	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("SMTP HELO: %w", err)
	}
	if s.cfg.TLSMode == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			// Refuse to send credentials or mail in clear text when TLS was expected.
			return errors.New("SMTP server does not offer STARTTLS; set SMTP_TLS=tls for port 465, or none only for a local dev server")
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("SMTP STARTTLS: %w", err)
		}
	}
	if s.cfg.Username != "" || s.cfg.Password != "" {
		if ok, _ := client.Extension("AUTH"); !ok {
			return errors.New("SMTP server does not support AUTH")
		}
		// net/smtp's PlainAuth refuses to send the password over an
		// unencrypted connection unless the host is localhost.
		if err := client.Auth(smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}
	if from == "" {
		_ = client.Quit()
		return nil
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		if isPermanent(err) {
			return RecipientError{Err: fmt.Errorf("recipient refused: %w", err)}
		}
		return fmt.Errorf("SMTP RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP message rejected: %w", err)
	}
	_ = client.Quit()
	return nil
}

func isPermanent(err error) bool {
	var tpErr *textproto.Error
	return errors.As(err, &tpErr) && tpErr.Code >= 500 && tpErr.Code < 600
}

func buildMessage(from, to *mail.Address, m Message) ([]byte, error) {
	var buf bytes.Buffer
	header := func(k, v string) { buf.WriteString(k + ": " + v + "\r\n") }
	header("From", from.String())
	header("To", to.String())
	header("Subject", mime.QEncoding.Encode("utf-8", sanitizeHeader(m.Subject)))
	header("Date", time.Now().UTC().Format(time.RFC1123Z))
	header("Message-ID", "<"+randomID()+"@"+domainOf(from.Address)+">")
	header("MIME-Version", "1.0")
	header("Auto-Submitted", "auto-generated")

	if strings.TrimSpace(m.HTML) == "" {
		header("Content-Type", "text/plain; charset=UTF-8")
		header("Content-Transfer-Encoding", "quoted-printable")
		buf.WriteString("\r\n")
		if err := writeQP(&buf, m.Text); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	mw := multipart.NewWriter(&buf)
	header("Content-Type", "multipart/alternative; boundary="+mw.Boundary())
	buf.WriteString("\r\n")
	for _, part := range []struct{ ctype, body string }{
		{"text/plain; charset=UTF-8", m.Text},
		{"text/html; charset=UTF-8", m.HTML},
	} {
		pw, err := mw.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {part.ctype},
			"Content-Transfer-Encoding": {"quoted-printable"},
		})
		if err != nil {
			return nil, err
		}
		if err := writeQP(pw, part.body); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeQP(w interface{ Write([]byte) (int, error) }, s string) error {
	qp := quotedprintable.NewWriter(w)
	if _, err := qp.Write([]byte(strings.ReplaceAll(s, "\r\n", "\n"))); err != nil {
		return err
	}
	return qp.Close()
}

func randomID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%d.%s", time.Now().UnixNano(), hex.EncodeToString(b))
}

func domainOf(addr string) string {
	if i := strings.LastIndex(addr, "@"); i >= 0 && i < len(addr)-1 {
		return addr[i+1:]
	}
	return "localhost"
}

func sanitizeHeader(value string) string {
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	return value
}
