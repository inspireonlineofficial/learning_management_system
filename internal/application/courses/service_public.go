package courses

import (
	"context"

	"lms-backend/internal/domain/courses"

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
		return nil, err
	}

	// Public endpoint should only return published courses
	if course.Status != courses.CourseStatusPublished {
		return nil, err
	}

	return s.buildCourseDetailResponse(ctx, course, studentID)
}
