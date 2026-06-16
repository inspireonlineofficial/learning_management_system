package enrollments

import (
	"context"
	"lms-backend/internal/domain/auth"
	"lms-backend/internal/domain/courses"
	"lms-backend/internal/domain/enrollments"
	"lms-backend/pkg/apperrors"
	"testing"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// **Property 41: Free course enrollment is idempotent**
// Validates: Requirements 13.1
func TestProperty41_FreeCourseEnrollmentIsIdempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Setup mocks
		enrollmentRepo := newMockEnrollmentRepo()
		lessonProgressRepo := newMockLessonProgressRepo()
		courseRepo := newMockCourseRepo()
		lessonRepo := newMockLessonRepo()
		videoRepo := newMockVideoRepo()
		userRepo := newMockUserRepo()
		signingKeyStore := &mockSigningKeyStore{}
		storageClient := newMockStorageClient()

		service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

		// Generate random student and course IDs
		studentID := uuid.New()
		courseID := uuid.New()

		// Create a student user with profile complete
		user := &auth.User{
			ID:              studentID,
			Role:            "student",
			ProfileComplete: true,
			Email:           rapid.StringMatching("[a-z]+@[a-z]+\\.[a-z]+").Draw(t, "email"),
		}
		userRepo.users[studentID] = user

		// Create a free course
		course := &courses.Course{
			ID:        courseID,
			PriceType: courses.PriceTypeFree,
			Status:    courses.CourseStatusPublished,
			Title:     rapid.String().Draw(t, "title"),
		}
		courseRepo.courses[courseID] = course

		// First enrollment attempt
		cmd := EnrollFreeCommand{
			StudentID: studentID,
			CourseID:  courseID,
		}

		result1, err1 := service.EnrollFree(context.Background(), cmd)
		if err1 != nil {
			t.Fatalf("First enrollment failed: %v", err1)
		}

		if result1 == nil {
			t.Fatal("First enrollment returned nil result")
		}

		// Second enrollment attempt (duplicate)
		result2, err2 := service.EnrollFree(context.Background(), cmd)

		// Property: Second enrollment must return ALREADY_ENROLLED error
		if err2 == nil {
			t.Fatal("Expected ALREADY_ENROLLED error on duplicate enrollment, got nil")
		}

		appErr, ok := err2.(*apperrors.AppError)
		if !ok {
			t.Fatalf("Expected AppError, got %T", err2)
		}

		if appErr.Code != "ALREADY_ENROLLED" {
			t.Fatalf("Expected error code ALREADY_ENROLLED, got %s", appErr.Code)
		}

		// Property: No second enrollment record should be created
		if result2 != nil {
			t.Fatal("Second enrollment should not return a result")
		}

		// Verify only one enrollment exists in the repository
		enrollments, _, _ := enrollmentRepo.FindByStudentID(context.Background(), studentID, 1, 10)
		if len(enrollments) != 1 {
			t.Fatalf("Expected exactly 1 enrollment, got %d", len(enrollments))
		}
	})
}

// **Property 42: Streaming URL access control — enrolled or free preview**
// Validates: Requirements 13.3
func TestProperty42_StreamingURLAccessControl(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Setup mocks
		enrollmentRepo := newMockEnrollmentRepo()
		lessonProgressRepo := newMockLessonProgressRepo()
		courseRepo := newMockCourseRepo()
		lessonRepo := newMockLessonRepo()
		videoRepo := newMockVideoRepo()
		userRepo := newMockUserRepo()
		signingKeyStore := &mockSigningKeyStore{}
		storageClient := newMockStorageClient()

		service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

		// Generate random IDs
		userID := uuid.New()
		lessonID := uuid.New()
		videoID := uuid.New()
		courseID := uuid.New()

		// Create a user with profile complete
		user := &auth.User{
			ID:              userID,
			Role:            "student",
			ProfileComplete: true,
			Email:           rapid.StringMatching("[a-z]+@[a-z]+\\.[a-z]+").Draw(t, "email"),
		}
		userRepo.users[userID] = user

		// Generate random is_free_preview flag
		isFreePreview := rapid.Bool().Draw(t, "is_free_preview")

		// Create a lesson
		lesson := &courses.Lesson{
			ID:            lessonID,
			VideoID:       &videoID,
			IsFreePreview: isFreePreview,
			Title:         rapid.String().Draw(t, "title"),
		}
		lessonRepo.lessons[lessonID] = lesson

		// Create a video
		video := &courses.Video{
			ID:        videoID,
			CourseID:  courseID,
			RustFSKey: "videos/" + uuid.New().String() + ".mp4",
			Status:    courses.VideoStatusReady,
		}
		videoRepo.videos[videoID] = video

		// Generate random enrollment status
		hasEnrollment := rapid.Bool().Draw(t, "has_enrollment")

		if hasEnrollment {
			// Create an active enrollment
			enrollment := &enrollments.Enrollment{
				ID:        uuid.New(),
				StudentID: userID,
				CourseID:  courseID,
				Status:    enrollments.EnrollmentStatusActive,
			}
			key := userID.String() + ":" + courseID.String()
			enrollmentRepo.enrollments[key] = enrollment
		}

		// Execute
		cmd := GetStreamingSignedURLCommand{
			UserID:   userID,
			LessonID: lessonID,
		}

		result, err := service.GetStreamingSignedURL(context.Background(), cmd)

		// Property: Access granted if and only if (is_free_preview OR has_active_enrollment)
		shouldHaveAccess := isFreePreview || hasEnrollment

		if shouldHaveAccess {
			// Should succeed
			if err != nil {
				t.Fatalf("Expected access to be granted (isFreePreview=%v, hasEnrollment=%v), but got error: %v", isFreePreview, hasEnrollment, err)
			}

			if result == nil {
				t.Fatal("Expected result, got nil")
			}

			if result.SignedURL == "" {
				t.Fatal("Expected signed URL, got empty string")
			}
		} else {
			// Should fail with NOT_ENROLLED
			if err == nil {
				t.Fatalf("Expected access to be denied (isFreePreview=%v, hasEnrollment=%v), but got success", isFreePreview, hasEnrollment)
			}

			appErr, ok := err.(*apperrors.AppError)
			if !ok {
				t.Fatalf("Expected AppError, got %T", err)
			}

			if appErr.Code != "NOT_ENROLLED" {
				t.Fatalf("Expected error code NOT_ENROLLED, got %s", appErr.Code)
			}
		}
	})
}

// **Property 43: Lesson marked complete when watched_percent >= 80 and completed flag submitted**
// Validates: Requirements 13.6
func TestProperty43_LessonMarkedCompleteWhenWatchedPercentAbove80(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Setup mocks
		enrollmentRepo := newMockEnrollmentRepo()
		lessonProgressRepo := newMockLessonProgressRepo()
		courseRepo := newMockCourseRepo()
		lessonRepo := newMockLessonRepo()
		videoRepo := newMockVideoRepo()
		userRepo := newMockUserRepo()
		signingKeyStore := &mockSigningKeyStore{}
		storageClient := newMockStorageClient()

		service := NewService(enrollmentRepo, lessonProgressRepo, courseRepo, lessonRepo, videoRepo, userRepo, signingKeyStore, storageClient, "videos")

		// Generate random IDs
		enrollmentID := uuid.New()
		lessonID := uuid.New()
		studentID := uuid.New()
		courseID := uuid.New()

		// Create an active enrollment
		enrollment := &enrollments.Enrollment{
			ID:        enrollmentID,
			StudentID: studentID,
			CourseID:  courseID,
			Status:    enrollments.EnrollmentStatusActive,
		}
		key := studentID.String() + ":" + courseID.String()
		enrollmentRepo.enrollments[key] = enrollment

		// Generate random watched_percent (0-100)
		watchedPercent := rapid.Float64Range(0.0, 100.0).Draw(t, "watched_percent")

		// Generate random completed flag
		completedFlag := rapid.Bool().Draw(t, "completed_flag")

		// Generate random position
		positionSeconds := rapid.IntRange(0, 3600).Draw(t, "position_seconds")

		// Execute
		cmd := UpdateLessonProgressCommand{
			EnrollmentID:    enrollmentID,
			LessonID:        lessonID,
			PositionSeconds: positionSeconds,
			WatchedPercent:  watchedPercent,
			Completed:       completedFlag,
		}

		result, err := service.UpdateLessonProgress(context.Background(), cmd)

		// Assert no error
		if err != nil {
			t.Fatalf("UpdateLessonProgress failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		// Property: Lesson marked complete if and only if (completed_flag AND watched_percent >= 80)
		shouldBeComplete := completedFlag && watchedPercent >= 80.0

		if shouldBeComplete {
			// Should be marked as completed
			if !result.Completed {
				t.Fatalf("Expected lesson to be marked complete (completed=%v, watched_percent=%.2f), but got Completed=%v", completedFlag, watchedPercent, result.Completed)
			}

			if result.CompletedAt == nil {
				t.Fatal("Expected CompletedAt to be set when lesson is complete")
			}

			// Verify in repository
			progressKey := enrollmentID.String() + ":" + lessonID.String()
			storedProgress, exists := lessonProgressRepo.progress[progressKey]
			if !exists {
				t.Fatal("Expected progress to be stored in repository")
			}

			if !storedProgress.Completed {
				t.Fatal("Expected stored progress to be marked as completed")
			}

			if storedProgress.CompletedAt == nil {
				t.Fatal("Expected stored progress to have CompletedAt set")
			}
		} else {
			// Should NOT be marked as completed
			if result.Completed {
				t.Fatalf("Expected lesson NOT to be marked complete (completed=%v, watched_percent=%.2f), but got Completed=%v", completedFlag, watchedPercent, result.Completed)
			}

			if result.CompletedAt != nil {
				t.Fatal("Expected CompletedAt to be nil when lesson is not complete")
			}

			// Verify in repository
			progressKey := enrollmentID.String() + ":" + lessonID.String()
			storedProgress, exists := lessonProgressRepo.progress[progressKey]
			if !exists {
				t.Fatal("Expected progress to be stored in repository")
			}

			if storedProgress.Completed {
				t.Fatal("Expected stored progress NOT to be marked as completed")
			}

			if storedProgress.CompletedAt != nil {
				t.Fatal("Expected stored progress to have CompletedAt as nil")
			}
		}
	})
}
