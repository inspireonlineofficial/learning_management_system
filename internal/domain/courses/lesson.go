package courses

import (
	"time"

	"github.com/google/uuid"
)

// LessonType represents the type of lesson content
type LessonType string

const (
	LessonTypeVideo      LessonType = "video"
	LessonTypeText       LessonType = "text"
	LessonTypeAttachment LessonType = "attachment"
)

// LessonStatus represents the publication status of a lesson
type LessonStatus string

const (
	LessonStatusDraft     LessonStatus = "draft"
	LessonStatusPublished LessonStatus = "published"
)

// Lesson represents an atomic learning unit within a chapter
type Lesson struct {
	ID              uuid.UUID
	ChapterID       uuid.UUID
	Title           string
	Type            LessonType
	VideoID         *uuid.UUID
	DurationSeconds int
	IsFreePreview   bool
	IsDownloadable  bool
	Position        int
	Status          LessonStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}
