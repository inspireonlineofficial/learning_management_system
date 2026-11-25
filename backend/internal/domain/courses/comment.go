package courses

import (
	"time"

	"github.com/google/uuid"
)

// CourseComment represents a discussion comment attached to course content.
type CourseComment struct {
	ID              uuid.UUID
	CourseID        uuid.UUID
	ModuleID        *uuid.UUID
	LessonID        *uuid.UUID
	QuizID          *uuid.UUID
	UserID          uuid.UUID
	ParentCommentID *uuid.UUID
	Content         string
	IsPinned        bool
	DeletedAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
