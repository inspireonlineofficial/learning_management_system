package courses

import (
	"context"
	"time"

	"lms-backend/internal/domain/courses"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// UpsertCourseReview creates or updates a course review
func (s *service) UpsertCourseReview(ctx context.Context, cmd UpsertCourseReviewCommand) (*CourseReviewResponse, error) {
	// Validate rating
	if cmd.Rating < 1 || cmd.Rating > 5 {
		return nil, apperrors.NewSimpleValidationError("INVALID_RATING", "rating must be between 1 and 5")
	}

	// Validate comment length
	if len(cmd.Comment) > 1000 {
		return nil, apperrors.NewSimpleValidationError("COMMENT_TOO_LONG", "comment must be 1000 characters or less")
	}

	// Check if course exists and is published
	course, err := s.courseRepo.FindByID(ctx, cmd.CourseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "course not found")
	}

	if course.Status != courses.CourseStatusPublished {
		return nil, apperrors.NewSimpleValidationError("COURSE_NOT_PUBLISHED", "can only review published courses")
	}

	// TODO: Check if student is enrolled in the course

	review := &courses.CourseReview{
		ID:        uuid.New(),
		CourseID:  cmd.CourseID,
		StudentID: cmd.StudentID,
		Rating:    cmd.Rating,
		Comment:   cmd.Comment,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.reviewRepo.Upsert(ctx, review)
	if err != nil {
		return nil, err
	}

	return &CourseReviewResponse{
		ID:        review.ID,
		CourseID:  review.CourseID,
		StudentID: review.StudentID,
		Rating:    review.Rating,
		Comment:   review.Comment,
		CreatedAt: review.CreatedAt,
		UpdatedAt: review.UpdatedAt,
	}, nil
}

// ListCourseReviews returns paginated reviews with rating distribution
func (s *service) ListCourseReviews(ctx context.Context, courseID uuid.UUID, page, limit int) (*CourseReviewsResponse, error) {
	reviews, total, err := s.reviewRepo.FindByCourseID(ctx, courseID, page, limit)
	if err != nil {
		return nil, err
	}

	distribution, err := s.reviewRepo.GetRatingDistribution(ctx, courseID)
	if err != nil {
		return nil, err
	}

	reviewResponses := make([]CourseReviewResponse, 0, len(reviews))
	for _, review := range reviews {
		reviewResponses = append(reviewResponses, CourseReviewResponse{
			ID:        review.ID,
			CourseID:  review.CourseID,
			StudentID: review.StudentID,
			Rating:    review.Rating,
			Comment:   review.Comment,
			CreatedAt: review.CreatedAt,
			UpdatedAt: review.UpdatedAt,
		})
	}

	distResponse := RatingDistributionResponse{
		Rating1: distribution[1],
		Rating2: distribution[2],
		Rating3: distribution[3],
		Rating4: distribution[4],
		Rating5: distribution[5],
	}

	// Calculate pagination meta
	totalPages := (total + limit - 1) / limit
	meta := map[string]interface{}{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": totalPages,
	}

	return &CourseReviewsResponse{
		Reviews:      reviewResponses,
		Distribution: distResponse,
		Meta:         meta,
	}, nil
}
