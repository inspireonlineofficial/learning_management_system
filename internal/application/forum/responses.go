package forum

import (
	"time"

	domainforum "lms-backend/internal/domain/forum"

	"github.com/google/uuid"
)

// PostResponse is the public-facing forum post representation.
// Requirements: 21.1, 21.2
type PostResponse struct {
	ID           uuid.UUID              `json:"id"`
	AuthorID     uuid.UUID              `json:"author_id"`
	CourseID     *uuid.UUID             `json:"course_id,omitempty"`
	Title        string                 `json:"title"`
	BodyMarkdown string                 `json:"body_markdown"`
	BodyHTML     string                 `json:"body_html"`
	Upvotes      int                    `json:"upvotes"`
	FlagCount    int                    `json:"flag_count"`
	Status       domainforum.PostStatus `json:"status" swaggertype:"string" enums:"pending,active,rejected,removed"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// PostListResponse wraps a paginated list of posts.
type PostListResponse struct {
	Data []*PostResponse        `json:"data"`
	Meta map[string]interface{} `json:"meta"`
}

// CommentResponse is the public-facing forum comment representation.
type CommentResponse struct {
	ID           uuid.UUID                 `json:"id"`
	PostID       uuid.UUID                 `json:"post_id"`
	AuthorID     uuid.UUID                 `json:"author_id"`
	BodyMarkdown string                    `json:"body_markdown"`
	BodyHTML     string                    `json:"body_html"`
	Status       domainforum.CommentStatus `json:"status" swaggertype:"string" enums:"active,removed"`
	CreatedAt    time.Time                 `json:"created_at"`
	UpdatedAt    time.Time                 `json:"updated_at"`
}

// CommentListResponse wraps a paginated list of comments.
type CommentListResponse struct {
	Data []*CommentResponse     `json:"data"`
	Meta map[string]interface{} `json:"meta"`
}

// UpvoteResponse is returned after toggling an upvote.
// Requirements: 21.4
type UpvoteResponse struct {
	PostID      uuid.UUID `json:"post_id"`
	Upvotes     int       `json:"upvotes"`
	UserUpvoted bool      `json:"user_upvoted"`
}

// FlagResponse is returned after flagging content.
// Requirements: 21.5
type FlagResponse struct {
	ID         uuid.UUID                  `json:"id"`
	TargetType domainforum.FlagTargetType `json:"target_type" swaggertype:"string" enums:"post,comment"`
	TargetID   uuid.UUID                  `json:"target_id"`
	Reason     domainforum.FlagReason     `json:"reason" swaggertype:"string" enums:"spam,offensive,misinformation,other"`
	Status     domainforum.FlagStatus     `json:"status" swaggertype:"string" enums:"pending,actioned,dismissed"`
	CreatedAt  time.Time                  `json:"created_at"`
}

// ModerationQueueItem represents a flagged post or comment in the moderation queue.
// Requirements: 21.7
type ModerationQueueItem struct {
	FlagID         uuid.UUID                  `json:"flag_id"`
	TargetType     domainforum.FlagTargetType `json:"target_type" swaggertype:"string" enums:"post,comment"`
	TargetID       uuid.UUID                  `json:"target_id"`
	ContentPreview string                     `json:"content_preview"`
	Reason         domainforum.FlagReason     `json:"reason" swaggertype:"string" enums:"spam,offensive,misinformation,other"`
	Note           *string                    `json:"note,omitempty"`
	Status         domainforum.FlagStatus     `json:"status" swaggertype:"string" enums:"pending,actioned,dismissed"`
	CreatedAt      time.Time                  `json:"created_at"`
}

// ModerationQueueResponse wraps a paginated moderation queue.
type ModerationQueueResponse struct {
	Data []*ModerationQueueItem `json:"data"`
	Meta map[string]interface{} `json:"meta"`
}

// ModerationActionResponse is returned after a moderation action.
// Requirements: 21.7, 21.8
type ModerationActionResponse struct {
	FlagID  uuid.UUID `json:"flag_id"`
	Action  string    `json:"action"`
	Success bool      `json:"success"`
}

// toPostResponse converts a domain ForumPost to a PostResponse.
func toPostResponse(p *domainforum.ForumPost) *PostResponse {
	return &PostResponse{
		ID:           p.ID,
		AuthorID:     p.AuthorID,
		CourseID:     p.CourseID,
		Title:        p.Title,
		BodyMarkdown: p.BodyMarkdown,
		BodyHTML:     p.BodyHTML,
		Upvotes:      p.Upvotes,
		FlagCount:    p.FlagCount,
		Status:       p.Status,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

// toCommentResponse converts a domain ForumComment to a CommentResponse.
func toCommentResponse(c *domainforum.ForumComment) *CommentResponse {
	return &CommentResponse{
		ID:           c.ID,
		PostID:       c.PostID,
		AuthorID:     c.AuthorID,
		BodyMarkdown: c.BodyMarkdown,
		BodyHTML:     c.BodyHTML,
		Status:       c.Status,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}
