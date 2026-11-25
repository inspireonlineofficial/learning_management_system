package workers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"lms-backend/internal/domain/notifications"
	"lms-backend/internal/infrastructure/email"
	"lms-backend/pkg/logger"

	"github.com/google/uuid"
)

// ContactLookup finds where and whether to email a user.
type ContactLookup interface {
	// Contact returns the user's name and email and whether they accept
	// email notifications (user_settings.email_notifications, default true).
	Contact(ctx context.Context, userID uuid.UUID) (name, address string, emailOK bool, err error)
}

// SQLContactLookup reads users and user_settings.
type SQLContactLookup struct{ DB *sql.DB }

func (l SQLContactLookup) Contact(ctx context.Context, userID uuid.UUID) (string, string, bool, error) {
	var name, address string
	var emailOK bool
	err := l.DB.QueryRowContext(ctx, `
		SELECT u.full_name, u.email, COALESCE(s.email_notifications, TRUE)
		FROM users u LEFT JOIN user_settings s ON s.user_id = u.id
		WHERE u.id = $1 AND u.deleted_at IS NULL`, userID).Scan(&name, &address, &emailOK)
	return name, address, emailOK, err
}

// JobEnqueuer queues a background job.
type JobEnqueuer interface {
	Enqueue(ctx context.Context, job notifications.Job) error
}

// StudentMessenger sends one event to a student as an in-app notification
// and, when they accept email, as an email. The email is a separate
// "send_email" job so an SMTP outage is retried on its own without
// duplicating the in-app notification.
type StudentMessenger struct {
	Notifications NotificationCreator
	Contacts      ContactLookup
	Queue         JobEnqueuer
	FrontendURL   string
}

// StudentMessage is one event for one student.
type StudentMessage struct {
	UserID     uuid.UUID
	Type       string
	Title      string
	Body       string
	Paragraphs []string // email body; defaults to Body
	Action     *email.Action
}

func (m StudentMessenger) link(path string) string {
	base := strings.TrimRight(m.FrontendURL, "/")
	if base == "" {
		return ""
	}
	return base + path
}

// Send records the in-app notification and queues the email.
func (m StudentMessenger) Send(ctx context.Context, msg StudentMessage) error {
	if err := insertNotification(ctx, m.Notifications, msg.UserID, msg.Type, msg.Title, msg.Body); err != nil {
		return err // nothing done yet; safe to retry the whole job
	}
	if m.Contacts == nil || m.Queue == nil {
		return nil
	}
	name, address, emailOK, err := m.Contacts.Contact(ctx, msg.UserID)
	if err != nil || !emailOK || address == "" {
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			logger.Warn(ctx, "Skipping email: contact lookup failed", "type", msg.Type, "error", err)
		}
		return nil
	}
	paragraphs := msg.Paragraphs
	if len(paragraphs) == 0 {
		paragraphs = []string{msg.Body}
	}
	greeting := "Hi,"
	if first := strings.Fields(name); len(first) > 0 {
		greeting = "Hi " + first[0] + ","
	}
	rendered := email.Transactional(msg.Title, greeting, paragraphs, msg.Action,
		"You are receiving this because you have an Inspire account. Turn off email notifications in your account settings.")
	payload, _ := json.Marshal(map[string]string{
		"to": address, "subject": rendered.Subject, "body": rendered.Text, "html": rendered.HTML,
	})
	// The email job's ID is derived from this job and recipient, so if this
	// job is retried the email is queued again under the same ID and the
	// worker sends it once.
	emailJob := notifications.Job{Type: "send_email", Payload: payload}
	if key := notifications.JobKey(ctx); key != "" {
		emailJob.ID = notifications.DedupeID(ctx, "email", msg.UserID.String(), msg.Type).String()
	}
	if err := m.Queue.Enqueue(ctx, emailJob); err != nil {
		// The in-app notification is already stored; don't fail (and repeat) it.
		logger.Error(ctx, "Failed to queue email", "type", msg.Type, "error", err)
	}
	return nil
}

// NewBookOrderJobHandler turns bookshop order events into student messages.
func NewBookOrderJobHandler(m StudentMessenger) JobHandler {
	return JobHandlerFunc(func(ctx context.Context, job notifications.Job) error {
		var p struct {
			OrderID        uuid.UUID `json:"order_id"`
			StudentID      uuid.UUID `json:"student_id"`
			BookID         uuid.UUID `json:"book_id"`
			BookTitle      string    `json:"book_title"`
			Format         string    `json:"format"`
			TrackingNumber string    `json:"tracking_number"`
		}
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return Permanent(fmt.Errorf("%s payload: %w", job.Type, err))
		}
		if p.StudentID == uuid.Nil {
			return Permanent(fmt.Errorf("%s payload has no student", job.Type))
		}
		book := p.BookTitle
		if book == "" {
			book = "your book"
		}
		orderLink := &email.Action{Label: "View order", URL: m.link("/student/bookshop/orders/" + p.OrderID.String())}
		msg := StudentMessage{UserID: p.StudentID, Type: job.Type, Action: orderLink}
		switch job.Type {
		case "order_placed":
			msg.Title = "Order confirmed"
			if p.Format == "digital" {
				msg.Body = book + " is now in your library."
				msg.Action = &email.Action{Label: "Read now", URL: m.link("/student/bookshop/reader/" + p.BookID.String())}
			} else {
				msg.Body = "We received your order for " + book + ". We'll let you know when it ships."
			}
		case "order_shipped":
			msg.Title = "Your book has shipped"
			msg.Body = book + " is on its way."
			if p.TrackingNumber != "" {
				msg.Body += " Tracking number: " + p.TrackingNumber + "."
			}
		case "order_delivered":
			msg.Title = "Book delivered"
			msg.Body = book + " has been marked as delivered. If it hasn't arrived, reply from your order page."
		case "order_cancelled":
			msg.Title = "Order cancelled"
			msg.Body = "Your order for " + book + " was cancelled."
		case "order_refunded":
			msg.Title = "Order refunded"
			msg.Body = "Your order for " + book + " was refunded."
			if p.Format == "digital" {
				msg.Body += " It has been removed from your library."
			}
		default:
			return nil
		}
		return m.Send(ctx, msg)
	})
}

// NewPurchaseDecisionJobHandler tells a student their purchase request for a
// course or book was approved or declined.
func NewPurchaseDecisionJobHandler(m StudentMessenger) JobHandler {
	return JobHandlerFunc(func(ctx context.Context, job notifications.Job) error {
		var p struct {
			StudentID  uuid.UUID `json:"student_id"`
			ItemType   string    `json:"item_type"`
			ItemID     uuid.UUID `json:"item_id"`
			ItemTitle  string    `json:"item_title"`
			BookFormat string    `json:"book_format"`
			OrderID    string    `json:"order_id"`
			Reason     string    `json:"reason"`
		}
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return Permanent(fmt.Errorf("%s payload: %w", job.Type, err))
		}
		if p.StudentID == uuid.Nil {
			return Permanent(fmt.Errorf("%s payload has no student", job.Type))
		}
		title := p.ItemTitle
		if title == "" {
			title = "your purchase"
		}
		msg := StudentMessage{UserID: p.StudentID, Type: job.Type}
		switch job.Type {
		case "purchase_approved":
			msg.Title = "Purchase approved"
			switch {
			case p.ItemType == "course":
				msg.Body = "Your payment was confirmed. You're now enrolled in " + title + "."
				msg.Action = &email.Action{Label: "Start learning", URL: m.link("/student/courses/" + p.ItemID.String())}
			case p.BookFormat == "physical":
				msg.Body = "Your payment for " + title + " was confirmed. We'll let you know when your printed copy ships."
				if p.OrderID != "" {
					msg.Action = &email.Action{Label: "View order", URL: m.link("/student/bookshop/orders/" + p.OrderID)}
				}
			default:
				msg.Body = "Your payment for " + title + " was confirmed. The book is in your library."
				msg.Action = &email.Action{Label: "Read now", URL: m.link("/student/bookshop/reader/" + p.ItemID.String())}
			}
		case "purchase_rejected":
			msg.Title = "Purchase request declined"
			msg.Body = "Your purchase request for " + title + " was declined."
			if strings.TrimSpace(p.Reason) != "" {
				msg.Body += " Reason: " + p.Reason
			}
			msg.Paragraphs = []string{msg.Body, "If you think this is a mistake, you can send a new request with your payment reference."}
			if p.ItemType == "book" {
				msg.Action = &email.Action{Label: "View requests", URL: m.link("/student/bookshop/requests")}
			}
		default:
			return nil
		}
		return m.Send(ctx, msg)
	})
}

// NewRefundRequestJobHandler tells a student what happened to their refund request.
func NewRefundRequestJobHandler(m StudentMessenger) JobHandler {
	return JobHandlerFunc(func(ctx context.Context, job notifications.Job) error {
		var p struct {
			StudentID uuid.UUID `json:"student_id"`
			OrderID   string    `json:"order_id"`
			BookTitle string    `json:"book_title"`
			Note      string    `json:"note"`
		}
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return Permanent(fmt.Errorf("%s payload: %w", job.Type, err))
		}
		if p.StudentID == uuid.Nil {
			return Permanent(fmt.Errorf("%s payload has no student", job.Type))
		}
		msg := StudentMessage{UserID: p.StudentID, Type: job.Type,
			Action: &email.Action{Label: "View order", URL: m.link("/student/bookshop/orders/" + p.OrderID)}}
		switch job.Type {
		case "refund_request_received":
			msg.Title = "Refund request received"
			msg.Body = "We received your refund request for " + p.BookTitle + ". An admin will review it."
		case "refund_request_rejected":
			msg.Title = "Refund request declined"
			msg.Body = "Your refund request for " + p.BookTitle + " was declined."
			if strings.TrimSpace(p.Note) != "" {
				msg.Body += " " + p.Note
			}
		default:
			return nil
		}
		return m.Send(ctx, msg)
	})
}

// AdminLookup lists the admins who should hear about new work.
type AdminLookup interface {
	ActiveAdminIDs(ctx context.Context) ([]uuid.UUID, error)
}

// SQLAdminLookup reads active admin accounts.
type SQLAdminLookup struct{ DB *sql.DB }

func (l SQLAdminLookup) ActiveAdminIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := l.DB.QueryContext(ctx, `
		SELECT id FROM users
		WHERE role = 'admin' AND status = 'active' AND deleted_at IS NULL
		ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// NewRefundRequestAdminAlertHandler tells every active admin, in the app and
// by email, that a student asked for a refund. Each admin is notified once
// even if the job is retried.
func NewRefundRequestAdminAlertHandler(m StudentMessenger, admins AdminLookup) JobHandler {
	return JobHandlerFunc(func(ctx context.Context, job notifications.Job) error {
		var p struct {
			RequestID   string  `json:"request_id"`
			OrderID     string  `json:"order_id"`
			BookTitle   string  `json:"book_title"`
			StudentName string  `json:"student_name"`
			Amount      float64 `json:"amount"`
			Currency    string  `json:"currency"`
			Reason      string  `json:"reason"`
		}
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return Permanent(fmt.Errorf("%s payload: %w", job.Type, err))
		}
		ids, err := admins.ActiveAdminIDs(ctx)
		if err != nil {
			return err
		}
		student := p.StudentName
		if student == "" {
			student = "A student"
		}
		body := fmt.Sprintf("%s asked for a refund of %s (%.2f %s).", student, p.BookTitle, p.Amount, p.Currency)
		paragraphs := []string{body}
		if strings.TrimSpace(p.Reason) != "" {
			paragraphs = append(paragraphs, "Reason: "+p.Reason)
		}
		paragraphs = append(paragraphs, "Send the refund first, then approve the request with the transaction ID, or decline it with a note.")
		var firstErr error
		for _, id := range ids {
			err := m.Send(ctx, StudentMessage{
				UserID: id, Type: "refund_request_admin", Title: "New refund request", Body: body, Paragraphs: paragraphs,
				Action: &email.Action{Label: "Review refund requests", URL: m.link("/admin/bookshop/refunds")},
			})
			if err != nil && firstErr == nil {
				firstErr = err // retried; admins already notified are skipped
			}
		}
		return firstErr
	})
}
