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

// AdminDeleteCourseCommand is the admin override of DeleteCourse: it skips the
// ownership check, the "is editable" check, and the "no enrollments" check, so
// admins can clean up spam / abandoned / policy-violating courses.
type AdminDeleteCourseCommand struct {
	CourseID uuid.UUID
	AdminID  uuid.UUID
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

// CompleteVideoUploadCommand is sent by the client after a direct-to-S3 PUT
// finishes. The server verifies the object actually landed (HEAD) and flips
// the video row to "ready" so playback unblocks.
type CompleteVideoUploadCommand struct {
	VideoID  uuid.UUID
	Uploader uuid.UUID
}

// InitDirectUploadCommand is the entry point for the direct-to-RustFS upload
// path. The client posts file metadata, the server returns a presigned PUT
// URL, and the bytes travel browser → RustFS directly.
type InitDirectUploadCommand struct {
	CourseID   uuid.UUID
	UploaderID uuid.UUID
	FileName   string
	FileSize   int64
	MimeType   string
	MagicBytes []byte
}

// InitMultipartUploadCommand starts a resumable S3 multipart upload. The
// server allocates an upload id and creates a "processing" video row that
// the client references on every subsequent part-upload request. The client
// chooses a chunk size (must be ≥ 5 MB except for the last chunk per S3);
// we expose what we'd recommend via MultipartInitResponse.ChunkSize.
type InitMultipartUploadCommand struct {
	CourseID   uuid.UUID
	UploaderID uuid.UUID
	FileName   string
	FileSize   int64
	MimeType   string
	MagicBytes []byte
	ChunkSize  int64 // suggested chunk size from the client
}

// PresignUploadPartCommand returns a presigned URL for one chunk. The
// (video_id, upload_id) pair must match an upload previously initialized via
// InitMultipartUpload; we re-verify ownership on every call so a leaked
// video id can't be used to upload to someone else's video.
type PresignUploadPartCommand struct {
	VideoID    uuid.UUID
	Uploader   uuid.UUID
	UploadID   string
	PartNumber int
}

// CompleteMultipartUploadCommand finishes the upload by submitting the list
// of completed parts in order. The server validates the parts, calls S3
// CompleteMultipartUpload, and flips the video row to "ready" so playback
// can start.
type CompleteMultipartUploadCommand struct {
	VideoID  uuid.UUID
	Uploader uuid.UUID
	UploadID string
	Parts    []CompletedPart
}

// CompletedPart is the local mirror of rustfs.CompletedPart. Keeping a local
// type here avoids leaking the storage driver through the application layer.
type CompletedPart struct {
	PartNumber int
	ETag       string
}

// AbortMultipartUploadCommand cancels an in-progress upload and removes the
// "processing" video row. Safe to call multiple times.
type AbortMultipartUploadCommand struct {
	VideoID  uuid.UUID
	Uploader uuid.UUID
	UploadID string
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
