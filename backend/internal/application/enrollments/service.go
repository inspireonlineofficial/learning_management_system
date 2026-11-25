package enrollments

import (
	"context"
	"fmt"
	"time"

	appcertificates "lms-backend/internal/application/certificates"
	"lms-backend/internal/domain/auth"
	"lms-backend/internal/domain/courses"
	"lms-backend/internal/domain/enrollments"
	"lms-backend/internal/infrastructure/rustfs"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// Service defines the interface for enrollment use cases
type Service interface {
	// Enrollment operations
	EnrollFree(ctx context.Context, cmd EnrollFreeCommand) (*EnrollmentResponse, error)
	RevokeEnrollment(ctx context.Context, cmd RevokeEnrollmentCommand) error
	GetEnrollment(ctx context.Context, studentID, courseID uuid.UUID) (*EnrollmentResponse, error)
	ListStudentEnrollments(ctx context.Context, studentID uuid.UUID, status string, page, limit int) ([]EnrollmentResponse, int, error)

	// Progress tracking
	UpdateLessonProgress(ctx context.Context, cmd UpdateLessonProgressCommand) (*LessonProgressResponse, error)
	GetLessonProgress(ctx context.Context, enrollmentID, lessonID uuid.UUID) (*LessonProgressResponse, error)
	GetCourseProgress(ctx context.Context, studentID, courseID uuid.UUID) (*CourseProgressResponse, error)

	// Video streaming
	GetStreamingSignedURL(ctx context.Context, cmd GetStreamingSignedURLCommand) (*StreamingSignedURLResponse, error)
	// GetHLSPlaylist serves a signed HLS master/rendition playlist with
	// presigned segment URLs.
	GetHLSPlaylist(ctx context.Context, videoID uuid.UUID, rendition, exp, sig string) ([]byte, error)
}

type service struct {
	enrollmentRepo     enrollments.EnrollmentRepository
	lessonProgressRepo enrollments.LessonProgressRepository
	courseRepo         courses.CourseRepository
	lessonRepo         courses.LessonRepository
	videoRepo          courses.VideoRepository
	userRepo           auth.UserRepository
	signingKeyStore    SigningKeyStore
	storageClient      StorageClient
	videoBucket        string
	certificateIssuer  CertificateIssuer
	// streamSigningKey authenticates the backend HLS playlist URLs (see
	// streaming.go). Nil disables HLS playlist delivery, in which case the
	// player uses the progressive MP4 URL.
	streamSigningKey []byte
	// edge configures optional authorization-aware CDN delivery.
	edge EdgeDeliveryConfig
}

// StreamSigningKeySetter is implemented by the enrollment service so the
// composition root can provide the HMAC key used for HLS playlist URLs.
type StreamSigningKeySetter interface {
	SetStreamSigningKey(key []byte)
}

// SetStreamSigningKey configures the HMAC key for signed HLS playlist URLs.
func (s *service) SetStreamSigningKey(key []byte) {
	s.streamSigningKey = append([]byte(nil), key...)
}

// SigningKeyStore defines the interface for managing user-scoped signing keys in Redis
type SigningKeyStore interface {
	InvalidateUserSigningKey(ctx context.Context, userID uuid.UUID, courseID uuid.UUID) error
}

// StorageClient defines the interface for object storage operations
type StorageClient interface {
	PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)
	PresignGetURLWithOptions(ctx context.Context, bucket, key string, ttl time.Duration, opts rustfs.PresignOptions) (string, error)
}

type CertificateIssuer interface {
	AutoGenerateCertificate(ctx context.Context, cmd appcertificates.AutoGenerateCertificateCommand) (*appcertificates.CertificateResponse, error)
}

// NewService creates a new enrollment service
func NewService(
	enrollmentRepo enrollments.EnrollmentRepository,
	lessonProgressRepo enrollments.LessonProgressRepository,
	courseRepo courses.CourseRepository,
	lessonRepo courses.LessonRepository,
	videoRepo courses.VideoRepository,
	userRepo auth.UserRepository,
	signingKeyStore SigningKeyStore,
	storageClient StorageClient,
	videoBucket string,
	certificateIssuers ...CertificateIssuer,
) Service {
	var certificateIssuer CertificateIssuer
	if len(certificateIssuers) > 0 {
		certificateIssuer = certificateIssuers[0]
	}

	return &service{
		enrollmentRepo:     enrollmentRepo,
		lessonProgressRepo: lessonProgressRepo,
		courseRepo:         courseRepo,
		lessonRepo:         lessonRepo,
		videoRepo:          videoRepo,
		userRepo:           userRepo,
		signingKeyStore:    signingKeyStore,
		storageClient:      storageClient,
		videoBucket:        videoBucket,
		certificateIssuer:  certificateIssuer,
	}
}

// EnrollFree implements idempotent free course enrollment
// Requirements: 13.1, 13.2
func (s *service) EnrollFree(ctx context.Context, cmd EnrollFreeCommand) (*EnrollmentResponse, error) {
	// Check if student profile is complete at service layer (Requirement 13.7)
	user, err := s.userRepo.FindByID(ctx, cmd.StudentID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("USER_NOT_FOUND", "user not found")
	}

	if user.Role != "student" {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "only students can enroll in courses")
	}

	if !user.ProfileComplete {
		return nil, apperrors.ErrProfileIncomplete
	}

	// Check if course exists and is published
	course, err := s.courseRepo.FindByID(ctx, cmd.CourseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}

	// Only published courses can be joined; drafts and courses under review
	// are not available to students.
	if course.Status != courses.CourseStatusPublished {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}

	// Reject paid courses with NOT_FREE_COURSE error
	if course.PriceType != courses.PriceTypeFree {
		return nil, apperrors.NewSimpleValidationError("NOT_FREE_COURSE", "this course requires enrollment approval before access")
	}

	// Check if already enrolled (idempotency check)
	existing, err := s.enrollmentRepo.FindByStudentAndCourse(ctx, cmd.StudentID, cmd.CourseID)
	if err == nil && existing != nil {
		// Already enrolled - return ALREADY_ENROLLED error (Requirement 13.1)
		return nil, apperrors.NewConflictError("ALREADY_ENROLLED", "you are already enrolled in this course")
	}

	// Create enrollment record
	enrollment := &enrollments.Enrollment{
		ID:             uuid.New(),
		StudentID:      cmd.StudentID,
		CourseID:       cmd.CourseID,
		EnrollmentType: enrollments.EnrollmentTypeFree,
		Status:         enrollments.EnrollmentStatusActive,
		EnrolledAt:     time.Now(),
	}

	// INSERT ... ON CONFLICT DO NOTHING is handled at repository layer
	err = s.enrollmentRepo.Create(ctx, enrollment)
	if err != nil {
		// If we get a unique constraint violation, it means concurrent enrollment
		// Return ALREADY_ENROLLED
		if isUniqueViolation(err) {
			return nil, apperrors.NewConflictError("ALREADY_ENROLLED", "you are already enrolled in this course")
		}
		return nil, err
	}

	return s.toEnrollmentResponse(ctx, enrollment), nil
}

// RevokeEnrollment sets enrollment status to cancelled/refunded and invalidates signing keys
// Requirements: 13.2, 13.8
func (s *service) RevokeEnrollment(ctx context.Context, cmd RevokeEnrollmentCommand) error {
	// Check enrollment status at application service layer (Requirement 13.7)
	enrollment, err := s.enrollmentRepo.FindByStudentAndCourse(ctx, cmd.StudentID, cmd.CourseID)
	if err != nil {
		return apperrors.NewNotFoundError("ENROLLMENT_NOT_FOUND", "enrollment not found")
	}

	if !enrollment.IsActive() {
		return apperrors.NewSimpleValidationError("ENROLLMENT_NOT_ACTIVE", "enrollment is not active")
	}

	// Set status to cancelled or refunded based on command
	enrollment.Status = enrollments.EnrollmentStatus(cmd.Status)

	err = s.enrollmentRepo.Update(ctx, enrollment)
	if err != nil {
		return err
	}

	// Invalidate user-scoped signing key in Redis (Requirement 13.8)
	err = s.signingKeyStore.InvalidateUserSigningKey(ctx, cmd.StudentID, cmd.CourseID)
	if err != nil {
		// Log error but don't fail the operation
		// The enrollment is already revoked in the database
		// TODO: Add structured logging
	}

	return nil
}

// GetEnrollment retrieves a specific enrollment
func (s *service) GetEnrollment(ctx context.Context, studentID, courseID uuid.UUID) (*EnrollmentResponse, error) {
	enrollment, err := s.enrollmentRepo.FindByStudentAndCourse(ctx, studentID, courseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ENROLLMENT_NOT_FOUND", "enrollment not found")
	}

	return s.toEnrollmentResponse(ctx, enrollment), nil
}

// ListStudentEnrollments retrieves all enrollments for a student
func (s *service) ListStudentEnrollments(ctx context.Context, studentID uuid.UUID, status string, page, limit int) ([]EnrollmentResponse, int, error) {
	type statusFilterRepository interface {
		FindByStudentIDAndStatus(ctx context.Context, studentID uuid.UUID, status string, page, limit int) ([]*enrollments.Enrollment, int, error)
	}

	var enrollmentList []*enrollments.Enrollment
	var total int
	var err error

	if status != "" {
		if repoWithStatus, ok := s.enrollmentRepo.(statusFilterRepository); ok {
			enrollmentList, total, err = repoWithStatus.FindByStudentIDAndStatus(ctx, studentID, status, page, limit)
		} else {
			// Fallback: fetch all and filter in-memory for tests/mocks
			allEnrollments, _, fetchErr := s.enrollmentRepo.FindByStudentID(ctx, studentID, 1, 1000)
			if fetchErr != nil {
				return nil, 0, fetchErr
			}
			filteredList := make([]*enrollments.Enrollment, 0)
			for _, e := range allEnrollments {
				if status == "completed" && e.CompletedAt != nil && e.Status == enrollments.EnrollmentStatusActive {
					filteredList = append(filteredList, e)
				} else if status == "active" && e.CompletedAt == nil && e.Status == enrollments.EnrollmentStatusActive {
					filteredList = append(filteredList, e)
				}
			}
			total = len(filteredList)
			offset := (page - 1) * limit
			if offset < total {
				end := offset + limit
				if end > total {
					end = total
				}
				enrollmentList = filteredList[offset:end]
			}
		}
	} else {
		enrollmentList, total, err = s.enrollmentRepo.FindByStudentID(ctx, studentID, page, limit)
	}

	if err != nil {
		return nil, 0, err
	}

	responses := make([]EnrollmentResponse, 0, len(enrollmentList))
	for _, enrollment := range enrollmentList {
		resp := s.toEnrollmentResponse(ctx, enrollment)
		// Hide enrollments whose course has been soft-deleted (e.g. by an
		// admin using DELETE /v1/admin/courses/{id}). Surfacing a row with
		// Course = nil would crash the student's /student/my-courses page
		// because the frontend dereferences course.id when building the
		// player link. We also subtract it from the returned total so the
		// client's pagination stays consistent.
		if resp.Course == nil {
			total--
			continue
		}
		responses = append(responses, *resp)
	}

	return responses, total, nil
}

func (s *service) issueCertificate(ctx context.Context, enrollment *enrollments.Enrollment) error {
	if s.certificateIssuer == nil {
		return nil
	}

	student, err := s.userRepo.FindByID(ctx, enrollment.StudentID)
	if err != nil {
		return fmt.Errorf("failed to load student for certificate: %w", err)
	}
	course, err := s.courseRepo.FindByID(ctx, enrollment.CourseID)
	if err != nil {
		return fmt.Errorf("failed to load course for certificate: %w", err)
	}
	teacher, err := s.userRepo.FindByID(ctx, course.TeacherID)
	if err != nil {
		return fmt.Errorf("failed to load instructor for certificate: %w", err)
	}

	_, err = s.certificateIssuer.AutoGenerateCertificate(ctx, appcertificates.AutoGenerateCertificateCommand{
		StudentID:      student.ID,
		CourseID:       course.ID,
		StudentName:    student.FullName,
		CourseTitle:    course.Title,
		InstructorName: teacher.FullName,
	})
	if err != nil {
		return fmt.Errorf("failed to issue certificate: %w", err)
	}
	return nil
}

// GetLessonProgress retrieves lesson progress
func (s *service) GetLessonProgress(ctx context.Context, enrollmentID, lessonID uuid.UUID) (*LessonProgressResponse, error) {
	progress, err := s.lessonProgressRepo.FindByEnrollmentAndLesson(ctx, enrollmentID, lessonID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("LESSON_PROGRESS_NOT_FOUND", "lesson progress not found")
	}

	return s.toLessonProgressResponse(progress), nil
}

func (s *service) toEnrollmentResponse(ctx context.Context, enrollment *enrollments.Enrollment) *EnrollmentResponse {
	resp := &EnrollmentResponse{
		ID:              enrollment.ID,
		StudentID:       enrollment.StudentID,
		CourseID:        enrollment.CourseID,
		EnrollmentType:  string(enrollment.EnrollmentType),
		Status:          string(enrollment.Status),
		ProgressPercent: enrollment.ProgressPercent,
		CompletedAt:     enrollment.CompletedAt,
		EnrolledAt:      enrollment.EnrolledAt,
	}

	if course, err := s.courseRepo.FindByID(ctx, enrollment.CourseID); err == nil && course != nil {
		resp.Course = &CourseSummaryResponse{
			ID:       course.ID,
			Title:    course.Title,
			CoverURL: course.ThumbnailURL,
			Category: &CategorySummaryResponse{
				Name: course.Subject,
			},
		}
	}

	return resp
}

func (s *service) toLessonProgressResponse(progress *enrollments.LessonProgress) *LessonProgressResponse {
	return &LessonProgressResponse{
		ID:              progress.ID,
		EnrollmentID:    progress.EnrollmentID,
		LessonID:        progress.LessonID,
		PositionSeconds: progress.PositionSeconds,
		WatchedPercent:  progress.WatchedPercent,
		Completed:       progress.Completed,
		CompletedAt:     progress.CompletedAt,
		LastWatchedAt:   progress.LastWatchedAt,
	}
}

// isUniqueViolation checks if the error is a unique constraint violation
func isUniqueViolation(err error) bool {
	// PostgreSQL unique violation error code is 23505
	// This is a simplified check - in production, use a proper error type check
	if err == nil {
		return false
	}

	errMsg := err.Error()
	// Check for common unique violation error messages
	return contains(errMsg, "unique constraint") ||
		contains(errMsg, "duplicate key") ||
		contains(errMsg, "23505") ||
		contains(errMsg, "already exists")
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
