package courses

import (
	"context"
	"io"
	"log"
	"time"

	"lms-backend/internal/domain/courses"
	"lms-backend/internal/domain/notifications"
	tsclient "lms-backend/internal/infrastructure/typesense"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// Service defines the interface for course use cases
type Service interface {
	// Teacher operations
	CreateCourse(ctx context.Context, cmd CreateCourseCommand) (*CourseResponse, error)
	UpdateCourse(ctx context.Context, cmd UpdateCourseCommand) (*CourseResponse, error)
	SubmitCourse(ctx context.Context, cmd SubmitCourseCommand) error
	GetTeacherCoursePreview(ctx context.Context, courseID, teacherID uuid.UUID) (*CourseDetailResponse, error)
	ListTeacherCourses(ctx context.Context, teacherID uuid.UUID, page, limit int) ([]CourseResponse, int, error)

	// Admin operations
	ApproveCourse(ctx context.Context, cmd ApproveCourseCommand) error
	RejectCourse(ctx context.Context, cmd RejectCourseCommand) error
	ListPendingCourses(ctx context.Context, page, limit int) ([]CourseResponse, int, error)
	GetAdminCourseDetail(ctx context.Context, courseID uuid.UUID) (*CourseDetailResponse, error)

	// Content tree management
	CreateModule(ctx context.Context, cmd CreateModuleCommand) (*ModuleResponse, error)
	UpdateModule(ctx context.Context, cmd UpdateModuleCommand) (*ModuleResponse, error)
	DeleteModule(ctx context.Context, cmd DeleteModuleCommand) error

	CreateChapter(ctx context.Context, cmd CreateChapterCommand) (*ChapterResponse, error)
	UpdateChapter(ctx context.Context, cmd UpdateChapterCommand) (*ChapterResponse, error)
	DeleteChapter(ctx context.Context, cmd DeleteChapterCommand) error

	CreateLesson(ctx context.Context, cmd CreateLessonCommand) (*LessonResponse, error)
	UpdateLesson(ctx context.Context, cmd UpdateLessonCommand) (*LessonResponse, error)
	DeleteLesson(ctx context.Context, cmd DeleteLessonCommand) error

	ReorderContent(ctx context.Context, cmd ReorderContentCommand) error

	// Public operations
	ListPublishedCourses(ctx context.Context, filters courses.CourseFilters, page, limit int) ([]CourseResponse, int, error)
	GetCourseDetail(ctx context.Context, courseID uuid.UUID, studentID *uuid.UUID) (*CourseDetailResponse, error)

	// Reviews
	UpsertCourseReview(ctx context.Context, cmd UpsertCourseReviewCommand) (*CourseReviewResponse, error)
	ListCourseReviews(ctx context.Context, courseID uuid.UUID, page, limit int) (*CourseReviewsResponse, error)

	// Video and file uploads
	UploadVideo(ctx context.Context, cmd UploadVideoCommand) (*VideoStatusResponse, error)
	GetVideoStatus(ctx context.Context, videoID uuid.UUID) (*VideoStatusResponse, error)
	UploadFile(ctx context.Context, cmd UploadFileCommand) (*FileUploadResponse, error)
}

type service struct {
	courseRepo  courses.CourseRepository
	moduleRepo  courses.ModuleRepository
	chapterRepo courses.ChapterRepository
	lessonRepo  courses.LessonRepository
	videoRepo   courses.VideoRepository
	reviewRepo  courses.CourseReviewRepository
	indexer     tsclient.Indexer
	storage     StorageClient
	jobQueue    notifications.JobQueue
	videoBucket string
	filesBucket string
	// Add audit logger and notification queue when implementing admin operations
}

type StorageClient interface {
	PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) error
	PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)
}

// NewService creates a new course service
func NewService(
	courseRepo courses.CourseRepository,
	moduleRepo courses.ModuleRepository,
	chapterRepo courses.ChapterRepository,
	lessonRepo courses.LessonRepository,
	videoRepo courses.VideoRepository,
	reviewRepo courses.CourseReviewRepository,
	indexer tsclient.Indexer,
) Service {
	return &service{
		courseRepo:  courseRepo,
		moduleRepo:  moduleRepo,
		chapterRepo: chapterRepo,
		lessonRepo:  lessonRepo,
		videoRepo:   videoRepo,
		reviewRepo:  reviewRepo,
		indexer:     indexer,
	}
}

func NewServiceWithUploadDeps(
	courseRepo courses.CourseRepository,
	moduleRepo courses.ModuleRepository,
	chapterRepo courses.ChapterRepository,
	lessonRepo courses.LessonRepository,
	videoRepo courses.VideoRepository,
	reviewRepo courses.CourseReviewRepository,
	indexer tsclient.Indexer,
	storage StorageClient,
	jobQueue notifications.JobQueue,
	videoBucket string,
	filesBucket string,
) Service {
	svc := NewService(courseRepo, moduleRepo, chapterRepo, lessonRepo, videoRepo, reviewRepo, indexer).(*service)
	svc.storage = storage
	svc.jobQueue = jobQueue
	svc.videoBucket = videoBucket
	svc.filesBucket = filesBucket
	return svc
}

// CreateCourse creates a new course in draft status
func (s *service) CreateCourse(ctx context.Context, cmd CreateCourseCommand) (*CourseResponse, error) {
	normalizedPriceType, normalizedPrice, normalizedCurrency, err := validateCoursePricing("", cmd.PriceType, cmd.Price, cmd.Currency, 0)
	if err != nil {
		return nil, err
	}
	course := &courses.Course{
		ID:               uuid.New(),
		TeacherID:        cmd.TeacherID,
		Title:            cmd.Title,
		Slug:             cmd.Slug,
		ShortDescription: cmd.ShortDescription,
		Description:      cmd.Description,
		Subject:          cmd.Subject,
		Level:            courses.CourseLevel(cmd.Level),
		PriceType:        normalizedPriceType,
		Price:            normalizedPrice,
		Currency:         normalizedCurrency,
		Prerequisites:    cmd.Prerequisites,
		ThumbnailURL:     cmd.ThumbnailURL,
		Status:           courses.CourseStatusDraft,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err = s.courseRepo.Create(ctx, course)
	if err != nil {
		return nil, err
	}

	return s.toCourseResponse(course), nil
}

// UpdateCourse updates an existing course
func (s *service) UpdateCourse(ctx context.Context, cmd UpdateCourseCommand) (*CourseResponse, error) {
	course, err := s.courseRepo.FindByID(ctx, cmd.CourseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}

	// Check ownership
	if !course.IsOwnedBy(cmd.TeacherID) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to update this course")
	}

	// Check if course is editable
	if !course.IsEditable() {
		return nil, apperrors.NewValidationErrorWithDetails("COURSE_LOCKED", "course is pending review and cannot be edited", nil)
	}

	normalizedPriceType, normalizedPrice, normalizedCurrency, err := validateCoursePricing(course.PriceType, cmd.PriceType, cmd.Price, cmd.Currency, course.TotalEnrolled)
	if err != nil {
		return nil, err
	}

	// Update fields
	course.Title = cmd.Title
	course.Slug = cmd.Slug
	course.ShortDescription = cmd.ShortDescription
	course.Description = cmd.Description
	course.Subject = cmd.Subject
	course.Level = courses.CourseLevel(cmd.Level)
	course.PriceType = normalizedPriceType
	course.Price = normalizedPrice
	course.Currency = normalizedCurrency
	course.Prerequisites = cmd.Prerequisites
	course.ThumbnailURL = cmd.ThumbnailURL

	err = s.courseRepo.Update(ctx, course)
	if err != nil {
		return nil, err
	}

	if course.Status == courses.CourseStatusPublished {
		if idxErr := s.indexer.UpsertCourse(ctx, tsclient.CourseDocument{
			ID:               course.ID.String(),
			Title:            course.Title,
			Slug:             course.Slug,
			ShortDescription: course.ShortDescription,
			Subject:          string(course.Subject),
			Level:            string(course.Level),
			Status:           string(course.Status),
			RatingAverage:    float32(course.RatingAverage),
		}); idxErr != nil {
			log.Printf("typesense index error: %v", idxErr)
		}
	} else {
		if idxErr := s.indexer.DeleteCourse(ctx, course.ID.String()); idxErr != nil {
			log.Printf("typesense index error: %v", idxErr)
		}
	}

	return s.toCourseResponse(course), nil
}

// SubmitCourse submits a course for review
func (s *service) SubmitCourse(ctx context.Context, cmd SubmitCourseCommand) error {
	course, err := s.courseRepo.FindByID(ctx, cmd.CourseID)
	if err != nil {
		return apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}

	// Check ownership
	if !course.IsOwnedBy(cmd.TeacherID) {
		return apperrors.NewForbiddenError("FORBIDDEN", "not authorized to submit this course")
	}

	// Check if course has published lessons
	lessonCount, err := s.courseRepo.CountPublishedLessons(ctx, cmd.CourseID)
	if err != nil {
		return err
	}

	if !course.HasPublishedLessons(lessonCount) {
		return apperrors.NewValidationErrorWithDetails("COURSE_EMPTY", "course must have at least one published lesson", nil)
	}

	// Validate state transition
	if !course.CanTransitionTo(courses.CourseStatusPending) {
		return apperrors.NewValidationErrorWithDetails("INVALID_TRANSITION", "cannot submit course in current status", nil)
	}

	course.Status = courses.CourseStatusPending
	return s.courseRepo.Update(ctx, course)
}

// GetTeacherCoursePreview returns course preview as student would see it
func (s *service) GetTeacherCoursePreview(ctx context.Context, courseID, teacherID uuid.UUID) (*CourseDetailResponse, error) {
	course, err := s.courseRepo.FindByID(ctx, courseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}

	// Check ownership
	if !course.IsOwnedBy(teacherID) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to preview this course")
	}

	return s.buildCourseDetailResponse(ctx, course, nil)
}

// ListTeacherCourses returns the authenticated teacher's courses with pagination.
func (s *service) ListTeacherCourses(ctx context.Context, teacherID uuid.UUID, page, limit int) ([]CourseResponse, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	courseList, total, err := s.courseRepo.FindByTeacherID(ctx, teacherID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]CourseResponse, 0, len(courseList))
	for _, course := range courseList {
		responses = append(responses, *s.toCourseResponse(course))
	}

	return responses, total, nil
}

// ListPendingCourses returns admin-reviewable courses.
func (s *service) ListPendingCourses(ctx context.Context, page, limit int) ([]CourseResponse, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	courseList, total, err := s.courseRepo.List(ctx, courses.CourseFilters{
		Status: courses.CourseStatusPending,
	}, page, limit)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]CourseResponse, 0, len(courseList))
	for _, course := range courseList {
		responses = append(responses, *s.toCourseResponse(course))
	}

	return responses, total, nil
}

// GetAdminCourseDetail returns a full course detail payload for admin review.
func (s *service) GetAdminCourseDetail(ctx context.Context, courseID uuid.UUID) (*CourseDetailResponse, error) {
	course, err := s.courseRepo.FindByID(ctx, courseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}

	return s.buildCourseDetailResponse(ctx, course, nil)
}

func (s *service) toCourseResponse(course *courses.Course) *CourseResponse {
	return &CourseResponse{
		ID:               course.ID,
		TeacherID:        course.TeacherID,
		Title:            course.Title,
		Slug:             course.Slug,
		ShortDescription: course.ShortDescription,
		Description:      course.Description,
		Subject:          course.Subject,
		Level:            string(course.Level),
		PriceType:        string(course.PriceType),
		Price:            course.Price,
		Currency:         course.Currency,
		Prerequisites:    course.Prerequisites,
		ThumbnailURL:     course.ThumbnailURL,
		Status:           string(course.Status),
		RatingAverage:    course.RatingAverage,
		RatingCount:      course.RatingCount,
		TotalEnrolled:    course.TotalEnrolled,
		PublishedAt:      course.PublishedAt,
		CreatedAt:        course.CreatedAt,
		UpdatedAt:        course.UpdatedAt,
	}
}

func (s *service) buildCourseDetailResponse(ctx context.Context, course *courses.Course, studentID *uuid.UUID) (*CourseDetailResponse, error) {
	modules, err := s.moduleRepo.FindByCourseID(ctx, course.ID)
	if err != nil {
		return nil, err
	}

	moduleResponses := make([]ModuleResponse, 0, len(modules))
	for _, module := range modules {
		chapters, err := s.chapterRepo.FindByModuleID(ctx, module.ID)
		if err != nil {
			return nil, err
		}

		chapterResponses := make([]ChapterResponse, 0, len(chapters))
		for _, chapter := range chapters {
			lessons, err := s.lessonRepo.FindByChapterID(ctx, chapter.ID)
			if err != nil {
				return nil, err
			}

			lessonResponses := make([]LessonResponse, 0, len(lessons))
			for _, lesson := range lessons {
				lessonResponses = append(lessonResponses, LessonResponse{
					ID:              lesson.ID,
					ChapterID:       lesson.ChapterID,
					Title:           lesson.Title,
					Type:            string(lesson.Type),
					DurationSeconds: lesson.DurationSeconds,
					IsFreePreview:   lesson.IsFreePreview,
					IsDownloadable:  lesson.IsDownloadable,
					Position:        lesson.Position,
					Status:          string(lesson.Status),
					CreatedAt:       lesson.CreatedAt,
					UpdatedAt:       lesson.UpdatedAt,
				})
			}

			chapterResponses = append(chapterResponses, ChapterResponse{
				ID:        chapter.ID,
				ModuleID:  chapter.ModuleID,
				Title:     chapter.Title,
				Position:  chapter.Position,
				Lessons:   lessonResponses,
				CreatedAt: chapter.CreatedAt,
				UpdatedAt: chapter.UpdatedAt,
			})
		}

		moduleResponses = append(moduleResponses, ModuleResponse{
			ID:        module.ID,
			CourseID:  module.CourseID,
			Title:     module.Title,
			Position:  module.Position,
			Chapters:  chapterResponses,
			CreatedAt: module.CreatedAt,
			UpdatedAt: module.UpdatedAt,
		})
	}

	return &CourseDetailResponse{
		CourseResponse: *s.toCourseResponse(course),
		Modules:        moduleResponses,
	}, nil
}

// ApproveCourse approves a pending course
func (s *service) ApproveCourse(ctx context.Context, cmd ApproveCourseCommand) error {
	course, err := s.courseRepo.FindByID(ctx, cmd.CourseID)
	if err != nil {
		return apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}

	// Validate state transition
	if !course.CanTransitionTo(courses.CourseStatusPublished) {
		return apperrors.NewValidationErrorWithDetails("INVALID_TRANSITION", "cannot approve course in current status", nil)
	}

	now := time.Now()
	course.Status = courses.CourseStatusPublished
	course.PublishedAt = &now

	err = s.courseRepo.Update(ctx, course)
	if err != nil {
		return err
	}

	if idxErr := s.indexer.UpsertCourse(ctx, tsclient.CourseDocument{
		ID:               course.ID.String(),
		Title:            course.Title,
		Slug:             course.Slug,
		ShortDescription: course.ShortDescription,
		Subject:          string(course.Subject),
		Level:            string(course.Level),
		Status:           string(course.Status),
		RatingAverage:    float32(course.RatingAverage),
	}); idxErr != nil {
		log.Printf("typesense index error: %v", idxErr)
	}

	// TODO: Record audit log and notify teacher
	return nil
}

// RejectCourse rejects a pending course
func (s *service) RejectCourse(ctx context.Context, cmd RejectCourseCommand) error {
	if cmd.Comment == "" {
		return apperrors.NewValidationErrorWithDetails("COMMENT_REQUIRED", "rejection comment is required", nil)
	}

	course, err := s.courseRepo.FindByID(ctx, cmd.CourseID)
	if err != nil {
		return apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}

	// Validate state transition
	if !course.CanTransitionTo(courses.CourseStatusRejected) {
		return apperrors.NewValidationErrorWithDetails("INVALID_TRANSITION", "cannot reject course in current status", nil)
	}

	course.Status = courses.CourseStatusRejected

	err = s.courseRepo.Update(ctx, course)
	if err != nil {
		return err
	}

	if idxErr := s.indexer.DeleteCourse(ctx, course.ID.String()); idxErr != nil {
		log.Printf("typesense index error: %v", idxErr)
	}

	// TODO: Record audit log and notify teacher with comment
	return nil
}
