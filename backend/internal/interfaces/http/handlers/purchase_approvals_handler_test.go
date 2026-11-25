package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apppayments "lms-backend/internal/application/payments"
	domainpayments "lms-backend/internal/domain/payments"

	"github.com/google/uuid"
)

type recordingPurchaseService struct {
	createResult  *apppayments.PurchaseRequestItem
	approveResult *apppayments.PurchaseRequestItem
	rejectResult  *apppayments.PurchaseRequestItem
	listResult    *apppayments.PurchaseRequestListResponse
	exportRows    []apppayments.PurchaseRequestExportRow

	lastCreateCommand apppayments.CreatePurchaseRequestCommand
	lastReviewCommand apppayments.ReviewPurchaseRequestCommand
	lastListCommand   apppayments.ListPurchaseRequestsCommand
	createCalled      int
	approveCalled     int
	rejectCalled      int
	exportCalled      int
}

func (m *recordingPurchaseService) CreatePurchaseRequest(_ context.Context, cmd apppayments.CreatePurchaseRequestCommand) (*apppayments.PurchaseRequestItem, error) {
	m.createCalled++
	m.lastCreateCommand = cmd
	if m.createResult != nil {
		return m.createResult, nil
	}
	return &apppayments.PurchaseRequestItem{ID: uuid.New(), Status: domainpayments.PurchaseRequestStatusPending}, nil
}

func (m *recordingPurchaseService) GetStudentPurchaseRequests(_ context.Context, cmd apppayments.ListPurchaseRequestsCommand) (*apppayments.PurchaseRequestListResponse, error) {
	m.lastListCommand = cmd
	if m.listResult != nil {
		return m.listResult, nil
	}
	return &apppayments.PurchaseRequestListResponse{}, nil
}

func (m *recordingPurchaseService) ListAdminPurchaseRequests(_ context.Context, cmd apppayments.ListPurchaseRequestsCommand) (*apppayments.PurchaseRequestListResponse, error) {
	m.lastListCommand = cmd
	if m.listResult != nil {
		return m.listResult, nil
	}
	return &apppayments.PurchaseRequestListResponse{}, nil
}

func (m *recordingPurchaseService) ApprovePurchaseRequest(_ context.Context, cmd apppayments.ReviewPurchaseRequestCommand) (*apppayments.PurchaseRequestItem, error) {
	m.approveCalled++
	m.lastReviewCommand = cmd
	if m.approveResult != nil {
		return m.approveResult, nil
	}
	return &apppayments.PurchaseRequestItem{ID: cmd.RequestID, Status: domainpayments.PurchaseRequestStatusApproved}, nil
}

func (m *recordingPurchaseService) RejectPurchaseRequest(_ context.Context, cmd apppayments.ReviewPurchaseRequestCommand) (*apppayments.PurchaseRequestItem, error) {
	m.rejectCalled++
	m.lastReviewCommand = cmd
	if m.rejectResult != nil {
		return m.rejectResult, nil
	}
	return &apppayments.PurchaseRequestItem{ID: cmd.RequestID, Status: domainpayments.PurchaseRequestStatusRejected}, nil
}

func (m *recordingPurchaseService) ExportAdminPurchaseRequests(_ context.Context, cmd apppayments.ListPurchaseRequestsCommand) ([]apppayments.PurchaseRequestExportRow, error) {
	m.exportCalled++
	m.lastListCommand = cmd
	return m.exportRows, nil
}

func userContextRequest(method, target string, body string, userID uuid.UUID) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	return req.WithContext(context.WithValue(req.Context(), "user_id", userID))
}

func TestCreatePurchaseRequestHandler_SubmitsApprovalRequest(t *testing.T) {
	service := &recordingPurchaseService{}
	h := NewPaymentsHandler(service)

	studentID := uuid.New()
	body := `{"item_type":"course","item_id":"11111111-1111-1111-1111-111111111111","file_name":"supporting-note.pdf"}`
	req := userContextRequest(http.MethodPost, "/v1/purchase-requests", body, studentID)
	rr := httptest.NewRecorder()

	h.CreatePurchaseRequest(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
	if service.createCalled != 1 {
		t.Fatalf("expected service to be called once, got %d", service.createCalled)
	}
	if service.lastCreateCommand.StudentID != studentID {
		t.Fatalf("expected student id %s, got %s", studentID, service.lastCreateCommand.StudentID)
	}
	if service.lastCreateCommand.FileName != "supporting-note.pdf" {
		t.Fatalf("expected file name to be forwarded, got %q", service.lastCreateCommand.FileName)
	}
}

func TestCreatePurchaseRequestHandler_RejectsInvalidBody(t *testing.T) {
	service := &recordingPurchaseService{}
	h := NewPaymentsHandler(service)

	req := userContextRequest(http.MethodPost, "/v1/purchase-requests", `{"item_id":"11111111-1111-1111-1111-111111111111"}`, uuid.New())
	rr := httptest.NewRecorder()

	h.CreatePurchaseRequest(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if service.createCalled != 0 {
		t.Fatalf("expected service not to be called, got %d", service.createCalled)
	}
}

func TestApproveAndRejectHandlers(t *testing.T) {
	service := &recordingPurchaseService{}
	h := NewPaymentsHandler(service)
	requestID := uuid.New()
	adminID := uuid.New()

	approveReq := userContextRequest(http.MethodPost, "/v1/admin/purchase-requests/"+requestID.String()+"/approve", "", adminID)
	approveReq.SetPathValue("requestId", requestID.String())
	approveRR := httptest.NewRecorder()
	h.ApprovePurchaseRequest(approveRR, approveReq)
	if approveRR.Code != http.StatusOK {
		t.Fatalf("expected approve status 200, got %d", approveRR.Code)
	}
	if service.approveCalled != 1 {
		t.Fatalf("expected approve service to be called once, got %d", service.approveCalled)
	}
	if service.lastReviewCommand.RequestID != requestID {
		t.Fatalf("expected request id %s, got %s", requestID, service.lastReviewCommand.RequestID)
	}

	rejectBody := `{"reason":"missing supporting documents"}`
	rejectReq := userContextRequest(http.MethodPost, "/v1/admin/purchase-requests/"+requestID.String()+"/reject", rejectBody, adminID)
	rejectReq.SetPathValue("requestId", requestID.String())
	rejectRR := httptest.NewRecorder()
	h.RejectPurchaseRequest(rejectRR, rejectReq)
	if rejectRR.Code != http.StatusOK {
		t.Fatalf("expected reject status 200, got %d", rejectRR.Code)
	}
	if service.rejectCalled != 1 {
		t.Fatalf("expected reject service to be called once, got %d", service.rejectCalled)
	}
	if service.lastReviewCommand.Reason != "missing supporting documents" {
		t.Fatalf("expected reason to be forwarded, got %q", service.lastReviewCommand.Reason)
	}
}

func TestExportAdminPurchaseRequestsHandler_WritesCSV(t *testing.T) {
	service := &recordingPurchaseService{
		exportRows: []apppayments.PurchaseRequestExportRow{
			{
				RequestID:    "req-1",
				StudentID:    "student-1",
				StudentName:  "Student One",
				StudentEmail: "student@example.com",
				ItemType:     "course",
				ItemID:       "item-1",
				ItemTitle:    "Biology 101",
				FileName:     "supporting-note.pdf",
				Status:       "pending",
				CreatedAt:    "2026-06-14T12:00:00Z",
				UpdatedAt:    "2026-06-14T12:05:00Z",
			},
		},
	}
	h := NewPaymentsHandler(service)

	req := userContextRequest(http.MethodGet, "/v1/admin/purchase-requests/export", "", uuid.New())
	rr := httptest.NewRecorder()

	h.ExportAdminPurchaseRequests(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Fatalf("expected csv content type, got %q", ct)
	}
	if !strings.Contains(rr.Body.String(), "request_id,student_id,student_name") {
		t.Fatalf("expected csv header, got %s", rr.Body.String())
	}
	if service.exportCalled != 1 {
		t.Fatalf("expected export to be called once, got %d", service.exportCalled)
	}
}
