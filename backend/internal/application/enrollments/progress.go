package enrollments

import (
	"context"
	"math"
	"time"

	"lms-backend/internal/domain/courses"
	"lms-backend/internal/domain/enrollments"
	"lms-backend/pkg/apperrors"
	"lms-backend/pkg/logger"

	"github.com/google/uuid"
)

const (
	// minWatchFraction is the share of a video's duration that must have
	// elapsed (wall-clock) since the student first opened the lesson before
	// the server accepts a completion. 0.25 allows 2x playback plus some
	// seeking over already-seen parts while still rejecting "open and
	// immediately mark complete" scripts.
	minWatchFraction = 0.25
	// maxRequiredWatch caps the wait for very long videos.
	maxRequiredWatch = 20 * time.Minute
	// maxPositionSeconds guards against absurd client values (24h).
	maxPositionSeconds = 24 * 60 * 60
)

// resolveLessonCourse returns the course that owns a lesson.
func (s *service) resolveLessonCourse(ctx context.Context, lesson *courses.Lesson) (*courses.LessonPlacement, error) {
	if finder, ok := s.lessonRepo.(courses.LessonPlacementFinder); ok {
		placement, err := finder.FindLessonPlacement(ctx, lesson.ID)
		if err != nil {
			return nil, apperrors.NewNotFoundError("LESSON_NOT_FOUND", "lesson not found")
		}
		return placement, nil
	}
	// Fallback for repositories without the placement query (test doubles):
	// derive the course from the attached video.
	if lesson.VideoID != nil {
		video, err := s.videoRepo.FindByID(ctx, *lesson.VideoID)
		if err == nil && video != nil {
			placement := &courses.LessonPlacement{CourseID: video.CourseID, ModulePublished: true, CourseStatus: courses.CourseStatusPublished}
			if course, cerr := s.courseRepo.FindByID(ctx, video.CourseID); cerr == nil && course != nil {
				placement.CourseStatus = course.Status
			}
			return placement, nil
		}
	}
	return nil, apperrors.NewNotFoundError("LESSON_NOT_FOUND", "lesson not found")
}

// isVisibleToStudents reports whether a lesson is part of the published
// curriculum (published lesson inside a published module).
func isVisibleToStudents(lesson *courses.Lesson, placement *courses.LessonPlacement) bool {
	return lesson.Status == courses.LessonStatusPublished && placement.ModulePublished
}

// UpdateLessonProgress updates or creates lesson progress.
// Requirements: 13.6
//
// Hardening (audit 2026-11):
//   - the lesson must belong to the enrollment's course and be published —
//     previously any lesson ID was accepted, so completions of other courses'
//     lessons counted towards this course's progress and certificate;
//   - client values are validated/clamped (NUMERIC(5,2) overflow caused 500s);
//   - completion is monotonic (stored with OR/GREATEST in the repository);
//   - a video lesson can only be completed after a plausible amount of time
//     since the student first opened it;
//   - certificate generation failures are logged instead of failing the
//     already-persisted progress update.
func (s *service) UpdateLessonProgress(ctx context.Context, cmd UpdateLessonProgressCommand) (*LessonProgressResponse, error) {
	if math.IsNaN(cmd.WatchedPercent) || math.IsInf(cmd.WatchedPercent, 0) || cmd.WatchedPercent < 0 || cmd.WatchedPercent > 100 {
		return nil, apperrors.NewSimpleValidationError("INVALID_WATCHED_PERCENT", "watched_percent must be between 0 and 100")
	}
	if cmd.PositionSeconds < 0 || cmd.PositionSeconds > maxPositionSeconds {
		return nil, apperrors.NewSimpleValidationError("INVALID_POSITION", "position_seconds is out of range")
	}

	// Check enrollment status at application service layer (Requirement 13.7)
	enrollment, err := s.enrollmentRepo.FindByID(ctx, cmd.EnrollmentID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ENROLLMENT_NOT_FOUND", "enrollment not found")
	}

	if !enrollment.CanAccess() {
		return nil, apperrors.NewForbiddenError("ENROLLMENT_REVOKED", "enrollment has been cancelled or refunded")
	}

	lesson, err := s.lessonRepo.FindByID(ctx, cmd.LessonID)
	if err != nil || lesson == nil {
		return nil, apperrors.NewNotFoundError("LESSON_NOT_FOUND", "lesson not found")
	}
	placement, err := s.resolveLessonCourse(ctx, lesson)
	if err != nil {
		return nil, err
	}
	if placement.CourseID != enrollment.CourseID {
		return nil, apperrors.NewNotFoundError("LESSON_NOT_FOUND", "lesson does not belong to this course")
	}
	if !isVisibleToStudents(lesson, placement) {
		return nil, apperrors.NewNotFoundError("LESSON_NOT_FOUND", "lesson is not available")
	}

	existingProgress, _ := s.lessonProgressRepo.FindByEnrollmentAndLesson(ctx, cmd.EnrollmentID, cmd.LessonID)
	wasAlreadyCompleted := existingProgress != nil && existingProgress.Completed

	now := time.Now()
	progress := &enrollments.LessonProgress{
		ID:              uuid.New(),
		EnrollmentID:    cmd.EnrollmentID,
		LessonID:        cmd.LessonID,
		PositionSeconds: cmd.PositionSeconds,
		WatchedPercent:  cmd.WatchedPercent,
		Completed:       false,
		LastWatchedAt:   &now,
	}

	// Mark complete if conditions are met (Requirement 13.6):
	// completed = true AND watched_percent >= 80, plus a server-side watch
	// time plausibility check for video lessons.
	if cmd.Completed && progress.CanMarkComplete() && !wasAlreadyCompleted {
		if s.watchTimePlausible(ctx, lesson, existingProgress, now) {
			progress.MarkComplete()
		} else {
			logger.Info(ctx, "lesson completion deferred: insufficient watch time",
				"enrollment_id", cmd.EnrollmentID, "lesson_id", cmd.LessonID)
		}
	}

	// Upsert progress via repository (completion and watched% are monotonic)
	if err := s.lessonProgressRepo.Upsert(ctx, progress); err != nil {
		return nil, err
	}

	// Recalculate enrollment.progress_percent after a new completion
	if progress.Completed && !wasAlreadyCompleted {
		if err := s.enrollmentRepo.RecalculateProgressPercent(ctx, cmd.EnrollmentID); err != nil {
			logger.Error(ctx, "failed to recalculate enrollment progress", "enrollment_id", cmd.EnrollmentID, "error", err)
		}

		updatedEnrollment, err := s.enrollmentRepo.FindByID(ctx, cmd.EnrollmentID)
		if err == nil && updatedEnrollment.ProgressPercent >= 100.0 && updatedEnrollment.CompletedAt != nil {
			if err := s.issueCertificate(ctx, updatedEnrollment); err != nil {
				// The progress is saved; certificate issuance is idempotent and
				// will be retried on the next completion event.
				logger.Error(ctx, "certificate issuance failed", "enrollment_id", cmd.EnrollmentID, "error", err)
			}
		}
	}

	return s.toLessonProgressResponse(progress), nil
}

// watchTimePlausible checks that enough wall-clock time has passed since the
// student first opened a video lesson to have plausibly watched it.
func (s *service) watchTimePlausible(ctx context.Context, lesson *courses.Lesson, existing *enrollments.LessonProgress, now time.Time) bool {
	if lesson.Type != courses.LessonTypeVideo && lesson.VideoID == nil {
		return true // text/attachment lessons: "mark complete" is explicit
	}

	duration := 0
	if lesson.VideoID != nil {
		if video, err := s.videoRepo.FindByID(ctx, *lesson.VideoID); err == nil && video != nil && video.DurationSeconds > 0 {
			duration = video.DurationSeconds // measured by ffprobe at transcode time
		}
	}
	if duration == 0 {
		duration = lesson.DurationSeconds // teacher-entered fallback
	}
	if duration <= 0 {
		return true
	}

	required := time.Duration(float64(duration)*minWatchFraction) * time.Second
	if required > maxRequiredWatch {
		required = maxRequiredWatch
	}
	if existing == nil || existing.StartedAt == nil {
		// First ever progress event for this lesson: only plausible for
		// very short clips.
		return required <= 5*time.Second
	}
	return now.Sub(*existing.StartedAt) >= required
}

// GetCourseProgress returns all lesson progress for the student's enrollment
// in a course with one repository query.
func (s *service) GetCourseProgress(ctx context.Context, studentID, courseID uuid.UUID) (*CourseProgressResponse, error) {
	enrollment, err := s.enrollmentRepo.FindByStudentAndCourse(ctx, studentID, courseID)
	if err != nil || enrollment == nil {
		return nil, apperrors.NewNotFoundError("ENROLLMENT_NOT_FOUND", "enrollment not found")
	}
	rows, err := s.lessonProgressRepo.FindByEnrollmentID(ctx, enrollment.ID)
	if err != nil {
		return nil, err
	}
	resp := &CourseProgressResponse{
		EnrollmentID:    enrollment.ID,
		CourseID:        enrollment.CourseID,
		ProgressPercent: enrollment.ProgressPercent,
		CompletedAt:     enrollment.CompletedAt,
		Lessons:         make([]LessonProgressResponse, 0, len(rows)),
	}
	for _, row := range rows {
		resp.Lessons = append(resp.Lessons, *s.toLessonProgressResponse(row))
	}
	return resp, nil
}
