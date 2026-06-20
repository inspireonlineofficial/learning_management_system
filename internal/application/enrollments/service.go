package enrollments

import (
	"context"
	"fmt"
	"time"

	appcertificates "lms-backend/internal/application/certificates"
	"lms-backend/internal/domain/auth"
	"lms-backend/internal/domain/courses"
	"lms-backend/internal/domain/enrollments"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// Service defines the interface for enrollment use cases
type Service interface {
	// Enrollment operations
	EnrollFree(ctx context.Context, cmd EnrollFreeCommand) (*EnrollmentResponse, error)
	RevokeEnrollment(ctx context.Context, cmd RevokeEnrollmentCommand) error
	GetEnrollment(ctx context.Context, studentID, courseID uuid.UUID) (*EnrollmentResponse, error)
	ListStudentEnrollments(ctx context.Context, studentID uuid.UUID, page, limit int) ([]EnrollmentResponse, int, error)

	// Progress tracking
	UpdateLessonProgress(ctx context.Context, cmd UpdateLessonProgressCommand) (*LessonProgressResponse, error)
	GetLessonProgress(ctx context.Context, enrollmentID, lessonID uuid.UUID) (*LessonProgressResponse, error)

	// Video streaming
	GetStreamingSignedURL(ctx context.Context, cmd GetStreamingSignedURLCommand) (*StreamingSignedURLResponse, error)
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
}

// SigningKeyStore defines the interface for managing user-scoped signing keys in Redis
type SigningKeyStore interface {
	InvalidateUserSigningKey(ctx context.Context, userID uuid.UUID, courseID uuid.UUID) error
}

// StorageClient defines the interface for object storage operations
type StorageClient interface {
	PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)
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
func (s *service) ListStudentEnrollments(ctx context.Context, studentID uuid.UUID, page, limit int) ([]EnrollmentResponse, int, error) {
	enrollmentList, total, err := s.enrollmentRepo.FindByStudentID(ctx, studentID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]EnrollmentResponse, 0, len(enrollmentList))
	for _, enrollment := range enrollmentList {
		responses = append(responses, *s.toEnrollmentResponse(ctx, enrollment))
	}

	return responses, total, nil
}

// UpdateLessonProgress updates or creates lesson progress
// Requirements: 13.6
func (s *service) UpdateLessonProgress(ctx context.Context, cmd UpdateLessonProgressCommand) (*LessonProgressResponse, error) {
	// Check enrollment status at application service layer (Requirement 13.7)
	enrollment, err := s.enrollmentRepo.FindByID(ctx, cmd.EnrollmentID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ENROLLMENT_NOT_FOUND", "enrollment not found")
	}

	if !enrollment.CanAccess() {
		return nil, apperrors.NewForbiddenError("ENROLLMENT_REVOKED", "enrollment has been cancelled or refunded")
	}

	// Check if lesson progress already exists to determine if this is a new completion
	existingProgress, _ := s.lessonProgressRepo.FindByEnrollmentAndLesson(ctx, cmd.EnrollmentID, cmd.LessonID)
	wasAlreadyCompleted := existingProgress != nil && existingProgress.Completed

	// Create or update lesson progress
	progress := &enrollments.LessonProgress{
		ID:              uuid.New(),
		EnrollmentID:    cmd.EnrollmentID,
		LessonID:        cmd.LessonID,
		PositionSeconds: cmd.PositionSeconds,
		WatchedPercent:  cmd.WatchedPercent,
		Completed:       false, // Will be set by MarkComplete if conditions are met
	}

	now := time.Now()
	progress.LastWatchedAt = &now

	// Mark complete if conditions are met (Requirement 13.6)
	// When completed = true AND watched_percent >= 80, mark lesson complete
	if cmd.Completed && progress.CanMarkComplete() {
		progress.MarkComplete()
	}

	// Upsert progress via repository
	err = s.lessonProgressRepo.Upsert(ctx, progress)
	if err != nil {
		return nil, err
	}

	// If lesson was just completed (not already completed before), trigger Points_Engine
	// TODO: Implement Points_Engine trigger when Phase 7 is complete
	// For now, add a comment placeholder
	if progress.Completed && !wasAlreadyCompleted {
		// TODO: Trigger Points_Engine to award video completion points
		// This will be implemented in Phase 7 (Points & Gamification)
	}

	// Recalculate enrollment.progress_percent after each completion (Requirement 13.6)
	if progress.Completed {
		err = s.enrollmentRepo.RecalculateProgressPercent(ctx, cmd.EnrollmentID)
		if err != nil {
			// Log error but don't fail the operation
			// The progress was already saved successfully
			// TODO: Add structured logging
		}

		// Check if enrollment reached 100% to trigger certificate generation
		updatedEnrollment, err := s.enrollmentRepo.FindByID(ctx, cmd.EnrollmentID)
		if err == nil && updatedEnrollment.ProgressPercent >= 100.0 && updatedEnrollment.CompletedAt != nil {
			if err := s.issueCertificate(ctx, updatedEnrollment); err != nil {
				return nil, err
			}
		}
	}

	return s.toLessonProgressResponse(progress), nil
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

// GetStreamingSignedURL generates a presigned URL for video streaming
// Requirements: 13.3, 13.4, 13.5, 7.3
func (s *service) GetStreamingSignedURL(ctx context.Context, cmd GetStreamingSignedURLCommand) (*StreamingSignedURLResponse, error) {
	// Fetch the lesson
	lesson, err := s.lessonRepo.FindByID(ctx, cmd.LessonID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("LESSON_NOT_FOUND", "lesson not found")
	}

	// Check if lesson has a video
	if lesson.VideoID == nil {
		return nil, apperrors.NewSimpleValidationError("NO_VIDEO", "this lesson does not have a video")
	}

	// Fetch the user to check profile_complete
	user, err := s.userRepo.FindByID(ctx, cmd.UserID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("USER_NOT_FOUND", "user not found")
	}

	// Check access: either free/free-preview content OR active enrollment.
	var hasAccess bool
	var courseID uuid.UUID

	if lesson.IsFreePreview || lesson.IsFree {
		// Free lessons are accessible to any authenticated user.
		hasAccess = true

		// Get course ID from lesson's chapter
		// We need to traverse: lesson -> chapter -> module -> course
		// For now, we'll need to fetch the video to get the course_id
		video, err := s.videoRepo.FindByID(ctx, *lesson.VideoID)
		if err != nil {
			return nil, apperrors.NewNotFoundError("VIDEO_NOT_FOUND", "video not found")
		}
		courseID = video.CourseID
	} else {
		// Non-preview lessons require profile_complete check (Requirement 7.3)
		if !user.ProfileComplete {
			return nil, apperrors.ErrProfileIncomplete
		}

		// Check for active enrollment (Requirement 13.3)
		// First, get the course ID from the video
		video, err := s.videoRepo.FindByID(ctx, *lesson.VideoID)
		if err != nil {
			return nil, apperrors.NewNotFoundError("VIDEO_NOT_FOUND", "video not found")
		}
		courseID = video.CourseID

		// Check enrollment status at application service layer (Requirement 13.7)
		enrollment, err := s.enrollmentRepo.FindByStudentAndCourse(ctx, cmd.UserID, courseID)
		if err != nil {
			return nil, apperrors.NewForbiddenError("NOT_ENROLLED", "you must be enrolled in this course to access this lesson")
		}

		if !enrollment.CanAccess() {
			return nil, apperrors.NewForbiddenError("ENROLLMENT_REVOKED", "your enrollment has been cancelled or refunded")
		}

		hasAccess = true
	}

	if !hasAccess {
		return nil, apperrors.NewForbiddenError("ACCESS_DENIED", "you do not have access to this lesson")
	}

	// Fetch the video to get the RustFS key
	video, err := s.videoRepo.FindByID(ctx, *lesson.VideoID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("VIDEO_NOT_FOUND", "video not found")
	}

	// Check video status
	if video.Status != courses.VideoStatusReady {
		return nil, apperrors.NewSimpleValidationError("VIDEO_NOT_READY", "video is still processing or failed")
	}

	// Generate presigned URL with 2-hour TTL (Requirement 13.4)
	ttl := 2 * time.Hour
	signedURL, err := s.storageClient.PresignGetURL(ctx, s.videoBucket, video.RustFSKey, ttl)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	// Never return raw RustFS URL (Requirement 13.5)
	// The presigned URL is scoped to the user ID via the signing key store
	// (though in this implementation, we're using the standard S3 presigning)

	return &StreamingSignedURLResponse{
		SignedURL: signedURL,
		ExpiresAt: time.Now().Add(ttl),
	}, nil
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
