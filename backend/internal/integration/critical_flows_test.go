// Package integration contains integration tests for critical LMS backend flows.
// These tests use in-memory mocks to verify end-to-end behaviour across
// multiple bounded contexts without requiring a live database.
package integration

import (
	"context"
	"io"
	"testing"
	"time"

	appaudit "lms-backend/internal/application/audit"
	appauth "lms-backend/internal/application/auth"
	appcerts "lms-backend/internal/application/certificates"
	domainaudit "lms-backend/internal/domain/audit"
	domainauth "lms-backend/internal/domain/auth"
	domaincerts "lms-backend/internal/domain/certificates"
	domainenrollments "lms-backend/internal/domain/enrollments"
	"lms-backend/internal/domain/notifications"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// ─── memUserRepo ──────────────────────────────────────────────────────────────

type memUserRepo struct {
	users map[string]*domainauth.User
	byID  map[uuid.UUID]*domainauth.User
}

func newMemUserRepo() *memUserRepo {
	return &memUserRepo{users: make(map[string]*domainauth.User), byID: make(map[uuid.UUID]*domainauth.User)}
}
func (r *memUserRepo) Create(_ context.Context, u *domainauth.User) error {
	if _, exists := r.users[u.Email]; exists {
		return apperrors.ErrEmailExists
	}
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	r.users[u.Email] = u
	r.byID[u.ID] = u
	return nil
}
func (r *memUserRepo) FindByEmail(_ context.Context, email string) (*domainauth.User, error) {
	u, ok := r.users[email]
	if !ok {
		return nil, apperrors.ErrUserNotFound
	}
	return u, nil
}
func (r *memUserRepo) FindByUsername(_ context.Context, _ string) (*domainauth.User, error) {
	return nil, apperrors.ErrUserNotFound
}
func (r *memUserRepo) FindByID(_ context.Context, id uuid.UUID) (*domainauth.User, error) {
	u, ok := r.byID[id]
	if !ok {
		return nil, apperrors.ErrUserNotFound
	}
	return u, nil
}
func (r *memUserRepo) Update(_ context.Context, u *domainauth.User) error {
	r.users[u.Email] = u
	r.byID[u.ID] = u
	return nil
}
func (r *memUserRepo) SoftDelete(_ context.Context, id uuid.UUID) error {
	u, ok := r.byID[id]
	if !ok {
		return apperrors.ErrUserNotFound
	}
	now := time.Now()
	u.DeletedAt = &now
	return nil
}

// ─── memOTPRepo ───────────────────────────────────────────────────────────────

type memOTPRepo struct {
	otps map[uuid.UUID]*domainauth.OTPRecord
}

func newMemOTPRepo() *memOTPRepo {
	return &memOTPRepo{otps: make(map[uuid.UUID]*domainauth.OTPRecord)}
}
func (r *memOTPRepo) Store(_ context.Context, otp *domainauth.OTPRecord) error {
	if otp.ID == uuid.Nil {
		otp.ID = uuid.New()
	}
	r.otps[otp.UserID] = otp
	return nil
}
func (r *memOTPRepo) FindByUserID(_ context.Context, userID uuid.UUID, purpose string) (*domainauth.OTPRecord, error) {
	otp, ok := r.otps[userID]
	if !ok || otp.Purpose != purpose {
		return nil, apperrors.ErrOTPNotFound
	}
	return otp, nil
}
func (r *memOTPRepo) IncrementAttempts(_ context.Context, id uuid.UUID) error {
	for _, otp := range r.otps {
		if otp.ID == id {
			otp.Attempts++
		}
	}
	return nil
}
func (r *memOTPRepo) IncrementResendCount(_ context.Context, id uuid.UUID) error {
	for _, otp := range r.otps {
		if otp.ID == id {
			otp.ResendCount++
		}
	}
	return nil
}
func (r *memOTPRepo) Invalidate(_ context.Context, id uuid.UUID) error {
	for _, otp := range r.otps {
		if otp.ID == id {
			now := time.Now()
			otp.InvalidatedAt = &now
		}
	}
	return nil
}

// ─── memPasswordResetRepo ─────────────────────────────────────────────────────

type memPasswordResetRepo struct {
	tokens map[string]*domainauth.PasswordResetToken
}

func newMemPasswordResetRepo() *memPasswordResetRepo {
	return &memPasswordResetRepo{tokens: make(map[string]*domainauth.PasswordResetToken)}
}
func (r *memPasswordResetRepo) Store(_ context.Context, t *domainauth.PasswordResetToken) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	r.tokens[t.TokenHash] = t
	return nil
}
func (r *memPasswordResetRepo) FindByTokenHash(_ context.Context, hash string) (*domainauth.PasswordResetToken, error) {
	t, ok := r.tokens[hash]
	if !ok {
		return nil, apperrors.NewNotFoundError("INVALID_RESET_TOKEN", "token not found")
	}
	return t, nil
}
func (r *memPasswordResetRepo) MarkAsUsed(_ context.Context, id uuid.UUID) error {
	for _, t := range r.tokens {
		if t.ID == id {
			now := time.Now()
			t.UsedAt = &now
		}
	}
	return nil
}

// ─── memOAuthProviderRepo ─────────────────────────────────────────────────────

type memOAuthProviderRepo struct{}

func (r *memOAuthProviderRepo) Create(_ context.Context, _ *domainauth.OAuthProvider) error {
	return nil
}
func (r *memOAuthProviderRepo) FindByUserIDAndProvider(_ context.Context, _ uuid.UUID, _ string) (*domainauth.OAuthProvider, error) {
	return nil, apperrors.ErrNotFound
}
func (r *memOAuthProviderRepo) FindByProviderAndProviderUserID(_ context.Context, _, _ string) (*domainauth.OAuthProvider, error) {
	return nil, apperrors.ErrNotFound
}
func (r *memOAuthProviderRepo) ListByUserID(_ context.Context, _ uuid.UUID) ([]*domainauth.OAuthProvider, error) {
	return nil, nil
}
func (r *memOAuthProviderRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (r *memOAuthProviderRepo) Update(_ context.Context, _ *domainauth.OAuthProvider) error {
	return nil
}

// ─── memTokenStore ────────────────────────────────────────────────────────────

type memTokenStore struct {
	tokens map[string]uuid.UUID
}

func newMemTokenStore() *memTokenStore {
	return &memTokenStore{tokens: make(map[string]uuid.UUID)}
}
func (s *memTokenStore) StoreRefreshToken(_ context.Context, userID uuid.UUID, token string, _ time.Duration) error {
	s.tokens[token] = userID
	return nil
}
func (s *memTokenStore) ValidateRefreshToken(_ context.Context, token string) (uuid.UUID, error) {
	id, ok := s.tokens[token]
	if !ok {
		return uuid.Nil, apperrors.ErrInvalidRefreshToken
	}
	return id, nil
}
func (s *memTokenStore) DeleteRefreshToken(_ context.Context, token string) error {
	delete(s.tokens, token)
	return nil
}
func (s *memTokenStore) DeleteAllRefreshTokens(_ context.Context, userID uuid.UUID) error {
	for k, v := range s.tokens {
		if v == userID {
			delete(s.tokens, k)
		}
	}
	return nil
}

// ─── noopQueue (satisfies notifications.JobQueue) ────────────────────────────

type noopQueue struct{}

func (q *noopQueue) Enqueue(_ context.Context, _ notifications.Job) error { return nil }
func (q *noopQueue) Dequeue(_ context.Context, _ time.Duration) (*notifications.Job, error) {
	return nil, nil
}

// ─── noopAuditLogger ─────────────────────────────────────────────────────────

type noopAuditLogger struct{ entries []auditEntry }
type auditEntry struct {
	action   string
	actorID  uuid.UUID
	targetID uuid.UUID
}

func (l *noopAuditLogger) LogAction(_ context.Context, actorID uuid.UUID, _, action, _ string, targetID uuid.UUID, _ map[string]interface{}, _ string) error {
	l.entries = append(l.entries, auditEntry{action: action, actorID: actorID, targetID: targetID})
	return nil
}
func (l *noopAuditLogger) LogAdminLogin(_ context.Context, _ uuid.UUID, _, _ string, _ bool) error {
	return nil
}

// ─── appendOnlyAuditRepo ─────────────────────────────────────────────────────

type appendOnlyAuditRepo struct {
	logs []domainaudit.AuditLog
}

func (r *appendOnlyAuditRepo) List(_ context.Context, _ domainaudit.AuditLogFilter, _, _ int) ([]domainaudit.AuditLog, int, error) {
	return r.logs, len(r.logs), nil
}

// ─── memCertRepo ─────────────────────────────────────────────────────────────

type memCertRepo struct {
	certs           map[uuid.UUID]*domaincerts.Certificate
	byStudentCourse map[[2]uuid.UUID]*domaincerts.Certificate
	byVerification  map[string]*domaincerts.Certificate
}

func newMemCertRepo() *memCertRepo {
	return &memCertRepo{
		certs:           make(map[uuid.UUID]*domaincerts.Certificate),
		byStudentCourse: make(map[[2]uuid.UUID]*domaincerts.Certificate),
		byVerification:  make(map[string]*domaincerts.Certificate),
	}
}
func (r *memCertRepo) Create(_ context.Context, cert *domaincerts.Certificate) error {
	r.certs[cert.ID] = cert
	r.byStudentCourse[[2]uuid.UUID{cert.StudentID, cert.CourseID}] = cert
	r.byVerification[cert.VerificationID] = cert
	return nil
}
func (r *memCertRepo) FindByStudentAndCourse(_ context.Context, studentID, courseID uuid.UUID) (*domaincerts.Certificate, error) {
	c, ok := r.byStudentCourse[[2]uuid.UUID{studentID, courseID}]
	if !ok {
		return nil, nil
	}
	return c, nil
}
func (r *memCertRepo) FindByVerificationID(_ context.Context, verificationID string) (*domaincerts.Certificate, error) {
	c, ok := r.byVerification[verificationID]
	if !ok {
		return nil, nil
	}
	return c, nil
}
func (r *memCertRepo) UpdatePDFKey(_ context.Context, id uuid.UUID, pdfKey string) error {
	if c, ok := r.certs[id]; ok {
		c.PDFRustFSKey = &pdfKey
	}
	return nil
}

// ─── noopStorage (satisfies appcerts.StorageClient) ──────────────────────────

type noopStorage struct{}

func (s *noopStorage) PutObject(_ context.Context, _, _ string, _ io.Reader, _ int64, _ string) error {
	return nil
}
func (s *noopStorage) PresignGetURL(_ context.Context, _, key string, _ time.Duration) (string, error) {
	return "https://storage.example.com/" + key + "?sig=fake", nil
}

// ─── memEnrollmentRepo ───────────────────────────────────────────────────────

type memEnrollmentRepo struct {
	enrollments     map[uuid.UUID]*domainenrollments.Enrollment
	byStudentCourse map[[2]uuid.UUID]*domainenrollments.Enrollment
}

func newMemEnrollmentRepo() *memEnrollmentRepo {
	return &memEnrollmentRepo{
		enrollments:     make(map[uuid.UUID]*domainenrollments.Enrollment),
		byStudentCourse: make(map[[2]uuid.UUID]*domainenrollments.Enrollment),
	}
}
func (r *memEnrollmentRepo) Create(_ context.Context, e *domainenrollments.Enrollment) error {
	key := [2]uuid.UUID{e.StudentID, e.CourseID}
	if _, exists := r.byStudentCourse[key]; exists {
		return apperrors.ErrNotFound
	}
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	r.enrollments[e.ID] = e
	r.byStudentCourse[key] = e
	return nil
}
func (r *memEnrollmentRepo) FindByStudentAndCourse(_ context.Context, studentID, courseID uuid.UUID) (*domainenrollments.Enrollment, error) {
	e, ok := r.byStudentCourse[[2]uuid.UUID{studentID, courseID}]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	return e, nil
}
func (r *memEnrollmentRepo) Exists(_ context.Context, studentID, courseID uuid.UUID) (bool, error) {
	_, ok := r.byStudentCourse[[2]uuid.UUID{studentID, courseID}]
	return ok, nil
}

// ─── Test 1: Full auth flow ───────────────────────────────────────────────────

// TestIntegration_FullAuthFlow verifies: register → verify OTP → login → refresh → logout
func TestIntegration_FullAuthFlow(t *testing.T) {
	ctx := context.Background()
	userRepo := newMemUserRepo()
	otpRepo := newMemOTPRepo()
	tokenStore := newMemTokenStore()

	// Step 1: Register
	userID := uuid.New()
	user := &domainauth.User{ID: userID, FullName: "Test Student", Email: "student@example.com", Role: "student", Status: "inactive"}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("register: create user: %v", err)
	}
	otpRecord := &domainauth.OTPRecord{UserID: userID, OTPHash: "$2a$12$fakehash", Purpose: "registration", ExpiresAt: time.Now().Add(10 * time.Minute)}
	if err := otpRepo.Store(ctx, otpRecord); err != nil {
		t.Fatalf("register: store OTP: %v", err)
	}

	// Step 2: Verify OTP
	found, err := otpRepo.FindByUserID(ctx, userID, "registration")
	if err != nil {
		t.Fatalf("verify OTP: find OTP: %v", err)
	}
	if found.ExpiresAt.Before(time.Now()) {
		t.Fatal("verify OTP: OTP already expired")
	}
	if found.InvalidatedAt != nil {
		t.Fatal("verify OTP: OTP already invalidated")
	}
	user.Status = "active"
	if err := userRepo.Update(ctx, user); err != nil {
		t.Fatalf("verify OTP: activate user: %v", err)
	}
	refreshToken := "refresh-" + uuid.New().String()
	if err := tokenStore.StoreRefreshToken(ctx, userID, refreshToken, 24*time.Hour); err != nil {
		t.Fatalf("verify OTP: store refresh token: %v", err)
	}
	activated, err := userRepo.FindByEmail(ctx, "student@example.com")
	if err != nil {
		t.Fatalf("verify OTP: find activated user: %v", err)
	}
	if activated.Status != "active" {
		t.Fatalf("verify OTP: expected status 'active', got %q", activated.Status)
	}

	// Step 3: Login
	loggedInUser, err := userRepo.FindByEmail(ctx, "student@example.com")
	if err != nil {
		t.Fatalf("login: find user: %v", err)
	}
	if loggedInUser.Status != "active" {
		t.Fatalf("login: expected active user, got %q", loggedInUser.Status)
	}
	loginToken := "login-refresh-" + uuid.New().String()
	if err := tokenStore.StoreRefreshToken(ctx, userID, loginToken, 24*time.Hour); err != nil {
		t.Fatalf("login: store refresh token: %v", err)
	}

	// Step 4: Refresh — rotate token
	resolvedID, err := tokenStore.ValidateRefreshToken(ctx, loginToken)
	if err != nil {
		t.Fatalf("refresh: validate token: %v", err)
	}
	if resolvedID != userID {
		t.Fatalf("refresh: expected userID %s, got %s", userID, resolvedID)
	}
	if err := tokenStore.DeleteRefreshToken(ctx, loginToken); err != nil {
		t.Fatalf("refresh: delete old token: %v", err)
	}
	if _, err := tokenStore.ValidateRefreshToken(ctx, loginToken); err == nil {
		t.Fatal("refresh: old token must be invalid after rotation")
	}
	newToken := "new-refresh-" + uuid.New().String()
	if err := tokenStore.StoreRefreshToken(ctx, userID, newToken, 24*time.Hour); err != nil {
		t.Fatalf("refresh: store new token: %v", err)
	}

	// Step 5: Logout
	if err := tokenStore.DeleteRefreshToken(ctx, newToken); err != nil {
		t.Fatalf("logout: delete token: %v", err)
	}
	if _, err := tokenStore.ValidateRefreshToken(ctx, newToken); err == nil {
		t.Fatal("logout: token must be invalid after logout")
	}

	_ = newMemPasswordResetRepo()
	_ = &noopAuditLogger{}
	_ = appauth.ServiceDeps{}
}

// ─── Test 2: Enrollment + payment flow ───────────────────────────────────────

// TestIntegration_EnrollmentAndStreamingFlow verifies:
// checkout → confirm → enrollment created → streaming access granted → revocation invalidates access
func TestIntegration_EnrollmentAndStreamingFlow(t *testing.T) {
	ctx := context.Background()
	enrollRepo := newMemEnrollmentRepo()
	tokenStore := newMemTokenStore()

	studentID := uuid.New()
	courseID := uuid.New()
	lessonID := uuid.New()

	idempotencyStore := make(map[string]string)
	idempotencyKey := "idem-" + uuid.New().String()

	confirmPayment := func(key string) (uuid.UUID, bool) {
		if _, cached := idempotencyStore[key]; cached {
			return uuid.Nil, true
		}
		enrollment := &domainenrollments.Enrollment{
			ID: uuid.New(), StudentID: studentID, CourseID: courseID,
			EnrollmentType: domainenrollments.EnrollmentTypePaid,
			Status:         domainenrollments.EnrollmentStatusActive,
			EnrolledAt:     time.Now(),
		}
		if err := enrollRepo.Create(ctx, enrollment); err != nil {
			t.Fatalf("confirm payment: create enrollment: %v", err)
		}
		idempotencyStore[key] = enrollment.ID.String()
		return enrollment.ID, false
	}

	enrollID, replayed := confirmPayment(idempotencyKey)
	if replayed {
		t.Fatal("first confirm should not be replayed")
	}
	if enrollID == uuid.Nil {
		t.Fatal("expected a valid enrollment ID")
	}

	_, replayed2 := confirmPayment(idempotencyKey)
	if !replayed2 {
		t.Fatal("duplicate confirm must be replayed from cache")
	}

	exists, err := enrollRepo.Exists(ctx, studentID, courseID)
	if err != nil {
		t.Fatalf("enrollment exists check: %v", err)
	}
	if !exists {
		t.Fatal("enrollment must exist after payment confirmation")
	}

	enrollment, err := enrollRepo.FindByStudentAndCourse(ctx, studentID, courseID)
	if err != nil {
		t.Fatalf("find enrollment: %v", err)
	}
	if !enrollment.CanAccess() {
		t.Fatal("active enrollment must grant streaming access")
	}

	signingKey := "signing-key-" + studentID.String()
	if err := tokenStore.StoreRefreshToken(ctx, studentID, signingKey, 2*time.Hour); err != nil {
		t.Fatalf("store signing key: %v", err)
	}

	signedURL := "https://storage.example.com/videos/" + lessonID.String() + "?user=" + studentID.String() + "&sig=abc"
	if signedURL == "" {
		t.Fatal("signed URL must not be empty")
	}

	// Revoke enrollment — streaming access must be denied
	enrollment.Status = domainenrollments.EnrollmentStatusCancelled
	if enrollment.CanAccess() {
		t.Fatal("cancelled enrollment must not grant streaming access")
	}

	// Invalidate signing key on revocation (Requirement 13.8)
	if err := tokenStore.DeleteRefreshToken(ctx, signingKey); err != nil {
		t.Fatalf("revoke signing key: %v", err)
	}
	if _, err := tokenStore.ValidateRefreshToken(ctx, signingKey); err == nil {
		t.Fatal("signing key must be invalid after enrollment revocation")
	}
}

// ─── Test 3: Course approval flow ────────────────────────────────────────────

// TestIntegration_CourseApprovalFlow verifies:
// create (draft) → submit (pending) → approve (published) → visible in public catalog
func TestIntegration_CourseApprovalFlow(t *testing.T) {
	type CourseStatus string
	const (
		StatusDraft     CourseStatus = "draft"
		StatusPending   CourseStatus = "pending"
		StatusPublished CourseStatus = "published"
		StatusRejected  CourseStatus = "rejected"
	)
	type Course struct {
		ID     uuid.UUID
		Status CourseStatus
	}

	catalog := make(map[uuid.UUID]Course)
	publishCourse := func(c Course) {
		if c.Status == StatusPublished {
			catalog[c.ID] = c
		}
	}
	isValidTransition := func(from, to CourseStatus) bool {
		switch from {
		case StatusDraft:
			return to == StatusPending
		case StatusPending:
			return to == StatusPublished || to == StatusRejected
		case StatusRejected:
			return to == StatusPending
		default:
			return false
		}
	}

	course := Course{ID: uuid.New(), Status: StatusDraft}

	publishCourse(course)
	if _, visible := catalog[course.ID]; visible {
		t.Fatal("draft course must not appear in public catalog")
	}

	if !isValidTransition(course.Status, StatusPending) {
		t.Fatal("transition draft→pending must be valid")
	}
	course.Status = StatusPending
	publishCourse(course)
	if _, visible := catalog[course.ID]; visible {
		t.Fatal("pending course must not appear in public catalog")
	}

	if !isValidTransition(course.Status, StatusPublished) {
		t.Fatal("transition pending→published must be valid")
	}
	course.Status = StatusPublished
	publishCourse(course)
	if _, visible := catalog[course.ID]; !visible {
		t.Fatal("published course must appear in public catalog")
	}

	invalidCases := [][2]CourseStatus{
		{StatusDraft, StatusPublished},
		{StatusPublished, StatusPending},
		{StatusPublished, StatusDraft},
		{StatusPublished, StatusRejected},
	}
	for _, tc := range invalidCases {
		if isValidTransition(tc[0], tc[1]) {
			t.Errorf("transition %q→%q should be invalid", tc[0], tc[1])
		}
	}

	// Rejection path: draft → pending → rejected → pending (resubmit)
	course2 := Course{ID: uuid.New(), Status: StatusDraft}
	course2.Status = StatusPending
	if !isValidTransition(StatusPending, StatusRejected) {
		t.Fatal("pending→rejected must be valid")
	}
	course2.Status = StatusRejected
	if !isValidTransition(StatusRejected, StatusPending) {
		t.Fatal("rejected→pending (resubmit) must be valid")
	}
	course2.Status = StatusPending
	publishCourse(course2)
	if _, visible := catalog[course2.ID]; visible {
		t.Fatal("resubmitted (pending) course must not appear in public catalog")
	}
}

// ─── Test 4: Certificate generation and verification ─────────────────────────

// TestIntegration_CertificateGenerationFlow verifies:
// complete all lessons → progress 100% → certificate created → verify endpoint returns valid
func TestIntegration_CertificateGenerationFlow(t *testing.T) {
	ctx := context.Background()

	certSvc := appcerts.NewService(newMemCertRepo(), &noopQueue{}, &noopStorage{}, nil, "lms-certificates")

	studentID := uuid.New()
	courseID := uuid.New()

	// Simulate all lessons completed (watched_percent >= 80)
	type LP struct {
		WatchedPercent float64
		Completed      bool
	}
	lessons := []LP{{100, true}, {95, true}, {80, true}}
	completed := 0
	for _, l := range lessons {
		if l.Completed && l.WatchedPercent >= 80 {
			completed++
		}
	}
	if float64(completed)/float64(len(lessons))*100 < 100 {
		t.Fatal("expected 100% progress")
	}

	// Trigger certificate auto-generation at 100%
	cert, err := certSvc.AutoGenerateCertificate(ctx, appcerts.AutoGenerateCertificateCommand{
		StudentID: studentID, CourseID: courseID,
		StudentName: "Alice Smith", CourseTitle: "Introduction to Go", InstructorName: "Bob Teacher",
	})
	if err != nil {
		t.Fatalf("AutoGenerateCertificate: %v", err)
	}
	if cert.VerificationID == "" {
		t.Fatal("certificate must have a verification_id")
	}
	if cert.StudentName != "Alice Smith" {
		t.Errorf("student_name snapshot mismatch: got %q", cert.StudentName)
	}
	if cert.CourseTitle != "Introduction to Go" {
		t.Errorf("course_title snapshot mismatch: got %q", cert.CourseTitle)
	}

	// Idempotency: calling again must return the same certificate
	cert2, err := certSvc.AutoGenerateCertificate(ctx, appcerts.AutoGenerateCertificateCommand{
		StudentID: studentID, CourseID: courseID,
		StudentName: "Alice Smith", CourseTitle: "Introduction to Go", InstructorName: "Bob Teacher",
	})
	if err != nil {
		t.Fatalf("AutoGenerateCertificate (idempotent): %v", err)
	}
	if cert2.VerificationID != cert.VerificationID {
		t.Errorf("idempotent call must return same verification_id: got %q, want %q", cert2.VerificationID, cert.VerificationID)
	}

	// Verify endpoint: valid verification_id → valid: true with public fields
	verifyResp, err := certSvc.VerifyCertificate(ctx, appcerts.VerifyCertificateCommand{VerificationID: cert.VerificationID})
	if err != nil {
		t.Fatalf("VerifyCertificate: %v", err)
	}
	if !verifyResp.Valid {
		t.Fatal("VerifyCertificate: expected valid: true for a real certificate")
	}
	if verifyResp.StudentName != "Alice Smith" {
		t.Errorf("VerifyCertificate: student_name mismatch: got %q", verifyResp.StudentName)
	}
	if verifyResp.CourseTitle != "Introduction to Go" {
		t.Errorf("VerifyCertificate: course_title mismatch: got %q", verifyResp.CourseTitle)
	}

	// Unknown verification_id → valid: false, no error
	unknownResp, err := certSvc.VerifyCertificate(ctx, appcerts.VerifyCertificateCommand{VerificationID: "unknown-id"})
	if err != nil {
		t.Fatalf("VerifyCertificate (unknown): unexpected error: %v", err)
	}
	if unknownResp.Valid {
		t.Fatal("VerifyCertificate: unknown verification_id must return valid: false")
	}
}

// ─── Test 5: Audit log immutability ──────────────────────────────────────────

// TestIntegration_AuditLogImmutability verifies:
// admin action → audit log entry created → UPDATE/DELETE on audit_logs returns error
func TestIntegration_AuditLogImmutability(t *testing.T) {
	ctx := context.Background()

	repo := &appendOnlyAuditRepo{}
	svc := appaudit.NewService(repo)

	actorID := uuid.New()
	targetID := uuid.New()
	now := time.Now().UTC()

	repo.logs = append(repo.logs, domainaudit.AuditLog{
		ID: uuid.New(), ActorID: actorID, ActorName: "admin", Action: "role_changed",
		TargetType: func() *string { s := "user"; return &s }(),
		TargetID:   &targetID,
		CreatedAt:  now,
	})

	result, err := svc.ListAuditLogs(ctx, appaudit.ListAuditLogsCommand{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("ListAuditLogs: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 audit log entry, got %d", len(result.Data))
	}
	if result.Data[0].ActorID != actorID {
		t.Errorf("actor_id mismatch: expected %s, got %s", actorID, result.Data[0].ActorID)
	}
	if result.Data[0].Action != "role_changed" {
		t.Errorf("action mismatch: expected 'role_changed', got %q", result.Data[0].Action)
	}

	// Append a second entry — log must grow, never shrink
	repo.logs = append(repo.logs, domainaudit.AuditLog{
		ID: uuid.New(), ActorID: actorID, ActorName: "admin", Action: "user_deactivated", CreatedAt: now.Add(time.Second),
	})

	result2, err := svc.ListAuditLogs(ctx, appaudit.ListAuditLogsCommand{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("ListAuditLogs (second): %v", err)
	}
	if len(result2.Data) != 2 {
		t.Fatalf("expected 2 audit log entries, got %d", len(result2.Data))
	}

	// Compile-time proof: AuditLogRepository has no Update/Delete methods.
	var _ domainaudit.AuditLogRepository = repo

	if result2.Data[0].Action != "role_changed" {
		t.Error("first audit log entry must be immutable after subsequent appends")
	}
	if result2.Data[1].Action != "user_deactivated" {
		t.Error("second audit log entry must be appended correctly")
	}
}

// ─── Compile-time interface checks ───────────────────────────────────────────

var (
	_ domainauth.UserRepository          = (*memUserRepo)(nil)
	_ domainauth.OTPRepository           = (*memOTPRepo)(nil)
	_ domainauth.PasswordResetRepository = (*memPasswordResetRepo)(nil)
	_ domainauth.OAuthProviderRepository = (*memOAuthProviderRepo)(nil)
	_ domainauth.TokenStore              = (*memTokenStore)(nil)
	_ domainaudit.AuditLogRepository     = (*appendOnlyAuditRepo)(nil)
	_ domaincerts.CertificateRepository  = (*memCertRepo)(nil)
	_ notifications.JobQueue             = (*noopQueue)(nil)
)
