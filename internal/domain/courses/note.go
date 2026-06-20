package courses

import (
	"time"

	"github.com/google/uuid"
)

// CourseNote is teacher-authored downloadable or rich-text material attached to a course,
// module, or lesson.
type CourseNote struct {
	ID          uuid.UUID
	CourseID    uuid.UUID
	ModuleID    *uuid.UUID
	LessonID    *uuid.UUID
	Title       string
	Content     string
	FileURL     string
	IsFree      bool
	IsPublished bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}
