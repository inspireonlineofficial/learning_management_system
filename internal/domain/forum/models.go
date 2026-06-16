package forum

import (
	"time"

	"github.com/google/uuid"
)

// PostStatus represents the lifecycle status of a forum post.
type PostStatus string

const (
	PostStatusActive   PostStatus = "active"
	PostStatusPending  PostStatus = "pending"
	PostStatusRejected PostStatus = "rejected"
	PostStatusRemoved  PostStatus = "removed"
)

type ForumPostReview struct {
	ID         uuid.UUID
	PostID     uuid.UUID
	ReviewerID uuid.UUID
	Action     string
	Reason     string
	CreatedAt  time.Time
}

// CommentStatus represents the lifecycle status of a forum comment.
type CommentStatus string

const (
	CommentStatusActive  CommentStatus = "active"
	CommentStatusRemoved CommentStatus = "removed"
)

// FlagTargetType represents the type of content being flagged.
type FlagTargetType string

const (
	FlagTargetPost    FlagTargetType = "post"
	FlagTargetComment FlagTargetType = "comment"
)

// FlagReason represents the reason for flagging content.
type FlagReason string

const (
	FlagReasonSpam           FlagReason = "spam"
	FlagReasonOffensive      FlagReason = "offensive"
	FlagReasonMisinformation FlagReason = "misinformation"
	FlagReasonOther          FlagReason = "other"
)

// FlagStatus represents the moderation status of a content flag.
type FlagStatus string

const (
	FlagStatusPending   FlagStatus = "pending"
	FlagStatusActioned  FlagStatus = "actioned"
	FlagStatusDismissed FlagStatus = "dismissed"
)

// ForumPost is the aggregate root for the forum context.
// Supports soft-delete via DeletedAt.
// Requirements: 21.1, 21.2, 21.3
type ForumPost struct {
	ID           uuid.UUID  `json:"id"`
	AuthorID     uuid.UUID  `json:"author_id"`
	CourseID     *uuid.UUID `json:"course_id,omitempty"` // nullable
	Title        string     `json:"title"`
	BodyMarkdown string     `json:"body_markdown"`
	BodyHTML     string     `json:"body_html"` // sanitised, stored for fast reads
	Upvotes      int        `json:"upvotes"`
	FlagCount    int        `json:"flag_count"`
	Status       PostStatus `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

// ForumComment is an entity belonging to a ForumPost.
// Supports soft-delete via DeletedAt.
// Requirements: 21.1
type ForumComment struct {
	ID           uuid.UUID     `json:"id"`
	PostID       uuid.UUID     `json:"post_id"`
	AuthorID     uuid.UUID     `json:"author_id"`
	BodyMarkdown string        `json:"body_markdown"`
	BodyHTML     string        `json:"body_html"` // sanitised
	FlagCount    int           `json:"flag_count"`
	Status       CommentStatus `json:"status"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	DeletedAt    *time.Time    `json:"deleted_at,omitempty"`
}

// PostUpvote is a value object representing a user's upvote on a post.
// PRIMARY KEY(post_id, user_id).
// Requirements: 21.4
type PostUpvote struct {
	PostID uuid.UUID `json:"post_id"`
	UserID uuid.UUID `json:"user_id"`
}

// ContentFlag is an entity representing a user's flag on a post or comment.
// Requirements: 21.5, 21.6
type ContentFlag struct {
	ID         uuid.UUID      `json:"id"`
	ReporterID uuid.UUID      `json:"reporter_id"`
	TargetType FlagTargetType `json:"target_type"`
	TargetID   uuid.UUID      `json:"target_id"`
	Reason     FlagReason     `json:"reason"`
	Note       *string        `json:"note,omitempty"` // required when reason = other
	Status     FlagStatus     `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
}
