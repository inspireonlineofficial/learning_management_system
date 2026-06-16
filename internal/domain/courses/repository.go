package courses

import (
	"context"

	"github.com/google/uuid"
)

// CourseRepository defines the interface for course persistence
type CourseRepository interface {
	Create(ctx context.Context, course *Course) error
	FindByID(ctx context.Context, id uuid.UUID) (*Course, error)
	FindBySlug(ctx context.Context, slug string) (*Course, error)
	FindByTeacherID(ctx context.Context, teacherID uuid.UUID, page, limit int) ([]*Course, int, error)
	Update(ctx context.Context, course *Course) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filters CourseFilters, page, limit int) ([]*Course, int, error)
	CountPublishedLessons(ctx context.Context, courseID uuid.UUID) (int, error)
}

// CourseFilters defines filtering options for course listing
type CourseFilters struct {
	Search    string
	Subject   string
	Level     CourseLevel
	PriceType PriceType
	MinPrice  *float64
	MaxPrice  *float64
	Status    CourseStatus
	SortBy    string // "newest", "popular", "rating", "price_asc", "price_desc"
}

// ModuleRepository defines the interface for module persistence
type ModuleRepository interface {
	Create(ctx context.Context, module *Module) error
	FindByID(ctx context.Context, id uuid.UUID) (*Module, error)
	FindByCourseID(ctx context.Context, courseID uuid.UUID) ([]*Module, error)
	Update(ctx context.Context, module *Module) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	CascadeSoftDelete(ctx context.Context, id uuid.UUID) error
	Reorder(ctx context.Context, courseID uuid.UUID, positions map[uuid.UUID]int) error
}

// ChapterRepository defines the interface for chapter persistence
type ChapterRepository interface {
	Create(ctx context.Context, chapter *Chapter) error
	FindByID(ctx context.Context, id uuid.UUID) (*Chapter, error)
	FindByModuleID(ctx context.Context, moduleID uuid.UUID) ([]*Chapter, error)
	Update(ctx context.Context, chapter *Chapter) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	CascadeSoftDelete(ctx context.Context, id uuid.UUID) error
	Reorder(ctx context.Context, moduleID uuid.UUID, positions map[uuid.UUID]int) error
}

// LessonRepository defines the interface for lesson persistence
type LessonRepository interface {
	Create(ctx context.Context, lesson *Lesson) error
	FindByID(ctx context.Context, id uuid.UUID) (*Lesson, error)
	FindByChapterID(ctx context.Context, chapterID uuid.UUID) ([]*Lesson, error)
	Update(ctx context.Context, lesson *Lesson) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Reorder(ctx context.Context, chapterID uuid.UUID, positions map[uuid.UUID]int) error
}

// VideoRepository defines the interface for video persistence
type VideoRepository interface {
	Create(ctx context.Context, video *Video) error
	FindByID(ctx context.Context, id uuid.UUID) (*Video, error)
	Update(ctx context.Context, video *Video) error
}

// CourseReviewRepository defines the interface for course review persistence
type CourseReviewRepository interface {
	Upsert(ctx context.Context, review *CourseReview) error
	FindByID(ctx context.Context, id uuid.UUID) (*CourseReview, error)
	FindByCourseID(ctx context.Context, courseID uuid.UUID, page, limit int) ([]*CourseReview, int, error)
	FindByStudentAndCourse(ctx context.Context, studentID, courseID uuid.UUID) (*CourseReview, error)
	GetRatingDistribution(ctx context.Context, courseID uuid.UUID) (map[int]int, error)
}
