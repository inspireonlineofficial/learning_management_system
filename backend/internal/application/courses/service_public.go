package courses

import (
	"context"

	"lms-backend/internal/domain/courses"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// ListPublishedCourses returns paginated list of published courses
func (s *service) ListPublishedCourses(ctx context.Context, filters courses.CourseFilters, page, limit int) ([]CourseResponse, int, error) {
	// Force status to published for public listing
	filters.Status = courses.CourseStatusPublished

	courseList, total, err := s.courseRepo.List(ctx, filters, page, limit)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]CourseResponse, 0, len(courseList))
	for _, course := range courseList {
		responses = append(responses, *s.toCourseResponse(course))
	}

	return responses, total, nil
}

// GetCourseDetail returns full course detail with content tree
func (s *service) GetCourseDetail(ctx context.Context, courseID uuid.UUID, studentID *uuid.UUID) (*CourseDetailResponse, error) {
	course, err := s.courseRepo.FindByID(ctx, courseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}

	// Public endpoint should only return published courses. (Previously this
	// returned (nil, nil) — a 200 response with a "null" body — for drafts.)
	if course.Status != courses.CourseStatusPublished {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}

	return s.buildCourseDetail(ctx, course, studentID, true)
}
