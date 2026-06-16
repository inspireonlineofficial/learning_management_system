package forum

import (
	domainforum "lms-backend/internal/domain/forum"

	"github.com/google/uuid"
)

// ListPostsCommand holds filter and pagination parameters for listing forum posts.
// Requirements: 21.1
type ListPostsCommand struct {
	CourseID  *uuid.UUID
	SortOrder domainforum.PostSortOrder
	Page      int
	Limit     int
}

// CreatePostCommand holds the data for creating a new forum post.
// Requirements: 21.2, 21.3
type CreatePostCommand struct {
	AuthorID     uuid.UUID
	CourseID     *uuid.UUID
	Title        string
	BodyMarkdown string
}

// UpdatePostCommand holds the data for updating a forum post.
type UpdatePostCommand struct {
	PostID       uuid.UUID
	AuthorID     uuid.UUID
	Title        *string
	BodyMarkdown *string
}

// DeletePostCommand soft-deletes a forum post.
type DeletePostCommand struct {
	PostID  uuid.UUID
	ActorID uuid.UUID
	IsAdmin bool
}

// ListCommentsCommand holds pagination parameters for listing comments on a post.
type ListCommentsCommand struct {
	PostID uuid.UUID
	Page   int
	Limit  int
}

// CreateCommentCommand holds the data for creating a comment on a post.
// Requirements: 21.2, 21.3
type CreateCommentCommand struct {
	PostID       uuid.UUID
	AuthorID     uuid.UUID
	BodyMarkdown string
}

// UpdateCommentCommand holds the data for updating a comment.
type UpdateCommentCommand struct {
	CommentID    uuid.UUID
	AuthorID     uuid.UUID
	BodyMarkdown string
}

// DeleteCommentCommand soft-deletes a comment.
type DeleteCommentCommand struct {
	CommentID uuid.UUID
	ActorID   uuid.UUID
	IsAdmin   bool
}

// ToggleUpvoteCommand toggles an upvote on a post.
// Requirements: 21.4
type ToggleUpvoteCommand struct {
	PostID uuid.UUID
	UserID uuid.UUID
}

// FlagContentCommand creates a content flag.
// Requirements: 21.5, 21.6
type FlagContentCommand struct {
	ReporterID uuid.UUID
	TargetType domainforum.FlagTargetType
	TargetID   uuid.UUID
	Reason     domainforum.FlagReason
	Note       *string
}

// ModerateContentCommand performs a moderation action on flagged content.
// Requirements: 21.7, 21.8
type ModerateContentCommand struct {
	ActorID   uuid.UUID
	ActorName string
	FlagID    uuid.UUID
	Action    string // "remove" or "ban_user"
	Reason    string
	IPAddress string
}

// GetModerationQueueCommand holds pagination for the moderation queue.
// Requirements: 21.7
type GetModerationQueueCommand struct {
	Page  int
	Limit int
}

type ListPostsForReviewCommand struct {
	Status domainforum.PostStatus
	Page   int
	Limit  int
}

type ReviewPostCommand struct {
	ActorID   uuid.UUID
	PostID    uuid.UUID
	Reason    string
	IPAddress string
}
