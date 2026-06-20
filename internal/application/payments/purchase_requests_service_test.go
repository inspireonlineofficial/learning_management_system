package payments

import (
	"context"
	"sort"
	"testing"
	"time"

	"lms-backend/internal/domain/auth"
	domainbookshop "lms-backend/internal/domain/bookshop"
	"lms-backend/internal/domain/courses"
	"lms-backend/internal/domain/enrollments"
	domainpayments "lms-backend/internal/domain/payments"

	"github.com/google/uuid"
)

type mockUserRepo struct {
	users map[uuid.UUID]*auth.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[uuid.UUID]*auth.User)}
}

func (m *mockUserRepo) Create(_ context.Context, u *auth.User) error {
	m.users[u.ID] = u
	return nil
}

func (m *mockUserRepo) FindByEmail(_ context.Context, email string) (*auth.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) FindByUsername(_ context.Context, username string) (*auth.User, error) {
	for _, user := range m.users {
		if user.Username != nil && *user.Username == username {
			return user, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) FindByID(_ context.Context, id uuid.UUID) (*auth.User, error) {
	return m.users[id], nil
}

func (m *mockUserRepo) Update(_ context.Context, u *auth.User) error {
	m.users[u.ID] = u
	return nil
}

func (m *mockUserRepo) SoftDelete(_ context.Context, id uuid.UUID) error {
	delete(m.users, id)
	return nil
}

type mockCourseRepo struct {
	courses map[uuid.UUID]*courses.Course
}

func newMockCourseRepo() *mockCourseRepo {
	return &mockCourseRepo{courses: make(map[uuid.UUID]*courses.Course)}
}

func (m *mockCourseRepo) Create(_ context.Context, course *courses.Course) error {
	m.courses[course.ID] = course
	return nil
}

func (m *mockCourseRepo) FindByID(_ context.Context, id uuid.UUID) (*courses.Course, error) {
	return m.courses[id], nil
}

func (m *mockCourseRepo) FindBySlug(_ context.Context, slug string) (*courses.Course, error) {
	for _, course := range m.courses {
		if course.Slug == slug {
			return course, nil
		}
	}
	return nil, nil
}

func (m *mockCourseRepo) FindByTeacherID(_ context.Context, teacherID uuid.UUID, page, limit int) ([]*courses.Course, int, error) {
	return nil, 0, nil
}

func (m *mockCourseRepo) Update(_ context.Context, course *courses.Course) error {
	m.courses[course.ID] = course
	return nil
}

func (m *mockCourseRepo) SoftDelete(_ context.Context, id uuid.UUID) error { return nil }

func (m *mockCourseRepo) List(_ context.Context, _ courses.CourseFilters, _ int, _ int) ([]*courses.Course, int, error) {
	return nil, 0, nil
}

func (m *mockCourseRepo) CountPublishedLessons(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

type mockBookRepo struct {
	books map[uuid.UUID]*domainbookshop.Book
}

func newMockBookRepo() *mockBookRepo {
	return &mockBookRepo{books: make(map[uuid.UUID]*domainbookshop.Book)}
}

func (m *mockBookRepo) Create(_ context.Context, book *domainbookshop.Book) error {
	m.books[book.ID] = book
	return nil
}

func (m *mockBookRepo) FindByID(_ context.Context, id uuid.UUID) (*domainbookshop.Book, error) {
	return m.books[id], nil
}

func (m *mockBookRepo) Update(_ context.Context, book *domainbookshop.Book) error {
	m.books[book.ID] = book
	return nil
}

func (m *mockBookRepo) List(_ context.Context, _ domainbookshop.BookFilter, _ int, _ int) ([]*domainbookshop.Book, int, error) {
	return nil, 0, nil
}

type mockEnrollmentRepo struct {
	enrollments map[string]*enrollments.Enrollment
}

func newMockEnrollmentRepo() *mockEnrollmentRepo {
	return &mockEnrollmentRepo{enrollments: make(map[string]*enrollments.Enrollment)}
}

func (m *mockEnrollmentRepo) Create(_ context.Context, enrollment *enrollments.Enrollment) error {
	m.enrollments[enrollment.StudentID.String()+":"+enrollment.CourseID.String()] = enrollment
	return nil
}

func (m *mockEnrollmentRepo) FindByID(_ context.Context, id uuid.UUID) (*enrollments.Enrollment, error) {
	for _, enrollment := range m.enrollments {
		if enrollment.ID == id {
			return enrollment, nil
		}
	}
	return nil, nil
}

func (m *mockEnrollmentRepo) FindByStudentAndCourse(_ context.Context, studentID, courseID uuid.UUID) (*enrollments.Enrollment, error) {
	return m.enrollments[studentID.String()+":"+courseID.String()], nil
}

func (m *mockEnrollmentRepo) FindByStudentID(_ context.Context, _ uuid.UUID, _ int, _ int) ([]*enrollments.Enrollment, int, error) {
	return nil, 0, nil
}

func (m *mockEnrollmentRepo) FindByCourseID(_ context.Context, _ uuid.UUID, _ int, _ int) ([]*enrollments.Enrollment, int, error) {
	return nil, 0, nil
}

func (m *mockEnrollmentRepo) Update(_ context.Context, enrollment *enrollments.Enrollment) error {
	m.enrollments[enrollment.StudentID.String()+":"+enrollment.CourseID.String()] = enrollment
	return nil
}

func (m *mockEnrollmentRepo) UpdateProgressPercent(_ context.Context, _ uuid.UUID, _ float64) error {
	return nil
}

func (m *mockEnrollmentRepo) RecalculateProgressPercent(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockEnrollmentRepo) CountTotalLessons(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockEnrollmentRepo) Exists(_ context.Context, studentID, courseID uuid.UUID) (bool, error) {
	_, ok := m.enrollments[studentID.String()+":"+courseID.String()]
	return ok, nil
}

type mockOrderRepo struct {
	orders map[string]*domainbookshop.Order
}

func newMockOrderRepo() *mockOrderRepo {
	return &mockOrderRepo{orders: make(map[string]*domainbookshop.Order)}
}

func (m *mockOrderRepo) Create(_ context.Context, order *domainbookshop.Order) error {
	m.orders[order.StudentID.String()+":"+order.BookID.String()] = order
	return nil
}

func (m *mockOrderRepo) FindByID(_ context.Context, id uuid.UUID) (*domainbookshop.Order, error) {
	for _, order := range m.orders {
		if order.ID == id {
			return order, nil
		}
	}
	return nil, nil
}

func (m *mockOrderRepo) FindByIdempotencyKey(_ context.Context, _ string) (*domainbookshop.Order, error) {
	return nil, nil
}

func (m *mockOrderRepo) Update(_ context.Context, order *domainbookshop.Order) error {
	m.orders[order.StudentID.String()+":"+order.BookID.String()] = order
	return nil
}

func (m *mockOrderRepo) FindByStudentID(_ context.Context, _ uuid.UUID, _ int, _ int) ([]*domainbookshop.Order, int, error) {
	return nil, 0, nil
}

func (m *mockOrderRepo) FindNonRefundedByStudentAndBook(_ context.Context, studentID, bookID uuid.UUID) (*domainbookshop.Order, error) {
	return m.orders[studentID.String()+":"+bookID.String()], nil
}

func (m *mockOrderRepo) DecrementPhysicalStock(_ context.Context, _ uuid.UUID) error { return nil }

func (m *mockOrderRepo) IncrementPhysicalStock(_ context.Context, _ uuid.UUID) error { return nil }

type mockRequestRepo struct {
	requests map[uuid.UUID]*domainpayments.PurchaseRequest
}

func newMockRequestRepo() *mockRequestRepo {
	return &mockRequestRepo{requests: make(map[uuid.UUID]*domainpayments.PurchaseRequest)}
}

func (m *mockRequestRepo) Create(_ context.Context, request *domainpayments.PurchaseRequest) error {
	m.requests[request.ID] = request
	return nil
}

func (m *mockRequestRepo) FindByID(_ context.Context, id uuid.UUID) (*domainpayments.PurchaseRequest, error) {
	return m.requests[id], nil
}

func (m *mockRequestRepo) FindByIdempotencyKey(_ context.Context, key string) (*domainpayments.PurchaseRequest, error) {
	for _, request := range m.requests {
		if request.IdempotencyKey != nil && *request.IdempotencyKey == key {
			return request, nil
		}
	}
	return nil, nil
}

func (m *mockRequestRepo) FindLatestByStudentAndItem(_ context.Context, studentID, itemID uuid.UUID, itemType domainpayments.PurchaseRequestItemType) (*domainpayments.PurchaseRequest, error) {
	var latest *domainpayments.PurchaseRequest
	for _, request := range m.requests {
		if request.StudentID == studentID && request.ItemID == itemID && request.ItemType == itemType {
			if latest == nil || request.CreatedAt.After(latest.CreatedAt) {
				latest = request
			}
		}
	}
	return latest, nil
}

func (m *mockRequestRepo) Update(_ context.Context, request *domainpayments.PurchaseRequest) error {
	m.requests[request.ID] = request
	return nil
}

func (m *mockRequestRepo) List(_ context.Context, filter domainpayments.PurchaseRequestFilter, page, limit int) ([]*domainpayments.PurchaseRequest, int, error) {
	requests := m.filtered(filter)
	return requests, len(requests), nil
}

func (m *mockRequestRepo) ListAll(_ context.Context, filter domainpayments.PurchaseRequestFilter) ([]*domainpayments.PurchaseRequest, error) {
	return m.filtered(filter), nil
}

func (m *mockRequestRepo) filtered(filter domainpayments.PurchaseRequestFilter) []*domainpayments.PurchaseRequest {
	requests := make([]*domainpayments.PurchaseRequest, 0, len(m.requests))
	for _, request := range m.requests {
		if filter.StudentID != nil && request.StudentID != *filter.StudentID {
			continue
		}
		if filter.ItemType != nil && request.ItemType != *filter.ItemType {
			continue
		}
		if filter.Status != nil && request.Status != *filter.Status {
			continue
		}
		requests = append(requests, request)
	}
	// Sort newest-first to mirror the real PostgreSQL repository's
	// `ORDER BY created_at DESC`. Map iteration order is randomized in Go,
	// so without this the export and pagination tests are non-deterministic.
	sort.Slice(requests, func(i, j int) bool {
		return requests[i].CreatedAt.After(requests[j].CreatedAt)
	})
	return requests
}

type mockTxRunner struct {
	calls int
}

func (m *mockTxRunner) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	m.calls++
	return fn(ctx)
}

type mockAuditLogger struct {
	actions []string
}

func (m *mockAuditLogger) LogAction(_ context.Context, _ uuid.UUID, _ string, action, _ string, _ uuid.UUID, _ map[string]interface{}, _ string) error {
	m.actions = append(m.actions, action)
	return nil
}

func newApprovalService() (Service, *mockUserRepo, *mockCourseRepo, *mockBookRepo, *mockEnrollmentRepo, *mockOrderRepo, *mockRequestRepo, *mockTxRunner, *mockAuditLogger) {
	users := newMockUserRepo()
	coursesRepo := newMockCourseRepo()
	booksRepo := newMockBookRepo()
	enrollmentsRepo := newMockEnrollmentRepo()
	ordersRepo := newMockOrderRepo()
	requestsRepo := newMockRequestRepo()
	txRunner := &mockTxRunner{}
	audit := &mockAuditLogger{}

	svc := NewService(ServiceDeps{
		RequestRepo:    requestsRepo,
		UserRepo:       users,
		CourseRepo:     coursesRepo,
		BookRepo:       booksRepo,
		EnrollmentRepo: enrollmentsRepo,
		OrderRepo:      ordersRepo,
		TxRunner:       txRunner,
		AuditLogger:    audit,
	})

	return svc, users, coursesRepo, booksRepo, enrollmentsRepo, ordersRepo, requestsRepo, txRunner, audit
}

func TestService_CreatePurchaseRequest_IsIdempotentAndRoleChecked(t *testing.T) {
	svc, users, coursesRepo, _, _, _, requestsRepo, _, _ := newApprovalService()
	studentID := uuid.New()
	courseID := uuid.New()
	users.users[studentID] = &auth.User{
		ID:              studentID,
		FullName:        "Student One",
		Email:           "student@example.com",
		Role:            "student",
		Status:          "active",
		ProfileComplete: true,
	}
	coursesRepo.courses[courseID] = &courses.Course{ID: courseID, Title: "Biology 101", Subject: "Science"}

	first, err := svc.CreatePurchaseRequest(context.Background(), CreatePurchaseRequestCommand{
		StudentID:      studentID,
		ItemType:       domainpayments.PurchaseRequestItemTypeCourse,
		ItemID:         courseID,
		FileName:       "note.pdf",
		IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatalf("CreatePurchaseRequest failed: %v", err)
	}
	if first.Status != domainpayments.PurchaseRequestStatusPending {
		t.Fatalf("expected pending status, got %s", first.Status)
	}

	second, err := svc.CreatePurchaseRequest(context.Background(), CreatePurchaseRequestCommand{
		StudentID:      studentID,
		ItemType:       domainpayments.PurchaseRequestItemTypeCourse,
		ItemID:         courseID,
		FileName:       "ignored.pdf",
		IdempotencyKey: "idem-1",
	})
	if err != nil {
		t.Fatalf("idempotent CreatePurchaseRequest failed: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected idempotent request to return the same record, got %s and %s", first.ID, second.ID)
	}
	if len(requestsRepo.requests) != 1 {
		t.Fatalf("expected exactly 1 stored request, got %d", len(requestsRepo.requests))
	}

	t.Run("role check", func(t *testing.T) {
		teacherID := uuid.New()
		users.users[teacherID] = &auth.User{
			ID:              teacherID,
			FullName:        "Teacher One",
			Email:           "teacher@example.com",
			Role:            "teacher",
			Status:          "active",
			ProfileComplete: true,
		}

		_, err := svc.CreatePurchaseRequest(context.Background(), CreatePurchaseRequestCommand{
			StudentID: teacherID,
			ItemType:  domainpayments.PurchaseRequestItemTypeCourse,
			ItemID:    courseID,
		})
		if err == nil {
			t.Fatal("expected teacher request to be rejected")
		}
	})
}

func TestService_ApproveRejectAndExportPurchaseRequests(t *testing.T) {
	svc, users, coursesRepo, booksRepo, enrollmentsRepo, ordersRepo, requestsRepo, txRunner, audit := newApprovalService()
	studentID := uuid.New()
	courseID := uuid.New()
	bookID := uuid.New()
	courseRequestID := uuid.New()
	bookRequestID := uuid.New()

	users.users[studentID] = &auth.User{
		ID:              studentID,
		FullName:        "Student One",
		Email:           "student@example.com",
		Role:            "student",
		Status:          "active",
		ProfileComplete: true,
	}
	coursesRepo.courses[courseID] = &courses.Course{ID: courseID, Title: "Biology 101", Subject: "Science"}
	booksRepo.books[bookID] = &domainbookshop.Book{ID: bookID, Title: "Physics Workbook", Author: "A. Teacher", Price: 150, Currency: "BDT", IsActive: true}

	requestsRepo.requests[courseRequestID] = &domainpayments.PurchaseRequest{
		ID:        courseRequestID,
		StudentID: studentID,
		ItemType:  domainpayments.PurchaseRequestItemTypeCourse,
		ItemID:    courseID,
		FileName:  "course-file.pdf",
		Status:    domainpayments.PurchaseRequestStatusPending,
		CreatedAt: time.Now().Add(-2 * time.Hour),
		UpdatedAt: time.Now().Add(-2 * time.Hour),
	}
	requestsRepo.requests[bookRequestID] = &domainpayments.PurchaseRequest{
		ID:        bookRequestID,
		StudentID: studentID,
		ItemType:  domainpayments.PurchaseRequestItemTypeBook,
		ItemID:    bookID,
		FileName:  "book-file.pdf",
		Status:    domainpayments.PurchaseRequestStatusPending,
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now().Add(-time.Hour),
	}

	approvedCourse, err := svc.ApprovePurchaseRequest(context.Background(), ReviewPurchaseRequestCommand{
		RequestID: courseRequestID,
		ActorID:   uuid.New(),
		ActorName: "Admin One",
		IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("ApprovePurchaseRequest(course) failed: %v", err)
	}
	if approvedCourse.Status != domainpayments.PurchaseRequestStatusApproved {
		t.Fatalf("expected approved status, got %s", approvedCourse.Status)
	}
	if approvedCourse.ResultEnrollmentID == nil {
		t.Fatal("expected course approval to create an enrollment")
	}
	if len(enrollmentsRepo.enrollments) != 1 {
		t.Fatalf("expected 1 enrollment, got %d", len(enrollmentsRepo.enrollments))
	}

	approvedBook, err := svc.ApprovePurchaseRequest(context.Background(), ReviewPurchaseRequestCommand{
		RequestID: bookRequestID,
		ActorID:   uuid.New(),
		ActorName: "Admin One",
		IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("ApprovePurchaseRequest(book) failed: %v", err)
	}
	if approvedBook.ResultOrderID == nil {
		t.Fatal("expected book approval to create an order")
	}
	if len(ordersRepo.orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(ordersRepo.orders))
	}
	for _, order := range ordersRepo.orders {
		if order.Amount != 150 {
			t.Fatalf("expected book order amount to match book price, got %v", order.Amount)
		}
	}

	rejectID := uuid.New()
	requestsRepo.requests[rejectID] = &domainpayments.PurchaseRequest{
		ID:        rejectID,
		StudentID: studentID,
		ItemType:  domainpayments.PurchaseRequestItemTypeCourse,
		ItemID:    courseID,
		FileName:  "course-file.pdf",
		Status:    domainpayments.PurchaseRequestStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	rejected, err := svc.RejectPurchaseRequest(context.Background(), ReviewPurchaseRequestCommand{
		RequestID: rejectID,
		ActorID:   uuid.New(),
		ActorName: "Admin One",
		Reason:    "Missing documents",
		IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("RejectPurchaseRequest failed: %v", err)
	}
	if rejected.Status != domainpayments.PurchaseRequestStatusRejected {
		t.Fatalf("expected rejected status, got %s", rejected.Status)
	}
	if rejected.RejectionReason == nil || *rejected.RejectionReason != "Missing documents" {
		t.Fatal("expected rejection reason to be stored")
	}

	rows, err := svc.ExportAdminPurchaseRequests(context.Background(), ListPurchaseRequestsCommand{})
	if err != nil {
		t.Fatalf("ExportAdminPurchaseRequests failed: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 export rows, got %d", len(rows))
	}
	if rows[0].StudentID == "" || rows[0].ItemTitle == "" || rows[0].FileName == "" {
		t.Fatalf("expected populated export row, got %+v", rows[0])
	}
	if txRunner.calls < 2 {
		t.Fatalf("expected tx runner to be used, got %d calls", txRunner.calls)
	}
	if len(audit.actions) < 3 {
		t.Fatalf("expected audit entries for approve/reject actions, got %d", len(audit.actions))
	}
}

func TestService_CreatePurchaseRequest_RejectsIncompleteProfiles(t *testing.T) {
	svc, users, coursesRepo, _, _, _, _, _, _ := newApprovalService()
	studentID := uuid.New()
	courseID := uuid.New()
	users.users[studentID] = &auth.User{
		ID:              studentID,
		FullName:        "Student One",
		Email:           "student@example.com",
		Role:            "student",
		Status:          "active",
		ProfileComplete: false,
	}
	coursesRepo.courses[courseID] = &courses.Course{ID: courseID, Title: "Biology 101"}

	_, err := svc.CreatePurchaseRequest(context.Background(), CreatePurchaseRequestCommand{
		StudentID: studentID,
		ItemType:  domainpayments.PurchaseRequestItemTypeCourse,
		ItemID:    courseID,
	})
	if err == nil {
		t.Fatal("expected incomplete profile request to fail")
	}
}
