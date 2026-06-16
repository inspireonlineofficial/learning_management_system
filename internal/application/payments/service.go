package payments

import (
	"context"
	"fmt"
	"strings"
	"time"

	"lms-backend/internal/domain/auth"
	domainbookshop "lms-backend/internal/domain/bookshop"
	"lms-backend/internal/domain/courses"
	"lms-backend/internal/domain/enrollments"
	domainpayments "lms-backend/internal/domain/payments"
	"lms-backend/pkg/apperrors"
	"lms-backend/pkg/pagination"

	"github.com/google/uuid"
)

// Service defines the approval workflow use cases.
type Service interface {
	CreatePurchaseRequest(ctx context.Context, cmd CreatePurchaseRequestCommand) (*PurchaseRequestItem, error)
	GetStudentPurchaseRequests(ctx context.Context, cmd ListPurchaseRequestsCommand) (*PurchaseRequestListResponse, error)
	ListAdminPurchaseRequests(ctx context.Context, cmd ListPurchaseRequestsCommand) (*PurchaseRequestListResponse, error)
	ApprovePurchaseRequest(ctx context.Context, cmd ReviewPurchaseRequestCommand) (*PurchaseRequestItem, error)
	RejectPurchaseRequest(ctx context.Context, cmd ReviewPurchaseRequestCommand) (*PurchaseRequestItem, error)
	ExportAdminPurchaseRequests(ctx context.Context, cmd ListPurchaseRequestsCommand) ([]PurchaseRequestExportRow, error)
}

// TxRunner executes a function within a database transaction.
type TxRunner interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// AuditLogger records privileged admin actions.
type AuditLogger interface {
	LogAction(ctx context.Context, actorID uuid.UUID, actorName, action, targetType string, targetID uuid.UUID, metadata map[string]interface{}, ipAddress string) error
}

type service struct {
	requestRepo    domainpayments.PurchaseRequestRepository
	userRepo       auth.UserRepository
	courseRepo     courses.CourseRepository
	bookRepo       domainbookshop.BookRepository
	enrollmentRepo enrollments.EnrollmentRepository
	orderRepo      domainbookshop.OrderRepository
	txRunner       TxRunner
	audit          AuditLogger
}

// ServiceDeps groups dependencies for the approval workflow.
type ServiceDeps struct {
	RequestRepo    domainpayments.PurchaseRequestRepository
	UserRepo       auth.UserRepository
	CourseRepo     courses.CourseRepository
	BookRepo       domainbookshop.BookRepository
	EnrollmentRepo enrollments.EnrollmentRepository
	OrderRepo      domainbookshop.OrderRepository
	TxRunner       TxRunner
	AuditLogger    AuditLogger
}

// NewService creates a new approval workflow service.
func NewService(deps ServiceDeps) Service {
	return &service{
		requestRepo:    deps.RequestRepo,
		userRepo:       deps.UserRepo,
		courseRepo:     deps.CourseRepo,
		bookRepo:       deps.BookRepo,
		enrollmentRepo: deps.EnrollmentRepo,
		orderRepo:      deps.OrderRepo,
		txRunner:       deps.TxRunner,
		audit:          deps.AuditLogger,
	}
}

// CreatePurchaseRequest stores a pending student request for admin approval.
func (s *service) CreatePurchaseRequest(ctx context.Context, cmd CreatePurchaseRequestCommand) (*PurchaseRequestItem, error) {
	user, err := s.userRepo.FindByID(ctx, cmd.StudentID)
	if err != nil || user == nil {
		return nil, apperrors.NewNotFoundError("USER_NOT_FOUND", "user not found")
	}
	if user.Role != "student" {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "only students can request approvals")
	}
	if !user.ProfileComplete {
		return nil, apperrors.ErrProfileIncomplete
	}

	itemType := cmd.ItemType
	if itemType != domainpayments.PurchaseRequestItemTypeCourse && itemType != domainpayments.PurchaseRequestItemTypeBook {
		return nil, apperrors.NewValidationError([]map[string]string{{"field": "item_type", "message": "must be course or book"}})
	}

	if _, err := s.validateItem(ctx, itemType, cmd.ItemID); err != nil {
		return nil, err
	}

	if cmd.IdempotencyKey != "" {
		if existing, _ := s.requestRepo.FindByIdempotencyKey(ctx, cmd.IdempotencyKey); existing != nil {
			if existing.StudentID == cmd.StudentID && existing.ItemID == cmd.ItemID && existing.ItemType == itemType {
				return s.toItem(ctx, existing)
			}
		}
	}

	if existing, _ := s.requestRepo.FindLatestByStudentAndItem(ctx, cmd.StudentID, cmd.ItemID, itemType); existing != nil {
		if existing.Status == domainpayments.PurchaseRequestStatusPending || existing.Status == domainpayments.PurchaseRequestStatusApproved {
			return s.toItem(ctx, existing)
		}
	}

	now := time.Now().UTC()
	var idempotencyKey *string
	if strings.TrimSpace(cmd.IdempotencyKey) != "" {
		key := strings.TrimSpace(cmd.IdempotencyKey)
		idempotencyKey = &key
	}
	request := &domainpayments.PurchaseRequest{
		ID:             uuid.New(),
		StudentID:      cmd.StudentID,
		ItemType:       itemType,
		ItemID:         cmd.ItemID,
		FileName:       strings.TrimSpace(cmd.FileName),
		IdempotencyKey: idempotencyKey,
		Status:         domainpayments.PurchaseRequestStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.requestRepo.Create(ctx, request); err != nil {
		return nil, fmt.Errorf("create purchase request: %w", err)
	}
	return s.toItem(ctx, request)
}

// GetStudentPurchaseRequests returns approval requests for the authenticated student.
func (s *service) GetStudentPurchaseRequests(ctx context.Context, cmd ListPurchaseRequestsCommand) (*PurchaseRequestListResponse, error) {
	if cmd.StudentID == nil {
		return nil, apperrors.NewValidationError([]map[string]string{{"field": "student_id", "message": "required"}})
	}
	return s.list(ctx, cmd)
}

// ListAdminPurchaseRequests returns approval requests for administrators.
func (s *service) ListAdminPurchaseRequests(ctx context.Context, cmd ListPurchaseRequestsCommand) (*PurchaseRequestListResponse, error) {
	return s.list(ctx, cmd)
}

// ApprovePurchaseRequest marks the request approved and creates the enrollment/order atomically.
func (s *service) ApprovePurchaseRequest(ctx context.Context, cmd ReviewPurchaseRequestCommand) (*PurchaseRequestItem, error) {
	request, err := s.requestRepo.FindByID(ctx, cmd.RequestID)
	if err != nil || request == nil {
		return nil, apperrors.NewNotFoundError("PURCHASE_REQUEST_NOT_FOUND", "purchase request not found")
	}
	if request.Status == domainpayments.PurchaseRequestStatusApproved {
		return s.toItem(ctx, request)
	}
	if request.Status != domainpayments.PurchaseRequestStatusPending {
		return nil, apperrors.NewSimpleValidationError("INVALID_STATUS", "request is not pending")
	}

	now := time.Now().UTC()
	reviewedBy := cmd.ActorID
	request.ReviewedBy = &reviewedBy
	request.ReviewedAt = &now
	request.Status = domainpayments.PurchaseRequestStatusApproved
	request.RejectionReason = nil

	var resultEnrollmentID *uuid.UUID
	var resultOrderID *uuid.UUID

	txErr := s.txRunner.RunInTx(ctx, func(txCtx context.Context) error {
		switch request.ItemType {
		case domainpayments.PurchaseRequestItemTypeCourse:
			existing, _ := s.enrollmentRepo.FindByStudentAndCourse(txCtx, request.StudentID, request.ItemID)
			if existing != nil {
				resultEnrollmentID = &existing.ID
				break
			}
			enrollment := &enrollments.Enrollment{
				ID:             uuid.New(),
				StudentID:      request.StudentID,
				CourseID:       request.ItemID,
				EnrollmentType: enrollments.EnrollmentTypePaid,
				Status:         enrollments.EnrollmentStatusActive,
				EnrolledAt:     now,
			}
			if err := s.enrollmentRepo.Create(txCtx, enrollment); err != nil {
				return fmt.Errorf("create enrollment: %w", err)
			}
			resultEnrollmentID = &enrollment.ID

		case domainpayments.PurchaseRequestItemTypeBook:
			existing, _ := s.orderRepo.FindNonRefundedByStudentAndBook(txCtx, request.StudentID, request.ItemID)
			if existing != nil {
				resultOrderID = &existing.ID
				break
			}
			book, err := s.bookRepo.FindByID(txCtx, request.ItemID)
			if err != nil || book == nil {
				return apperrors.NewNotFoundError("BOOK_NOT_FOUND", "book not found")
			}
			order := &domainbookshop.Order{
				ID:        uuid.New(),
				StudentID: request.StudentID,
				BookID:    request.ItemID,
				Format:    domainbookshop.OrderFormatDigital,
				Amount:    book.Price,
				Currency:  book.Currency,
				Status:    domainbookshop.OrderStatusPlaced,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if strings.TrimSpace(order.Currency) == "" {
				order.Currency = "BDT"
			}
			if err := s.orderRepo.Create(txCtx, order); err != nil {
				return fmt.Errorf("create order: %w", err)
			}
			resultOrderID = &order.ID
		}

		request.ResultEnrollmentID = resultEnrollmentID
		request.ResultOrderID = resultOrderID
		request.ReviewedBy = &reviewedBy
		request.ReviewedAt = &now
		request.UpdatedAt = now
		return s.requestRepo.Update(txCtx, request)
	})
	if txErr != nil {
		return nil, txErr
	}

	_ = s.auditAction(ctx, cmd.ActorID, cmd.ActorName, "purchase_request_approved", request.ID, cmd.IPAddress, map[string]interface{}{
		"item_type": request.ItemType,
		"item_id":   request.ItemID,
	})

	return s.toItem(ctx, request)
}

// RejectPurchaseRequest marks the request rejected with a reason.
func (s *service) RejectPurchaseRequest(ctx context.Context, cmd ReviewPurchaseRequestCommand) (*PurchaseRequestItem, error) {
	request, err := s.requestRepo.FindByID(ctx, cmd.RequestID)
	if err != nil || request == nil {
		return nil, apperrors.NewNotFoundError("PURCHASE_REQUEST_NOT_FOUND", "purchase request not found")
	}
	if request.Status == domainpayments.PurchaseRequestStatusRejected {
		return s.toItem(ctx, request)
	}
	if request.Status != domainpayments.PurchaseRequestStatusPending {
		return nil, apperrors.NewSimpleValidationError("INVALID_STATUS", "request is not pending")
	}
	reason := strings.TrimSpace(cmd.Reason)
	if reason == "" {
		return nil, apperrors.NewValidationError([]map[string]string{{"field": "reason", "message": "required"}})
	}

	now := time.Now().UTC()
	reviewedBy := cmd.ActorID
	request.Status = domainpayments.PurchaseRequestStatusRejected
	request.RejectionReason = &reason
	request.ReviewedBy = &reviewedBy
	request.ReviewedAt = &now
	request.UpdatedAt = now

	if err := s.requestRepo.Update(ctx, request); err != nil {
		return nil, fmt.Errorf("reject purchase request: %w", err)
	}

	_ = s.auditAction(ctx, cmd.ActorID, cmd.ActorName, "purchase_request_rejected", request.ID, cmd.IPAddress, map[string]interface{}{
		"item_type": request.ItemType,
		"item_id":   request.ItemID,
		"reason":    reason,
	})

	return s.toItem(ctx, request)
}

// ExportAdminPurchaseRequests returns all matching approval requests for CSV export.
func (s *service) ExportAdminPurchaseRequests(ctx context.Context, cmd ListPurchaseRequestsCommand) ([]PurchaseRequestExportRow, error) {
	requests, err := s.requestRepo.ListAll(ctx, domainpayments.PurchaseRequestFilter{
		StudentID: cmd.StudentID,
		ItemType:  cmd.ItemType,
		Status:    cmd.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("export purchase requests: %w", err)
	}

	rows := make([]PurchaseRequestExportRow, 0, len(requests))
	for _, request := range requests {
		item, err := s.toItem(ctx, request)
		if err != nil {
			return nil, err
		}
		rows = append(rows, toExportRow(*item))
	}

	return rows, nil
}

func (s *service) list(ctx context.Context, cmd ListPurchaseRequestsCommand) (*PurchaseRequestListResponse, error) {
	page := cmd.Page
	limit := cmd.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	requests, total, err := s.requestRepo.List(ctx, domainpayments.PurchaseRequestFilter{
		StudentID: cmd.StudentID,
		ItemType:  cmd.ItemType,
		Status:    cmd.Status,
	}, page, limit)
	if err != nil {
		return nil, fmt.Errorf("list purchase requests: %w", err)
	}

	items := make([]*PurchaseRequestItem, 0, len(requests))
	for _, request := range requests {
		item, err := s.toItem(ctx, request)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return &PurchaseRequestListResponse{
		Data: items,
		Meta: pagination.NewMeta(total, page, limit),
	}, nil
}

func (s *service) validateItem(ctx context.Context, itemType domainpayments.PurchaseRequestItemType, itemID uuid.UUID) (string, error) {
	switch itemType {
	case domainpayments.PurchaseRequestItemTypeCourse:
		course, err := s.courseRepo.FindByID(ctx, itemID)
		if err != nil || course == nil {
			return "", apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
		}
		return course.Title, nil
	case domainpayments.PurchaseRequestItemTypeBook:
		book, err := s.bookRepo.FindByID(ctx, itemID)
		if err != nil || book == nil || !book.IsActive {
			return "", apperrors.NewNotFoundError("BOOK_NOT_FOUND", "book not found")
		}
		return book.Title, nil
	default:
		return "", apperrors.NewValidationError([]map[string]string{{"field": "item_type", "message": "must be course or book"}})
	}
}

func (s *service) toItem(ctx context.Context, request *domainpayments.PurchaseRequest) (*PurchaseRequestItem, error) {
	user, err := s.userRepo.FindByID(ctx, request.StudentID)
	if err != nil || user == nil {
		return nil, apperrors.NewNotFoundError("USER_NOT_FOUND", "user not found")
	}

	itemTitle := ""
	itemSubtitle := ""
	switch request.ItemType {
	case domainpayments.PurchaseRequestItemTypeCourse:
		course, err := s.courseRepo.FindByID(ctx, request.ItemID)
		if err != nil || course == nil {
			return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
		}
		itemTitle = course.Title
		itemSubtitle = course.Subject
	case domainpayments.PurchaseRequestItemTypeBook:
		book, err := s.bookRepo.FindByID(ctx, request.ItemID)
		if err != nil || book == nil {
			return nil, apperrors.NewNotFoundError("BOOK_NOT_FOUND", "book not found")
		}
		itemTitle = book.Title
		itemSubtitle = book.Author
	}

	return &PurchaseRequestItem{
		ID:                 request.ID,
		StudentID:          request.StudentID,
		StudentName:        user.FullName,
		StudentEmail:       user.Email,
		ItemType:           request.ItemType,
		ItemID:             request.ItemID,
		ItemTitle:          itemTitle,
		ItemSubtitle:       itemSubtitle,
		FileName:           request.FileName,
		Status:             request.Status,
		RejectionReason:    request.RejectionReason,
		ResultEnrollmentID: request.ResultEnrollmentID,
		ResultOrderID:      request.ResultOrderID,
		ReviewedBy:         request.ReviewedBy,
		ReviewedAt:         request.ReviewedAt,
		CreatedAt:          request.CreatedAt,
		UpdatedAt:          request.UpdatedAt,
	}, nil
}

func toExportRow(item PurchaseRequestItem) PurchaseRequestExportRow {
	return PurchaseRequestExportRow{
		RequestID:          item.ID.String(),
		StudentID:          item.StudentID.String(),
		StudentName:        item.StudentName,
		StudentEmail:       item.StudentEmail,
		ItemType:           string(item.ItemType),
		ItemID:             item.ItemID.String(),
		ItemTitle:          item.ItemTitle,
		ItemSubtitle:       item.ItemSubtitle,
		FileName:           item.FileName,
		Status:             string(item.Status),
		RejectionReason:    valueOrEmpty(item.RejectionReason),
		ResultEnrollmentID: uuidToString(item.ResultEnrollmentID),
		ResultOrderID:      uuidToString(item.ResultOrderID),
		ReviewedBy:         uuidToString(item.ReviewedBy),
		ReviewedAt:         timeToString(item.ReviewedAt),
		CreatedAt:          item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:          item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func uuidToString(value *uuid.UUID) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func timeToString(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func (s *service) auditAction(ctx context.Context, actorID uuid.UUID, actorName, action string, targetID uuid.UUID, ipAddress string, metadata map[string]interface{}) error {
	if s.audit == nil {
		return nil
	}
	return s.audit.LogAction(ctx, actorID, actorName, action, "purchase_request", targetID, metadata, ipAddress)
}
