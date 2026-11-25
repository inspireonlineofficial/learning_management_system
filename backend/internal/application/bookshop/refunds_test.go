package bookshop

import (
	"context"
	"strings"
	"testing"
	"time"

	domainbookshop "lms-backend/internal/domain/bookshop"

	"github.com/google/uuid"
)

type mockRefundRepo struct {
	requests map[uuid.UUID]*domainbookshop.RefundRequest
	refunds  map[uuid.UUID]*domainbookshop.OrderRefund
}

func newMockRefundRepo() *mockRefundRepo {
	return &mockRefundRepo{requests: map[uuid.UUID]*domainbookshop.RefundRequest{}, refunds: map[uuid.UUID]*domainbookshop.OrderRefund{}}
}

func (m *mockRefundRepo) CreateRequest(_ context.Context, r *domainbookshop.RefundRequest) error {
	for _, x := range m.requests {
		if x.OrderID == r.OrderID && x.Status == domainbookshop.RefundRequestPending {
			return domainbookshop.ErrRefundRequestExists
		}
	}
	c := *r
	m.requests[r.ID] = &c
	return nil
}
func (m *mockRefundRepo) FindRequestByID(_ context.Context, id uuid.UUID) (*domainbookshop.RefundRequest, error) {
	if r, ok := m.requests[id]; ok {
		c := *r
		return &c, nil
	}
	return nil, nil
}
func (m *mockRefundRepo) LockRequest(ctx context.Context, id uuid.UUID) (*domainbookshop.RefundRequest, error) {
	return m.FindRequestByID(ctx, id)
}
func (m *mockRefundRepo) FindPendingRequestByOrder(_ context.Context, orderID uuid.UUID) (*domainbookshop.RefundRequest, error) {
	for _, r := range m.requests {
		if r.OrderID == orderID && r.Status == domainbookshop.RefundRequestPending {
			c := *r
			return &c, nil
		}
	}
	return nil, nil
}
func (m *mockRefundRepo) LatestRequestByOrder(_ context.Context, orderID uuid.UUID) (*domainbookshop.RefundRequest, error) {
	var latest *domainbookshop.RefundRequest
	for _, r := range m.requests {
		if r.OrderID == orderID && (latest == nil || r.CreatedAt.After(latest.CreatedAt)) {
			latest = r
		}
	}
	return latest, nil
}
func (m *mockRefundRepo) UpdateRequest(_ context.Context, r *domainbookshop.RefundRequest) error {
	c := *r
	m.requests[r.ID] = &c
	return nil
}
func (m *mockRefundRepo) CountRequestsForOrder(_ context.Context, orderID uuid.UUID) (int, error) {
	n := 0
	for _, r := range m.requests {
		if r.OrderID == orderID {
			n++
		}
	}
	return n, nil
}
func (m *mockRefundRepo) ListRequests(_ context.Context, status domainbookshop.RefundRequestStatus, studentID *uuid.UUID, _, _ int) ([]*domainbookshop.RefundRequest, int, error) {
	var out []*domainbookshop.RefundRequest
	for _, r := range m.requests {
		if (status == "" || r.Status == status) && (studentID == nil || r.StudentID == *studentID) {
			out = append(out, r)
		}
	}
	return out, len(out), nil
}
func (m *mockRefundRepo) CreateRefund(_ context.Context, r *domainbookshop.OrderRefund) error {
	m.refunds[r.OrderID] = r
	return nil
}
func (m *mockRefundRepo) FindRefundByOrder(_ context.Context, orderID uuid.UUID) (*domainbookshop.OrderRefund, error) {
	return m.refunds[orderID], nil
}

func refundSetup(t *testing.T) (*service, *bookshopPropDeps, *mockRefundRepo) {
	t.Helper()
	d := newBookshopPropDeps(map[uuid.UUID]int{})
	svc := d.service().(*service)
	repo := newMockRefundRepo()
	svc.SetRefundRepository(repo)
	return svc, d, repo
}

func paidDigitalOrder(d *bookshopPropDeps, student uuid.UUID, age time.Duration) *domainbookshop.Order {
	o := &domainbookshop.Order{ID: uuid.New(), StudentID: student, BookID: uuid.New(), Format: domainbookshop.OrderFormatDigital,
		Amount: 350, Currency: "BDT", Status: domainbookshop.OrderStatusPlaced, CreatedAt: time.Now().Add(-age), BookTitle: "Guide"}
	d.orderRepo.orders[o.ID] = o
	return o
}

func TestRefundRequest_Eligibility(t *testing.T) {
	svc, d, _ := refundSetup(t)
	student := uuid.New()
	ctx := context.Background()

	free := paidDigitalOrder(d, student, time.Hour)
	free.Amount = 0
	if _, err := svc.RequestRefund(ctx, RequestRefundCommand{StudentID: student, OrderID: free.ID, Reason: "x"}); bookshopErrCode(err) != "REFUND_NOT_ALLOWED" {
		t.Fatalf("free order: %v", err)
	}
	old := paidDigitalOrder(d, student, 8*24*time.Hour)
	if _, err := svc.RequestRefund(ctx, RequestRefundCommand{StudentID: student, OrderID: old.ID, Reason: "x"}); bookshopErrCode(err) != "REFUND_NOT_ALLOWED" {
		t.Fatalf("outside window: %v", err)
	}
	o := paidDigitalOrder(d, student, time.Hour)
	if _, err := svc.RequestRefund(ctx, RequestRefundCommand{StudentID: uuid.New(), OrderID: o.ID, Reason: "x"}); bookshopErrCode(err) != "ORDER_NOT_FOUND" {
		t.Fatalf("another student's order must look missing: %v", err)
	}
	if _, err := svc.RequestRefund(ctx, RequestRefundCommand{StudentID: student, OrderID: o.ID, Reason: "  "}); err == nil {
		t.Fatal("empty reason accepted")
	}
	if _, err := svc.RequestRefund(ctx, RequestRefundCommand{StudentID: student, OrderID: o.ID, Reason: "Wrong edition"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RequestRefund(ctx, RequestRefundCommand{StudentID: student, OrderID: o.ID, Reason: "again"}); bookshopErrCode(err) != "REFUND_NOT_ALLOWED" {
		t.Fatalf("second open request: %v", err)
	}
	// Printed and not yet delivered: no deadline yet.
	p := paidDigitalOrder(d, student, 30*24*time.Hour)
	p.Format = domainbookshop.OrderFormatPhysical
	if ok, deadline, why := svc.refundEligibility(ctx, p); !ok || deadline != nil {
		t.Fatalf("undelivered printed order should be refundable: %v %v %s", ok, deadline, why)
	}
}

func TestRefundRequest_ApproveRecordsPayout(t *testing.T) {
	svc, d, repo := refundSetup(t)
	student := uuid.New()
	o := paidDigitalOrder(d, student, time.Hour)
	req, err := svc.RequestRefund(context.Background(), RequestRefundCommand{StudentID: student, OrderID: o.ID, Reason: "Duplicate purchase"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ApproveRefundRequest(context.Background(), ReviewRefundRequestCommand{RequestID: req.ID, ActorID: uuid.New(), PayoutMethod: "bkash"}); err == nil {
		t.Fatal("bKash refund without a transaction ID must be rejected")
	}
	over := 999.0
	if _, err := svc.ApproveRefundRequest(context.Background(), ReviewRefundRequestCommand{RequestID: req.ID, ActorID: uuid.New(), PayoutMethod: "bkash", PayoutReference: "T1", Amount: &over}); err == nil {
		t.Fatal("refund larger than the order must be rejected")
	}
	part := 200.0
	res, err := svc.ApproveRefundRequest(context.Background(), ReviewRefundRequestCommand{RequestID: req.ID, ActorID: uuid.New(), PayoutMethod: "bkash", PayoutReference: "8N7A6B5C", Amount: &part})
	if err != nil {
		t.Fatal(err)
	}
	if res.Refund == nil || res.Refund.Amount != 200 || *res.Refund.PayoutReference != "8N7A6B5C" {
		t.Fatalf("payout not recorded: %+v", res.Refund)
	}
	if repo.requests[req.ID].Status != domainbookshop.RefundRequestApproved || repo.refunds[o.ID].RefundRequestID == nil {
		t.Fatal("request must be approved and linked to the payout")
	}
	if d.orderRepo.orders[o.ID].Status != domainbookshop.OrderStatusRefunded {
		t.Fatal("order must be refunded")
	}
	if _, err := svc.ApproveRefundRequest(context.Background(), ReviewRefundRequestCommand{RequestID: req.ID, ActorID: uuid.New(), PayoutMethod: "cash"}); bookshopErrCode(err) != "REFUND_REQUEST_DECIDED" {
		t.Fatalf("second approval: %v", err)
	}
}

func TestRefundRequest_RejectNeedsNoteAndNotifies(t *testing.T) {
	svc, d, repo := refundSetup(t)
	student := uuid.New()
	o := paidDigitalOrder(d, student, time.Hour)
	req, _ := svc.RequestRefund(context.Background(), RequestRefundCommand{StudentID: student, OrderID: o.ID, Reason: "Changed my mind"})
	if _, err := svc.RejectRefundRequest(context.Background(), ReviewRefundRequestCommand{RequestID: req.ID, ActorID: uuid.New()}); err == nil {
		t.Fatal("reject without a note accepted")
	}
	if _, err := svc.RejectRefundRequest(context.Background(), ReviewRefundRequestCommand{RequestID: req.ID, ActorID: uuid.New(), Note: "The book was read past the sample."}); err != nil {
		t.Fatal(err)
	}
	if repo.requests[req.ID].Status != domainbookshop.RefundRequestRejected || d.orderRepo.orders[o.ID].Status != domainbookshop.OrderStatusPlaced {
		t.Fatal("reject must not refund the order")
	}
	types := []string{}
	for _, j := range d.jobQueue.jobs {
		types = append(types, j.Type)
	}
	// The student is told twice (received, rejected); admins are alerted once.
	want := []string{"refund_request_received", "refund_request_admin_alert", "refund_request_rejected"}
	if strings.Join(types, ",") != strings.Join(want, ",") {
		t.Fatalf("expected %v, got %v", want, types)
	}
}

func TestDirectRefund_FreeOrderNeedsNoPayout(t *testing.T) {
	svc, d, repo := refundSetup(t)
	o := paidDigitalOrder(d, uuid.New(), time.Hour)
	o.Amount = 0
	if _, err := svc.ProcessRefund(context.Background(), ProcessRefundCommand{OrderID: o.ID, ActorID: uuid.New()}); err != nil {
		t.Fatal(err)
	}
	if repo.refunds[o.ID].PayoutMethod != domainbookshop.PayoutNone {
		t.Fatalf("free order payout should be 'none', got %q", repo.refunds[o.ID].PayoutMethod)
	}
}
