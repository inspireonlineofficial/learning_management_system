package courses

import (
	"io"

	"github.com/google/uuid"
)

// CreateCourseCommand represents the command to create a new course
type CreateCourseCommand struct {
	TeacherID                uuid.UUID
	Title                    string
	Slug                     string
	ShortDescription         string
	Description              string
	Subject                  string
	Level                    string
	PriceType                string
	Price                    float64
	Currency                 string
	Prerequisites            string
	Visibility               string
	LearningOutcomes         string
	Requirements             string
	TargetAudience           string
	EstimatedDurationMinutes int
	ThumbnailURL             string
}

// UpdateCourseCommand represents the command to update a course
type UpdateCourseCommand struct {
	CourseID                 uuid.UUID
	TeacherID                uuid.UUID
	Title                    string
	Slug                     string
	ShortDescription         string
	Description              string
	Subject                  string
	Level                    string
	PriceType                string
	Price                    float64
	Currency                 string
	Prerequisites            string
	Visibility               string
	LearningOutcomes         string
	Requirements             string
	TargetAudience           string
	EstimatedDurationMinutes int
	ThumbnailURL             string
}

// SubmitCourseCommand represents the command to submit a course for review
type SubmitCourseCommand struct {
	CourseID  uuid.UUID
	TeacherID uuid.UUID
}

// DeleteCourseCommand represents the command to delete a teacher-owned course.
type DeleteCourseCommand struct {
	CourseID  uuid.UUID
	TeacherID uuid.UUID
}

// ApproveCourseCommand represents the command to approve a course
type ApproveCourseCommand struct {
	CourseID uuid.UUID
	AdminID  uuid.UUID
}

// RejectCourseCommand represents the command to reject a course
type RejectCourseCommand struct {
	CourseID uuid.UUID
	AdminID  uuid.UUID
	Comment  string
}

// CreateModuleCommand represents the command to create a module
type CreateModuleCommand struct {
	CourseID    uuid.UUID
	TeacherID   uuid.UUID
	Title       string
	Description string
	Position    int
	IsFree      bool
	IsPublished bool
}

// UpdateModuleCommand represents the command to update a module
type UpdateModuleCommand struct {
	ModuleID    uuid.UUID
	TeacherID   uuid.UUID
	Title       string
	Description string
	Position    int
	IsFree      bool
	IsPublished bool
}

// DeleteModuleCommand represents the command to delete a module
type DeleteModuleCommand struct {
	ModuleID  uuid.UUID
	TeacherID uuid.UUID
}

// CreateChapterCommand represents the command to create a chapter
type CreateChapterCommand struct {
	ModuleID  uuid.UUID
	TeacherID uuid.UUID
	Title     string
	Position  int
}

// UpdateChapterCommand represents the command to update a chapter
type UpdateChapterCommand struct {
	ChapterID uuid.UUID
	TeacherID uuid.UUID
	Title     string
	Position  int
}

// DeleteChapterCommand represents the command to delete a chapter
type DeleteChapterCommand struct {
	ChapterID uuid.UUID
	TeacherID uuid.UUID
}

// CreateLessonCommand represents the command to create a lesson
type CreateLessonCommand struct {
	ChapterID       uuid.UUID
	TeacherID       uuid.UUID
	Title           string
	Description     string
	Type            string
	VideoID         *uuid.UUID
	DurationSeconds int
	IsFreePreview   bool
	IsFree          bool
	IsDownloadable  bool
	Position        int
	Status          string
}

// UpdateLessonCommand represents the command to update a lesson
type UpdateLessonCommand struct {
	LessonID        uuid.UUID
	TeacherID       uuid.UUID
	Title           string
	Description     string
	Type            string
	VideoID         *uuid.UUID
	DurationSeconds int
	IsFreePreview   bool
	IsFree          bool
	IsDownloadable  bool
	Position        int
	Status          string
}

// DeleteLessonCommand represents the command to delete a lesson
type DeleteLessonCommand struct {
	LessonID  uuid.UUID
	TeacherID uuid.UUID
}

// CreateCourseNoteCommand represents the command to create a course note.
type CreateCourseNoteCommand struct {
	CourseID    uuid.UUID
	TeacherID   uuid.UUID
	ModuleID    *uuid.UUID
	LessonID    *uuid.UUID
	Title       string
	Content     string
	FileURL     string
	IsFree      bool
	IsPublished bool
}

// UpdateCourseNoteCommand represents the command to update a course note.
type UpdateCourseNoteCommand struct {
	NoteID      uuid.UUID
	TeacherID   uuid.UUID
	ModuleID    *uuid.UUID
	LessonID    *uuid.UUID
	Title       string
	Content     string
	FileURL     string
	IsFree      bool
	IsPublished bool
}

// DeleteCourseNoteCommand represents the command to delete a course note.
type DeleteCourseNoteCommand struct {
	NoteID    uuid.UUID
	TeacherID uuid.UUID
}

// CreateCourseCommentCommand creates a discussion comment on course content.
type CreateCourseCommentCommand struct {
	CourseID        uuid.UUID
	UserID          uuid.UUID
	Role            string
	ModuleID        *uuid.UUID
	LessonID        *uuid.UUID
	QuizID          *uuid.UUID
	ParentCommentID *uuid.UUID
	Content         string
}

// UpdateCourseCommentCommand updates a discussion comment.
type UpdateCourseCommentCommand struct {
	CommentID uuid.UUID
	UserID    uuid.UUID
	Role      string
	Content   string
	IsPinned  *bool
}

// DeleteCourseCommentCommand deletes a discussion comment.
type DeleteCourseCommentCommand struct {
	CommentID uuid.UUID
	UserID    uuid.UUID
	Role      string
}

// ReorderContentCommand represents the command to reorder content
type ReorderContentCommand struct {
	TeacherID uuid.UUID
	Type      string // "module", "chapter", "lesson"
	ParentID  uuid.UUID
	Positions map[uuid.UUID]int
}

// UploadVideoCommand represents the command to upload a video
type UploadVideoCommand struct {
	CourseID   uuid.UUID
	UploaderID uuid.UUID
	FileName   string
	FileSize   int64
	MimeType   string
	MagicBytes []byte
	Reader     io.Reader
}

// UploadFileCommand represents the command to upload a file
type UploadFileCommand struct {
	UploaderID uuid.UUID
	FileName   string
	FileSize   int64
	MimeType   string
	MagicBytes []byte
	Reader     io.Reader
}

// UpsertCourseReviewCommand represents the command to create or update a review
type UpsertCourseReviewCommand struct {
	CourseID  uuid.UUID
	StudentID uuid.UUID
	Rating    int
	Comment   string
}

// DeleteCourseReviewCommand deletes a student's own course review.
type DeleteCourseReviewCommand struct {
	CourseID  uuid.UUID
	StudentID uuid.UUID
}
