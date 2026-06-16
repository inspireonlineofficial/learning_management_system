package forum

import (
	"context"

	"github.com/google/uuid"
)

// PostSortOrder defines the sort order for listing forum posts.
type PostSortOrder string

const (
	PostSortNewest PostSortOrder = "newest"
	PostSortTop    PostSortOrder = "top"
)

// PostFilter holds filter and pagination parameters for listing posts.
type PostFilter struct {
	CourseID  *uuid.UUID
	SortOrder PostSortOrder
}

// ForumPostRepository defines the persistence interface for ForumPost.
type ForumPostRepository interface {
	Create(ctx context.Context, post *ForumPost) error
	FindByID(ctx context.Context, id uuid.UUID) (*ForumPost, error)
	Update(ctx context.Context, post *ForumPost) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	// List returns non-removed posts with pagination.
	List(ctx context.Context, filter PostFilter, page, limit int) ([]*ForumPost, int, error)
	// ListWithFlagCountGTE returns posts/comments with flag_count >= threshold (moderation queue).
	ListWithFlagCountGTE(ctx context.Context, threshold int, page, limit int) ([]*ForumPost, int, error)
	// IncrementFlagCount atomically increments the flag_count for a post.
	IncrementFlagCount(ctx context.Context, postID uuid.UUID) error
}

type ForumPostReviewRepository interface {
	Create(ctx context.Context, review *ForumPostReview) error
}

// ForumCommentRepository defines the persistence interface for ForumComment.
type ForumCommentRepository interface {
	Create(ctx context.Context, comment *ForumComment) error
	FindByID(ctx context.Context, id uuid.UUID) (*ForumComment, error)
	Update(ctx context.Context, comment *ForumComment) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	// ListByPostID returns non-removed comments for a post with pagination.
	ListByPostID(ctx context.Context, postID uuid.UUID, page, limit int) ([]*ForumComment, int, error)
	// ListWithFlagCountGTE returns comments with flag_count >= threshold.
	ListWithFlagCountGTE(ctx context.Context, threshold int, page, limit int) ([]*ForumComment, int, error)
	// IncrementFlagCount atomically increments the flag_count for a comment.
	IncrementFlagCount(ctx context.Context, commentID uuid.UUID) error
}

// PostUpvoteRepository defines the persistence interface for PostUpvote.
type PostUpvoteRepository interface {
	// Exists returns true if the user has already upvoted the post.
	Exists(ctx context.Context, postID, userID uuid.UUID) (bool, error)
	// Create adds an upvote record.
	Create(ctx context.Context, upvote *PostUpvote) error
	// Delete removes an upvote record.
	Delete(ctx context.Context, postID, userID uuid.UUID) error
}

// ContentFlagRepository defines the persistence interface for ContentFlag.
type ContentFlagRepository interface {
	Create(ctx context.Context, flag *ContentFlag) error
	FindByID(ctx context.Context, id uuid.UUID) (*ContentFlag, error)
	Update(ctx context.Context, flag *ContentFlag) error
	// ListPending returns pending flags with pagination.
	ListPending(ctx context.Context, page, limit int) ([]*ContentFlag, int, error)
}
