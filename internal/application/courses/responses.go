package courses

import (
	"time"

	"github.com/google/uuid"
)

// CourseResponse represents a course in API responses
type CourseResponse struct {
	ID                       uuid.UUID  `json:"id"`
	TeacherID                uuid.UUID  `json:"teacher_id"`
	Title                    string     `json:"title"`
	Slug                     string     `json:"slug"`
	ShortDescription         string     `json:"short_description"`
	Description              string     `json:"description"`
	Subject                  string     `json:"subject"`
	Level                    string     `json:"level"`
	PriceType                string     `json:"price_type"`
	Price                    float64    `json:"price"`
	Currency                 string     `json:"currency"`
	Prerequisites            string     `json:"prerequisites"`
	Visibility               string     `json:"visibility"`
	LearningOutcomes         string     `json:"learning_outcomes"`
	Requirements             string     `json:"requirements"`
	TargetAudience           string     `json:"target_audience"`
	EstimatedDurationMinutes int        `json:"estimated_duration_minutes"`
	ThumbnailURL             string     `json:"thumbnail_url"`
	Status                   string     `json:"status"`
	RatingAverage            float64    `json:"rating_average"`
	RatingCount              int        `json:"rating_count"`
	TotalEnrolled            int        `json:"total_enrolled"`
	PublishedAt              *time.Time `json:"published_at"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

// CourseDetailResponse represents detailed course information with content tree
type CourseDetailResponse struct {
	CourseResponse
	Modules    []ModuleResponse  `json:"modules"`
	Notes      []NoteResponse    `json:"notes"`
	Comments   []CommentResponse `json:"comments"`
	IsEnrolled bool              `json:"is_enrolled"`
}

// ModuleResponse represents a module in API responses
type ModuleResponse struct {
	ID          uuid.UUID         `json:"id"`
	CourseID    uuid.UUID         `json:"course_id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Position    int               `json:"position"`
	IsFree      bool              `json:"is_free"`
	IsPublished bool              `json:"is_published"`
	Chapters    []ChapterResponse `json:"chapters"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// ChapterResponse represents a chapter in API responses
type ChapterResponse struct {
	ID        uuid.UUID        `json:"id"`
	ModuleID  uuid.UUID        `json:"module_id"`
	Title     string           `json:"title"`
	Position  int              `json:"position"`
	Lessons   []LessonResponse `json:"lessons"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// LessonResponse represents a lesson in API responses
type LessonResponse struct {
	ID               uuid.UUID  `json:"id"`
	ChapterID        uuid.UUID  `json:"chapter_id"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	Type             string     `json:"type"`
	DurationSeconds  int        `json:"duration_seconds"`
	HasVideo         bool       `json:"has_video"`
	IsFreePreview    bool       `json:"is_free_preview"`
	IsFree           bool       `json:"is_free"`
	IsDownloadable   bool       `json:"is_downloadable"`
	Position         int        `json:"position"`
	Status           string     `json:"status"`
	CompletionStatus *string    `json:"completion_status,omitempty"` // For enrolled students
	LastWatchedAt    *time.Time `json:"last_watched_at,omitempty"`   // For enrolled students
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// NoteResponse represents a course/module/lesson note in API responses.
type NoteResponse struct {
	ID          uuid.UUID  `json:"id"`
	CourseID    uuid.UUID  `json:"course_id"`
	ModuleID    *uuid.UUID `json:"module_id,omitempty"`
	LessonID    *uuid.UUID `json:"lesson_id,omitempty"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	FileURL     string     `json:"file_url"`
	IsFree      bool       `json:"is_free"`
	IsPublished bool       `json:"is_published"`
	IsLocked    bool       `json:"is_locked"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// VideoStatusResponse represents video processing status
type VideoStatusResponse struct {
	VideoID  uuid.UUID `json:"video_id"`
	Status   string    `json:"status"` // "processing", "ready", "failed"
	PollURL  string    `json:"poll_url,omitempty"`
	Duration int       `json:"duration_seconds,omitempty"`
}

// DirectUploadResponse is what the client gets back from InitDirectUpload.
// UploadURL is a presigned PUT URL the browser uses to upload bytes directly
// to RustFS without streaming them through the Go API process.
type DirectUploadResponse struct {
	VideoID   uuid.UUID `json:"video_id"`
	UploadURL string    `json:"upload_url"`
	RustFSKey string    `json:"rustfs_key"`
	PollURL   string    `json:"poll_url"`
}

// MultipartInitResponse is what the client gets back from
// InitMultipartUpload. The browser persists these to IndexedDB so a page
// refresh can resume from the last completed part. UploadID is the S3
// upload id and is opaque to the client; it must be replayed on every
// part request and the final completion call.
type MultipartInitResponse struct {
	VideoID     uuid.UUID `json:"video_id"`
	UploadID    string    `json:"upload_id"`
	RustFSKey   string    `json:"rustfs_key"`
	ChunkSize   int64     `json:"chunk_size"`
	TotalChunks int       `json:"total_chunks"`
	PollURL     string    `json:"poll_url"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// PresignUploadPartResponse is the presigned URL the browser uses to PUT a
// single chunk. The URL embeds the upload id and part number; it is valid
// for 1 hour, which is plenty for any single chunk.
type PresignUploadPartResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// FileUploadResponse represents file upload result
type FileUploadResponse struct {
	FileID       uuid.UUID `json:"file_id"`
	PresignedURL string    `json:"presigned_url"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// CourseReviewResponse represents a course review
type CourseReviewResponse struct {
	ID        uuid.UUID `json:"id"`
	CourseID  uuid.UUID `json:"course_id"`
	StudentID uuid.UUID `json:"student_id"`
	Rating    int       `json:"rating"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CommentResponse represents a course discussion comment.
type CommentResponse struct {
	ID              uuid.UUID  `json:"id"`
	CourseID        uuid.UUID  `json:"course_id"`
	ModuleID        *uuid.UUID `json:"module_id,omitempty"`
	LessonID        *uuid.UUID `json:"lesson_id,omitempty"`
	QuizID          *uuid.UUID `json:"quiz_id,omitempty"`
	UserID          uuid.UUID  `json:"user_id"`
	ParentCommentID *uuid.UUID `json:"parent_comment_id,omitempty"`
	Content         string     `json:"content"`
	IsPinned        bool       `json:"is_pinned"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// CommentsResponse represents paginated course comments.
type CommentsResponse struct {
	Comments []CommentResponse `json:"comments"`
	Meta     interface{}       `json:"meta"`
}

// RatingDistributionResponse represents rating distribution summary
type RatingDistributionResponse struct {
	Rating1 int `json:"rating_1"`
	Rating2 int `json:"rating_2"`
	Rating3 int `json:"rating_3"`
	Rating4 int `json:"rating_4"`
	Rating5 int `json:"rating_5"`
}

// CourseReviewsResponse represents paginated reviews with distribution
type CourseReviewsResponse struct {
	Reviews      []CourseReviewResponse     `json:"reviews"`
	Distribution RatingDistributionResponse `json:"distribution"`
	Meta         interface{}                `json:"meta"`
}
