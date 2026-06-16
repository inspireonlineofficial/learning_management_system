package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	apppayments "lms-backend/internal/application/payments"
	domainpayments "lms-backend/internal/domain/payments"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// PurchaseApprovalsHandler handles HTTP requests for admin-approved purchases.
type PurchaseApprovalsHandler struct {
	service apppayments.Service
}

// NewPaymentsHandler creates a new purchase approval handler.
func NewPaymentsHandler(service apppayments.Service) *PurchaseApprovalsHandler {
	return &PurchaseApprovalsHandler{service: service}
}

// CreatePurchaseRequest handles POST /v1/purchase-requests.
func (h *PurchaseApprovalsHandler) CreatePurchaseRequest(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	var req struct {
		ItemType string `json:"item_type"`
		ItemID   string `json:"item_id"`
		FileName string `json:"file_name"`
	}
	if err := decodeJSONBody(r, &req); err != nil {
		writeErrorResponse(w, err)
		return
	}

	itemID, err := uuid.Parse(req.ItemID)
	if err != nil {
		writeErrorResponse(w, apperrors.NewValidationError([]map[string]string{{"field": "item_id", "message": "must be a valid UUID"}}))
		return
	}

	itemType := domainpayments.PurchaseRequestItemType(req.ItemType)
	if itemType != domainpayments.PurchaseRequestItemTypeCourse && itemType != domainpayments.PurchaseRequestItemTypeBook {
		writeErrorResponse(w, apperrors.NewValidationError([]map[string]string{{"field": "item_type", "message": "must be course or book"}}))
		return
	}

	cmd := apppayments.CreatePurchaseRequestCommand{
		StudentID:      userID,
		ItemType:       itemType,
		ItemID:         itemID,
		FileName:       req.FileName,
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
	}

	result, err := h.service.CreatePurchaseRequest(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, result)
}

// ListMyPurchaseRequests handles GET /v1/student/purchase-requests.
func (h *PurchaseApprovalsHandler) ListMyPurchaseRequests(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	page, limit := parsePageAndLimit(r)
	cmd := apppayments.ListPurchaseRequestsCommand{
		StudentID: &userID,
		Page:      page,
		Limit:     limit,
	}

	if itemType := r.URL.Query().Get("item_type"); itemType != "" {
		it := domainpayments.PurchaseRequestItemType(itemType)
		if it != domainpayments.PurchaseRequestItemTypeCourse && it != domainpayments.PurchaseRequestItemTypeBook {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ITEM_TYPE", "item_type must be course or book"))
			return
		}
		cmd.ItemType = &it
	}

	if status := r.URL.Query().Get("status"); status != "" {
		st := domainpayments.PurchaseRequestStatus(status)
		if st != domainpayments.PurchaseRequestStatusPending && st != domainpayments.PurchaseRequestStatusApproved && st != domainpayments.PurchaseRequestStatusRejected {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_STATUS", "status must be pending, approved, or rejected"))
			return
		}
		cmd.Status = &st
	}

	result, err := h.service.GetStudentPurchaseRequests(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// ListAdminPurchaseRequests handles GET /v1/admin/purchase-requests.
func (h *PurchaseApprovalsHandler) ListAdminPurchaseRequests(w http.ResponseWriter, r *http.Request) {
	page, limit := parsePageAndLimit(r)
	cmd, err := parsePurchaseRequestFilters(r, page, limit)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	result, err := h.service.ListAdminPurchaseRequests(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// ExportAdminPurchaseRequests handles GET /v1/admin/purchase-requests/export.
func (h *PurchaseApprovalsHandler) ExportAdminPurchaseRequests(w http.ResponseWriter, r *http.Request) {
	cmd, err := parsePurchaseRequestFilters(r, 1, 0)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	rows, err := h.service.ExportAdminPurchaseRequests(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="purchase-requests.csv"`)
	w.WriteHeader(http.StatusOK)

	writer := csv.NewWriter(w)
	_ = writer.Write([]string{
		"request_id", "student_id", "student_name", "student_email", "item_type", "item_id", "item_title", "item_subtitle", "file_name", "status", "rejection_reason", "result_enrollment_id", "result_order_id", "reviewed_by", "reviewed_at", "created_at", "updated_at",
	})
	for _, row := range rows {
		_ = writer.Write([]string{
			row.RequestID, row.StudentID, row.StudentName, row.StudentEmail, row.ItemType, row.ItemID, row.ItemTitle, row.ItemSubtitle, row.FileName, row.Status, row.RejectionReason, row.ResultEnrollmentID, row.ResultOrderID, row.ReviewedBy, row.ReviewedAt, row.CreatedAt, row.UpdatedAt,
		})
	}
	writer.Flush()
}

// ApprovePurchaseRequest handles POST /v1/admin/purchase-requests/{requestId}/approve.
func (h *PurchaseApprovalsHandler) ApprovePurchaseRequest(w http.ResponseWriter, r *http.Request) {
	actorID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	requestID, err := parseUUIDParam(r, "requestId")
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	cmd := apppayments.ReviewPurchaseRequestCommand{
		RequestID: requestID,
		ActorID:   actorID,
		IPAddress: requestIP(r),
	}

	result, err := h.service.ApprovePurchaseRequest(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// RejectPurchaseRequest handles POST /v1/admin/purchase-requests/{requestId}/reject.
func (h *PurchaseApprovalsHandler) RejectPurchaseRequest(w http.ResponseWriter, r *http.Request) {
	actorID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	requestID, err := parseUUIDParam(r, "requestId")
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSONBody(r, &req); err != nil {
		writeErrorResponse(w, err)
		return
	}

	cmd := apppayments.ReviewPurchaseRequestCommand{
		RequestID: requestID,
		ActorID:   actorID,
		Reason:    req.Reason,
		IPAddress: requestIP(r),
	}

	result, err := h.service.RejectPurchaseRequest(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

func parsePageAndLimit(r *http.Request) (int, int) {
	page := 1
	limit := 20
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	return page, limit
}

func parsePurchaseRequestFilters(r *http.Request, page, limit int) (apppayments.ListPurchaseRequestsCommand, error) {
	cmd := apppayments.ListPurchaseRequestsCommand{Page: page, Limit: limit}
	q := r.URL.Query()

	if v := q.Get("student_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return cmd, apperrors.NewSimpleValidationError("INVALID_STUDENT_ID", "student_id must be a valid UUID")
		}
		cmd.StudentID = &id
	}
	if v := q.Get("item_type"); v != "" {
		it := domainpayments.PurchaseRequestItemType(v)
		if it != domainpayments.PurchaseRequestItemTypeCourse && it != domainpayments.PurchaseRequestItemTypeBook {
			return cmd, apperrors.NewSimpleValidationError("INVALID_ITEM_TYPE", "item_type must be course or book")
		}
		cmd.ItemType = &it
	}
	if v := q.Get("status"); v != "" {
		st := domainpayments.PurchaseRequestStatus(v)
		if st != domainpayments.PurchaseRequestStatusPending && st != domainpayments.PurchaseRequestStatusApproved && st != domainpayments.PurchaseRequestStatusRejected {
			return cmd, apperrors.NewSimpleValidationError("INVALID_STATUS", "status must be pending, approved, or rejected")
		}
		cmd.Status = &st
	}

	return cmd, nil
}

// ListApprovalsCompatibility handles GET /v1/admin/approvals.
func (h *PurchaseApprovalsHandler) ListApprovalsCompatibility(w http.ResponseWriter, r *http.Request) {
	status := domainpayments.PurchaseRequestStatusPending
	cmd := apppayments.ListPurchaseRequestsCommand{
		Page:   1,
		Limit:  1000,
		Status: &status,
	}

	result, err := h.service.ListAdminPurchaseRequests(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	type requester struct {
		ID       string `json:"id"`
		FullName string `json:"full_name"`
	}
	type approvalItem struct {
		ID             string    `json:"id"`
		Kind           string    `json:"kind"` // "course_publish" | "book_purchase"
		Requester      requester `json:"requester"`
		PayloadSummary string    `json:"payload_summary"`
		CreatedAt      string    `json:"created_at"`
	}

	items := make([]approvalItem, 0, len(result.Data))
	for _, item := range result.Data {
		kind := "book_purchase"
		if item.ItemType == domainpayments.PurchaseRequestItemTypeCourse {
			kind = "course_publish"
		}
		items = append(items, approvalItem{
			ID:   item.ID.String(),
			Kind: kind,
			Requester: requester{
				ID:       item.StudentID.String(),
				FullName: item.StudentName,
			},
			PayloadSummary: item.ItemTitle,
			CreatedAt:      item.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{"items": items})
}

// ApproveApprovalCompatibility handles POST /v1/admin/approvals/{id}/approve.
func (h *PurchaseApprovalsHandler) ApproveApprovalCompatibility(w http.ResponseWriter, r *http.Request) {
	actorID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	requestID, err := parseUUIDParam(r, "id")
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	cmd := apppayments.ReviewPurchaseRequestCommand{
		RequestID: requestID,
		ActorID:   actorID,
		IPAddress: requestIP(r),
	}

	_, err = h.service.ApprovePurchaseRequest(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// RejectApprovalCompatibility handles POST /v1/admin/approvals/{id}/reject.
func (h *PurchaseApprovalsHandler) RejectApprovalCompatibility(w http.ResponseWriter, r *http.Request) {
	actorID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	requestID, err := parseUUIDParam(r, "id")
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	var req struct {
		Note string `json:"note"`
	}
	if err := decodeJSONBody(r, &req); err != nil {
		writeErrorResponse(w, err)
		return
	}

	reason := req.Note
	if len(reason) == 0 {
		reason = "Rejected by administrator"
	}

	cmd := apppayments.ReviewPurchaseRequestCommand{
		RequestID: requestID,
		ActorID:   actorID,
		Reason:    reason,
		IPAddress: requestIP(r),
	}

	_, err = h.service.RejectPurchaseRequest(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// CreatePurchaseRequestCompatibility handles POST /v1/student/requests.
func (h *PurchaseApprovalsHandler) CreatePurchaseRequestCompatibility(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	var req struct {
		ItemType string `json:"item_type"` // "book" or "course"
		ItemID   string `json:"item_id"`
		Note     string `json:"note"`
	}
	if err := decodeJSONBody(r, &req); err != nil {
		writeErrorResponse(w, err)
		return
	}

	itemID, err := uuid.Parse(req.ItemID)
	if err != nil {
		writeErrorResponse(w, apperrors.NewValidationError([]map[string]string{{"field": "item_id", "message": "must be a valid UUID"}}))
		return
	}

	itemType := domainpayments.PurchaseRequestItemType(req.ItemType)
	if itemType != domainpayments.PurchaseRequestItemTypeCourse && itemType != domainpayments.PurchaseRequestItemTypeBook {
		writeErrorResponse(w, apperrors.NewValidationError([]map[string]string{{"field": "item_type", "message": "must be course or book"}}))
		return
	}

	cmd := apppayments.CreatePurchaseRequestCommand{
		StudentID:      userID,
		ItemType:       itemType,
		ItemID:         itemID,
		FileName:       req.Note,
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
	}

	result, err := h.service.CreatePurchaseRequest(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, map[string]interface{}{"id": result.ID.String()})
}

// ListMyRequestsCompatibility handles GET /v1/student/requests.
func (h *PurchaseApprovalsHandler) ListMyRequestsCompatibility(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	cmd := apppayments.ListPurchaseRequestsCommand{
		StudentID: &userID,
		Page:      1,
		Limit:     1000,
	}

	result, err := h.service.GetStudentPurchaseRequests(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	type requester struct {
		ID       string `json:"id"`
		FullName string `json:"full_name"`
	}
	type approvalItem struct {
		ID             string    `json:"id"`
		Kind           string    `json:"kind"` // "course_publish" | "book_purchase"
		Requester      requester `json:"requester"`
		PayloadSummary string    `json:"payload_summary"`
		CreatedAt      string    `json:"created_at"`
	}

	items := make([]approvalItem, 0, len(result.Data))
	for _, item := range result.Data {
		kind := "book_purchase"
		if item.ItemType == domainpayments.PurchaseRequestItemTypeCourse {
			kind = "course_publish"
		}
		items = append(items, approvalItem{
			ID:   item.ID.String(),
			Kind: kind,
			Requester: requester{
				ID:       item.StudentID.String(),
				FullName: item.StudentName,
			},
			PayloadSummary: fmt.Sprintf("%s (%s)", item.ItemTitle, item.Status),
			CreatedAt:      item.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{"items": items})
}
