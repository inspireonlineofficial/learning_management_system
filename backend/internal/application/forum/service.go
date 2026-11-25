package forum

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	domainforum "lms-backend/internal/domain/forum"
	"lms-backend/internal/domain/notifications"
	tsclient "lms-backend/internal/infrastructure/typesense"
	"lms-backend/pkg/apperrors"
	"lms-backend/pkg/logger"
	"lms-backend/pkg/pagination"

	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"
)

// flagThreshold is the number of flags that auto-surfaces content to the moderation queue.
const flagThreshold = 3

// Service defines the interface for all forum use cases.
type Service interface {
	// Public / authenticated use cases (Requirements: 21.1–21.6)
	ListPosts(ctx context.Context, cmd ListPostsCommand) (*PostListResponse, error)
	GetPost(ctx context.Context, postID uuid.UUID) (*PostResponse, error)
	CreatePost(ctx context.Context, cmd CreatePostCommand) (*PostResponse, error)
	UpdatePost(ctx context.Context, cmd UpdatePostCommand) (*PostResponse, error)
	DeletePost(ctx context.Context, cmd DeletePostCommand) error

	ListComments(ctx context.Context, cmd ListCommentsCommand) (*CommentListResponse, error)
	CreateComment(ctx context.Context, cmd CreateCommentCommand) (*CommentResponse, error)
	UpdateComment(ctx context.Context, cmd UpdateCommentCommand) (*CommentResponse, error)
	DeleteComment(ctx context.Context, cmd DeleteCommentCommand) error

	ToggleUpvote(ctx context.Context, cmd ToggleUpvoteCommand) (*UpvoteResponse, error)
	FlagContent(ctx context.Context, cmd FlagContentCommand) (*FlagResponse, error)

	// Admin moderation use cases (Requirements: 21.7, 21.8)
	GetModerationQueue(ctx context.Context, cmd GetModerationQueueCommand) (*ModerationQueueResponse, error)
	ModerateContent(ctx context.Context, cmd ModerateContentCommand) (*ModerationActionResponse, error)
	ListPostsForReview(ctx context.Context, cmd ListPostsForReviewCommand) (*PostListResponse, error)
	ApprovePost(ctx context.Context, cmd ReviewPostCommand) (*PostResponse, error)
	RejectPost(ctx context.Context, cmd ReviewPostCommand) (*PostResponse, error)
}

// AuditLogger records privileged admin actions.
type AuditLogger interface {
	LogAction(ctx context.Context, actorID uuid.UUID, actorName, action, targetType string, targetID uuid.UUID, metadata map[string]interface{}, ipAddress string) error
}

// UserDeactivator deactivates a user account and invalidates their sessions.
type UserDeactivator interface {
	DeactivateUser(ctx context.Context, userID uuid.UUID) error
}

type service struct {
	postRepo    domainforum.ForumPostRepository
	commentRepo domainforum.ForumCommentRepository
	upvoteRepo  domainforum.PostUpvoteRepository
	flagRepo    domainforum.ContentFlagRepository
	reviewRepo  domainforum.ForumPostReviewRepository
	jobQueue    notifications.JobQueue
	audit       AuditLogger
	deactivator UserDeactivator
	sanitizer   *bluemonday.Policy
	indexer     tsclient.Indexer
}

// NewService creates a new forum service.
func NewService(
	postRepo domainforum.ForumPostRepository,
	commentRepo domainforum.ForumCommentRepository,
	upvoteRepo domainforum.PostUpvoteRepository,
	flagRepo domainforum.ContentFlagRepository,
	jobQueue notifications.JobQueue,
	audit AuditLogger,
	deactivator UserDeactivator,
	indexer tsclient.Indexer,
) Service {
	return NewServiceWithReviewRepo(postRepo, commentRepo, upvoteRepo, flagRepo, nil, jobQueue, audit, deactivator, indexer)
}

func NewServiceWithReviewRepo(
	postRepo domainforum.ForumPostRepository,
	commentRepo domainforum.ForumCommentRepository,
	upvoteRepo domainforum.PostUpvoteRepository,
	flagRepo domainforum.ContentFlagRepository,
	reviewRepo domainforum.ForumPostReviewRepository,
	jobQueue notifications.JobQueue,
	audit AuditLogger,
	deactivator UserDeactivator,
	indexer tsclient.Indexer,
) Service {
	// UGCPolicy allows safe HTML from markdown renderers (strips scripts, event handlers, etc.)
	sanitizer := bluemonday.UGCPolicy()
	return &service{
		postRepo:    postRepo,
		commentRepo: commentRepo,
		upvoteRepo:  upvoteRepo,
		flagRepo:    flagRepo,
		reviewRepo:  reviewRepo,
		jobQueue:    jobQueue,
		audit:       audit,
		deactivator: deactivator,
		sanitizer:   sanitizer,
		indexer:     indexer,
	}
}

// truncate returns the first n bytes of s, or s itself if shorter.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// ─── Post use cases ───────────────────────────────────────────────────────────

// ListPosts returns a paginated list of non-removed posts.
// Public endpoint — no auth required.
// Requirements: 21.1
func (s *service) ListPosts(ctx context.Context, cmd ListPostsCommand) (*PostListResponse, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.Limit < 1 || cmd.Limit > 100 {
		cmd.Limit = 20
	}
	if cmd.SortOrder == "" {
		cmd.SortOrder = domainforum.PostSortNewest
	}

	filter := domainforum.PostFilter{
		CourseID:  cmd.CourseID,
		SortOrder: cmd.SortOrder,
	}

	posts, total, err := s.postRepo.List(ctx, filter, cmd.Page, cmd.Limit)
	if err != nil {
		return nil, apperrors.NewInternalError("LIST_POSTS_FAILED", "failed to list posts")
	}

	data := make([]*PostResponse, 0, len(posts))
	for _, p := range posts {
		data = append(data, toPostResponse(p))
	}

	meta := pagination.NewMeta(total, cmd.Page, cmd.Limit)
	return &PostListResponse{
		Data: data,
		Meta: map[string]interface{}{
			"page":        meta.Page,
			"limit":       meta.Limit,
			"total":       meta.Total,
			"total_pages": meta.TotalPages,
		},
	}, nil
}

// GetPost returns a single post by ID.
func (s *service) GetPost(ctx context.Context, postID uuid.UUID) (*PostResponse, error) {
	post, err := s.postRepo.FindByID(ctx, postID)
	if err != nil || post == nil {
		return nil, apperrors.NewNotFoundError("POST_NOT_FOUND", "post not found")
	}
	if post.Status == domainforum.PostStatusRemoved {
		return nil, apperrors.NewNotFoundError("POST_NOT_FOUND", "post not found")
	}
	return toPostResponse(post), nil
}

// CreatePost creates a new forum post.
// Sanitises markdown through bluemonday before storing body_html.
// Requirements: 21.2, 21.3
func (s *service) CreatePost(ctx context.Context, cmd CreatePostCommand) (*PostResponse, error) {
	if len(cmd.Title) < 3 || len(cmd.Title) > 255 {
		return nil, apperrors.NewSimpleValidationError("INVALID_TITLE", "title must be between 3 and 255 characters")
	}
	if len(cmd.BodyMarkdown) == 0 {
		return nil, apperrors.NewSimpleValidationError("INVALID_BODY", "body cannot be empty")
	}

	// Sanitise markdown content — strip XSS vectors before storage
	safeHTML := s.sanitizer.Sanitize(cmd.BodyMarkdown)

	now := time.Now().UTC()
	post := &domainforum.ForumPost{
		ID:           uuid.New(),
		AuthorID:     cmd.AuthorID,
		CourseID:     cmd.CourseID,
		Title:        cmd.Title,
		BodyMarkdown: cmd.BodyMarkdown,
		BodyHTML:     safeHTML,
		Upvotes:      0,
		FlagCount:    0,
		Status:       domainforum.PostStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.postRepo.Create(ctx, post); err != nil {
		return nil, apperrors.NewInternalError("CREATE_POST_FAILED", "failed to create post")
	}

	if s.indexer != nil {
		if err := s.indexer.UpsertForumPost(ctx, tsclient.ForumPostDocument{
			ID:          post.ID.String(),
			Title:       post.Title,
			BodyExcerpt: truncate(post.BodyMarkdown, 200),
			Status:      string(post.Status),
			CreatedAt:   post.CreatedAt.Unix(),
		}); err != nil {
			log.Printf("typesense index error: %v", err)
		}
	}

	logger.Info(ctx, "Forum post created", "post_id", post.ID, "author_id", cmd.AuthorID)
	return toPostResponse(post), nil
}

// UpdatePost updates a forum post's title and/or body.
// Only the author may update their own post.
// Requirements: 21.2, 21.3
func (s *service) UpdatePost(ctx context.Context, cmd UpdatePostCommand) (*PostResponse, error) {
	post, err := s.postRepo.FindByID(ctx, cmd.PostID)
	if err != nil || post == nil {
		return nil, apperrors.NewNotFoundError("POST_NOT_FOUND", "post not found")
	}
	if post.AuthorID != cmd.AuthorID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "you can only edit your own posts")
	}
	if post.Status == domainforum.PostStatusRemoved {
		return nil, apperrors.NewNotFoundError("POST_NOT_FOUND", "post not found")
	}

	if cmd.Title != nil {
		post.Title = *cmd.Title
	}
	if cmd.BodyMarkdown != nil {
		post.BodyMarkdown = *cmd.BodyMarkdown
		post.BodyHTML = s.sanitizer.Sanitize(*cmd.BodyMarkdown)
	}
	post.UpdatedAt = time.Now().UTC()

	if err := s.postRepo.Update(ctx, post); err != nil {
		return nil, apperrors.NewInternalError("UPDATE_POST_FAILED", "failed to update post")
	}

	if s.indexer != nil {
		if err := s.indexer.UpsertForumPost(ctx, tsclient.ForumPostDocument{
			ID:          post.ID.String(),
			Title:       post.Title,
			BodyExcerpt: truncate(post.BodyMarkdown, 200),
			Status:      string(post.Status),
			CreatedAt:   post.CreatedAt.Unix(),
		}); err != nil {
			log.Printf("typesense index error: %v", err)
		}
	}

	return toPostResponse(post), nil
}

// DeletePost soft-deletes a forum post.
func (s *service) DeletePost(ctx context.Context, cmd DeletePostCommand) error {
	post, err := s.postRepo.FindByID(ctx, cmd.PostID)
	if err != nil || post == nil {
		return apperrors.NewNotFoundError("POST_NOT_FOUND", "post not found")
	}
	if !cmd.IsAdmin && post.AuthorID != cmd.ActorID {
		return apperrors.NewForbiddenError("FORBIDDEN", "you can only delete your own posts")
	}
	return s.postRepo.SoftDelete(ctx, cmd.PostID)
}

// ─── Comment use cases ────────────────────────────────────────────────────────

// ListComments returns paginated comments for a post.
func (s *service) ListComments(ctx context.Context, cmd ListCommentsCommand) (*CommentListResponse, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.Limit < 1 || cmd.Limit > 100 {
		cmd.Limit = 20
	}

	comments, total, err := s.commentRepo.ListByPostID(ctx, cmd.PostID, cmd.Page, cmd.Limit)
	if err != nil {
		return nil, apperrors.NewInternalError("LIST_COMMENTS_FAILED", "failed to list comments")
	}

	data := make([]*CommentResponse, 0, len(comments))
	for _, c := range comments {
		data = append(data, toCommentResponse(c))
	}

	meta := pagination.NewMeta(total, cmd.Page, cmd.Limit)
	return &CommentListResponse{
		Data: data,
		Meta: map[string]interface{}{
			"page":        meta.Page,
			"limit":       meta.Limit,
			"total":       meta.Total,
			"total_pages": meta.TotalPages,
		},
	}, nil
}

// CreateComment creates a comment on a post.
// Sanitises markdown through bluemonday before storing body_html.
// Requirements: 21.2, 21.3
func (s *service) CreateComment(ctx context.Context, cmd CreateCommentCommand) (*CommentResponse, error) {
	if len(cmd.BodyMarkdown) == 0 {
		return nil, apperrors.NewSimpleValidationError("INVALID_BODY", "comment body cannot be empty")
	}

	// Verify the post exists and is active
	post, err := s.postRepo.FindByID(ctx, cmd.PostID)
	if err != nil || post == nil || post.Status == domainforum.PostStatusRemoved {
		return nil, apperrors.NewNotFoundError("POST_NOT_FOUND", "post not found")
	}

	safeHTML := s.sanitizer.Sanitize(cmd.BodyMarkdown)

	now := time.Now().UTC()
	comment := &domainforum.ForumComment{
		ID:           uuid.New(),
		PostID:       cmd.PostID,
		AuthorID:     cmd.AuthorID,
		BodyMarkdown: cmd.BodyMarkdown,
		BodyHTML:     safeHTML,
		Status:       domainforum.CommentStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.commentRepo.Create(ctx, comment); err != nil {
		return nil, apperrors.NewInternalError("CREATE_COMMENT_FAILED", "failed to create comment")
	}

	return toCommentResponse(comment), nil
}

// UpdateComment updates a comment's body.
func (s *service) UpdateComment(ctx context.Context, cmd UpdateCommentCommand) (*CommentResponse, error) {
	comment, err := s.commentRepo.FindByID(ctx, cmd.CommentID)
	if err != nil || comment == nil {
		return nil, apperrors.NewNotFoundError("COMMENT_NOT_FOUND", "comment not found")
	}
	if comment.AuthorID != cmd.AuthorID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "you can only edit your own comments")
	}
	if comment.Status == domainforum.CommentStatusRemoved {
		return nil, apperrors.NewNotFoundError("COMMENT_NOT_FOUND", "comment not found")
	}

	comment.BodyMarkdown = cmd.BodyMarkdown
	comment.BodyHTML = s.sanitizer.Sanitize(cmd.BodyMarkdown)
	comment.UpdatedAt = time.Now().UTC()

	if err := s.commentRepo.Update(ctx, comment); err != nil {
		return nil, apperrors.NewInternalError("UPDATE_COMMENT_FAILED", "failed to update comment")
	}
	return toCommentResponse(comment), nil
}

// DeleteComment soft-deletes a comment.
func (s *service) DeleteComment(ctx context.Context, cmd DeleteCommentCommand) error {
	comment, err := s.commentRepo.FindByID(ctx, cmd.CommentID)
	if err != nil || comment == nil {
		return apperrors.NewNotFoundError("COMMENT_NOT_FOUND", "comment not found")
	}
	if !cmd.IsAdmin && comment.AuthorID != cmd.ActorID {
		return apperrors.NewForbiddenError("FORBIDDEN", "you can only delete your own comments")
	}
	return s.commentRepo.SoftDelete(ctx, cmd.CommentID)
}

// ─── Upvote use cases ─────────────────────────────────────────────────────────

// ToggleUpvote adds an upvote if not already upvoted, removes it if already upvoted.
// Returns the updated upvote count and whether the user has upvoted.
// Requirements: 21.4
func (s *service) ToggleUpvote(ctx context.Context, cmd ToggleUpvoteCommand) (*UpvoteResponse, error) {
	post, err := s.postRepo.FindByID(ctx, cmd.PostID)
	if err != nil || post == nil || post.Status == domainforum.PostStatusRemoved {
		return nil, apperrors.NewNotFoundError("POST_NOT_FOUND", "post not found")
	}

	exists, err := s.upvoteRepo.Exists(ctx, cmd.PostID, cmd.UserID)
	if err != nil {
		return nil, apperrors.NewInternalError("UPVOTE_CHECK_FAILED", "failed to check upvote status")
	}

	if exists {
		// Remove upvote
		if err := s.upvoteRepo.Delete(ctx, cmd.PostID, cmd.UserID); err != nil {
			return nil, apperrors.NewInternalError("UPVOTE_REMOVE_FAILED", "failed to remove upvote")
		}
		post.Upvotes--
		if post.Upvotes < 0 {
			post.Upvotes = 0
		}
	} else {
		// Add upvote
		upvote := &domainforum.PostUpvote{PostID: cmd.PostID, UserID: cmd.UserID}
		if err := s.upvoteRepo.Create(ctx, upvote); err != nil {
			return nil, apperrors.NewInternalError("UPVOTE_ADD_FAILED", "failed to add upvote")
		}
		post.Upvotes++
	}

	post.UpdatedAt = time.Now().UTC()
	if err := s.postRepo.Update(ctx, post); err != nil {
		return nil, apperrors.NewInternalError("UPVOTE_UPDATE_FAILED", "failed to update upvote count")
	}

	return &UpvoteResponse{
		PostID:      cmd.PostID,
		Upvotes:     post.Upvotes,
		UserUpvoted: !exists,
	}, nil
}

// ─── Flag use cases ───────────────────────────────────────────────────────────

// FlagContent creates a content flag.
// Validates reason enum; requires note when reason = other.
// Auto-surfaces to moderation queue when flag_count >= 3.
// Requirements: 21.5, 21.6
func (s *service) FlagContent(ctx context.Context, cmd FlagContentCommand) (*FlagResponse, error) {
	// Validate reason
	switch cmd.Reason {
	case domainforum.FlagReasonSpam, domainforum.FlagReasonOffensive,
		domainforum.FlagReasonMisinformation, domainforum.FlagReasonOther:
		// valid
	default:
		return nil, apperrors.NewSimpleValidationError("INVALID_REASON", "invalid flag reason")
	}

	// Require note when reason = other
	if cmd.Reason == domainforum.FlagReasonOther && (cmd.Note == nil || *cmd.Note == "") {
		return nil, apperrors.NewSimpleValidationError("NOTE_REQUIRED", "note is required when reason is 'other'")
	}

	// Validate target exists
	switch cmd.TargetType {
	case domainforum.FlagTargetPost:
		post, err := s.postRepo.FindByID(ctx, cmd.TargetID)
		if err != nil || post == nil || post.Status == domainforum.PostStatusRemoved {
			return nil, apperrors.NewNotFoundError("POST_NOT_FOUND", "post not found")
		}
	case domainforum.FlagTargetComment:
		comment, err := s.commentRepo.FindByID(ctx, cmd.TargetID)
		if err != nil || comment == nil || comment.Status == domainforum.CommentStatusRemoved {
			return nil, apperrors.NewNotFoundError("COMMENT_NOT_FOUND", "comment not found")
		}
	default:
		return nil, apperrors.NewSimpleValidationError("INVALID_TARGET_TYPE", "invalid target type")
	}

	now := time.Now().UTC()
	flag := &domainforum.ContentFlag{
		ID:         uuid.New(),
		ReporterID: cmd.ReporterID,
		TargetType: cmd.TargetType,
		TargetID:   cmd.TargetID,
		Reason:     cmd.Reason,
		Note:       cmd.Note,
		Status:     domainforum.FlagStatusPending,
		CreatedAt:  now,
	}

	if err := s.flagRepo.Create(ctx, flag); err != nil {
		return nil, apperrors.NewInternalError("FLAG_FAILED", "failed to create flag")
	}

	// Increment flag_count on the target and auto-surface to moderation queue at threshold
	switch cmd.TargetType {
	case domainforum.FlagTargetPost:
		if err := s.postRepo.IncrementFlagCount(ctx, cmd.TargetID); err != nil {
			logger.Error(ctx, "Failed to increment post flag count", "post_id", cmd.TargetID, "error", err)
		}
	case domainforum.FlagTargetComment:
		if err := s.commentRepo.IncrementFlagCount(ctx, cmd.TargetID); err != nil {
			logger.Error(ctx, "Failed to increment comment flag count", "comment_id", cmd.TargetID, "error", err)
		}
	}

	return &FlagResponse{
		ID:         flag.ID,
		TargetType: flag.TargetType,
		TargetID:   flag.TargetID,
		Reason:     flag.Reason,
		Status:     flag.Status,
		CreatedAt:  flag.CreatedAt,
	}, nil
}

// ─── Moderation use cases ─────────────────────────────────────────────────────

// GetModerationQueue returns posts/comments with flag_count >= 3.
// Requirements: 21.7
func (s *service) GetModerationQueue(ctx context.Context, cmd GetModerationQueueCommand) (*ModerationQueueResponse, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.Limit < 1 || cmd.Limit > 100 {
		cmd.Limit = 20
	}

	flags, total, err := s.flagRepo.ListPending(ctx, cmd.Page, cmd.Limit)
	if err != nil {
		return nil, apperrors.NewInternalError("MODERATION_QUEUE_FAILED", "failed to fetch moderation queue")
	}

	data := make([]*ModerationQueueItem, 0, len(flags))
	for _, f := range flags {
		contentPreview := ""
		switch f.TargetType {
		case domainforum.FlagTargetPost:
			if post, findErr := s.postRepo.FindByID(ctx, f.TargetID); findErr == nil && post != nil {
				contentPreview = strings.TrimSpace(post.Title + "\n" + post.BodyMarkdown)
			}
		case domainforum.FlagTargetComment:
			if comment, findErr := s.commentRepo.FindByID(ctx, f.TargetID); findErr == nil && comment != nil {
				contentPreview = strings.TrimSpace(comment.BodyMarkdown)
			}
		}

		data = append(data, &ModerationQueueItem{
			FlagID:         f.ID,
			TargetType:     f.TargetType,
			TargetID:       f.TargetID,
			ContentPreview: contentPreview,
			Reason:         f.Reason,
			Note:           f.Note,
			Status:         f.Status,
			CreatedAt:      f.CreatedAt,
		})
	}

	meta := pagination.NewMeta(total, cmd.Page, cmd.Limit)
	return &ModerationQueueResponse{
		Data: data,
		Meta: map[string]interface{}{
			"page":        meta.Page,
			"limit":       meta.Limit,
			"total":       meta.Total,
			"total_pages": meta.TotalPages,
		},
	}, nil
}

// ModerateContent performs a moderation action on flagged content.
// "remove" → soft-delete content, record audit log, notify author.
// "ban_user" → deactivate account, record audit log, require non-empty reason.
// Requirements: 21.7, 21.8
func (s *service) ModerateContent(ctx context.Context, cmd ModerateContentCommand) (*ModerationActionResponse, error) {
	flag, err := s.flagRepo.FindByID(ctx, cmd.FlagID)
	if err != nil || flag == nil {
		return nil, apperrors.NewNotFoundError("FLAG_NOT_FOUND", "flag not found")
	}

	switch cmd.Action {
	case "remove":
		if err := s.removeContent(ctx, flag); err != nil {
			return nil, err
		}
		// Record audit log
		if s.audit != nil {
			_ = s.audit.LogAction(ctx, cmd.ActorID, cmd.ActorName, "content_removed",
				string(flag.TargetType), flag.TargetID,
				map[string]interface{}{"flag_id": flag.ID, "reason": flag.Reason},
				cmd.IPAddress,
			)
		}
		// Notify author (best-effort)
		s.enqueueContentRemovedNotification(ctx, flag)

	case "ban_user":
		if cmd.Reason == "" {
			return nil, apperrors.NewSimpleValidationError("REASON_REQUIRED", "reason is required for ban_user action")
		}
		authorID, err := s.getContentAuthorID(ctx, flag)
		if err != nil {
			return nil, err
		}
		if s.deactivator != nil {
			if err := s.deactivator.DeactivateUser(ctx, authorID); err != nil {
				return nil, apperrors.NewInternalError("BAN_USER_FAILED", "failed to deactivate user")
			}
		}
		// Record audit log
		if s.audit != nil {
			_ = s.audit.LogAction(ctx, cmd.ActorID, cmd.ActorName, "user_banned",
				"user", authorID,
				map[string]interface{}{"flag_id": flag.ID, "reason": cmd.Reason},
				cmd.IPAddress,
			)
		}

	default:
		return nil, apperrors.NewSimpleValidationError("INVALID_ACTION", "action must be 'remove' or 'ban_user'")
	}

	// Mark flag as actioned
	flag.Status = domainforum.FlagStatusActioned
	if err := s.flagRepo.Update(ctx, flag); err != nil {
		logger.Error(ctx, "Failed to update flag status", "flag_id", flag.ID, "error", err)
	}

	return &ModerationActionResponse{
		FlagID:  flag.ID,
		Action:  cmd.Action,
		Success: true,
	}, nil
}

func (s *service) ListPostsForReview(ctx context.Context, cmd ListPostsForReviewCommand) (*PostListResponse, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.Limit < 1 || cmd.Limit > 100 {
		cmd.Limit = 20
	}
	if cmd.Status == "" {
		cmd.Status = domainforum.PostStatusPending
	}
	repo, ok := s.postRepo.(interface {
		ListByStatus(ctx context.Context, status domainforum.PostStatus, page, limit int) ([]*domainforum.ForumPost, int, error)
	})
	if !ok {
		return nil, apperrors.NewInternalError("REVIEW_REPO_UNSUPPORTED", "forum review listing is not supported")
	}
	posts, total, err := repo.ListByStatus(ctx, cmd.Status, cmd.Page, cmd.Limit)
	if err != nil {
		return nil, apperrors.NewInternalError("LIST_REVIEW_POSTS_FAILED", "failed to list posts for review")
	}
	data := make([]*PostResponse, 0, len(posts))
	for _, p := range posts {
		data = append(data, toPostResponse(p))
	}
	meta := pagination.NewMeta(total, cmd.Page, cmd.Limit)
	return &PostListResponse{
		Data: data,
		Meta: map[string]interface{}{
			"page": meta.Page, "limit": meta.Limit, "total": meta.Total, "total_pages": meta.TotalPages,
		},
	}, nil
}

func (s *service) ApprovePost(ctx context.Context, cmd ReviewPostCommand) (*PostResponse, error) {
	post, err := s.postRepo.FindByID(ctx, cmd.PostID)
	if err != nil || post == nil {
		return nil, apperrors.NewNotFoundError("POST_NOT_FOUND", "post not found")
	}
	if post.Status != domainforum.PostStatusPending {
		return nil, apperrors.NewSimpleValidationError("INVALID_STATUS", "only pending posts can be approved")
	}
	post.Status = domainforum.PostStatusActive
	post.UpdatedAt = time.Now().UTC()
	if err := s.postRepo.Update(ctx, post); err != nil {
		return nil, apperrors.NewInternalError("APPROVE_POST_FAILED", "failed to approve post")
	}
	s.recordPostReview(ctx, cmd.ActorID, post.ID, "approve", "", cmd.IPAddress)
	return toPostResponse(post), nil
}

func (s *service) RejectPost(ctx context.Context, cmd ReviewPostCommand) (*PostResponse, error) {
	if cmd.Reason == "" {
		return nil, apperrors.NewSimpleValidationError("REASON_REQUIRED", "rejection reason is required")
	}
	post, err := s.postRepo.FindByID(ctx, cmd.PostID)
	if err != nil || post == nil {
		return nil, apperrors.NewNotFoundError("POST_NOT_FOUND", "post not found")
	}
	if post.Status != domainforum.PostStatusPending {
		return nil, apperrors.NewSimpleValidationError("INVALID_STATUS", "only pending posts can be rejected")
	}
	post.Status = domainforum.PostStatusRejected
	post.UpdatedAt = time.Now().UTC()
	if err := s.postRepo.Update(ctx, post); err != nil {
		return nil, apperrors.NewInternalError("REJECT_POST_FAILED", "failed to reject post")
	}
	s.recordPostReview(ctx, cmd.ActorID, post.ID, "reject", cmd.Reason, cmd.IPAddress)
	s.enqueuePostRejectedNotification(ctx, post, cmd.Reason)
	return toPostResponse(post), nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// removeContent soft-deletes the flagged post or comment.
func (s *service) removeContent(ctx context.Context, flag *domainforum.ContentFlag) error {
	switch flag.TargetType {
	case domainforum.FlagTargetPost:
		post, err := s.postRepo.FindByID(ctx, flag.TargetID)
		if err != nil || post == nil {
			return apperrors.NewNotFoundError("POST_NOT_FOUND", "post not found")
		}
		post.Status = domainforum.PostStatusRemoved
		post.UpdatedAt = time.Now().UTC()
		return s.postRepo.Update(ctx, post)
	case domainforum.FlagTargetComment:
		comment, err := s.commentRepo.FindByID(ctx, flag.TargetID)
		if err != nil || comment == nil {
			return apperrors.NewNotFoundError("COMMENT_NOT_FOUND", "comment not found")
		}
		comment.Status = domainforum.CommentStatusRemoved
		comment.UpdatedAt = time.Now().UTC()
		return s.commentRepo.Update(ctx, comment)
	}
	return nil
}

// getContentAuthorID returns the author ID of the flagged content.
func (s *service) getContentAuthorID(ctx context.Context, flag *domainforum.ContentFlag) (uuid.UUID, error) {
	switch flag.TargetType {
	case domainforum.FlagTargetPost:
		post, err := s.postRepo.FindByID(ctx, flag.TargetID)
		if err != nil || post == nil {
			return uuid.Nil, apperrors.NewNotFoundError("POST_NOT_FOUND", "post not found")
		}
		return post.AuthorID, nil
	case domainforum.FlagTargetComment:
		comment, err := s.commentRepo.FindByID(ctx, flag.TargetID)
		if err != nil || comment == nil {
			return uuid.Nil, apperrors.NewNotFoundError("COMMENT_NOT_FOUND", "comment not found")
		}
		return comment.AuthorID, nil
	}
	return uuid.Nil, apperrors.NewSimpleValidationError("INVALID_TARGET_TYPE", "invalid target type")
}

// enqueueContentRemovedNotification enqueues a notification to the content author.
func (s *service) enqueueContentRemovedNotification(ctx context.Context, flag *domainforum.ContentFlag) {
	type payload struct {
		TargetType string    `json:"target_type"`
		TargetID   uuid.UUID `json:"target_id"`
	}
	data, err := json.Marshal(payload{
		TargetType: string(flag.TargetType),
		TargetID:   flag.TargetID,
	})
	if err != nil {
		return
	}
	job := notifications.Job{
		Type:    "content_removed",
		Payload: json.RawMessage(data),
	}
	if err := s.jobQueue.Enqueue(ctx, job); err != nil {
		logger.Error(ctx, "Failed to enqueue content_removed notification", "flag_id", flag.ID, "error", err)
	}
}

func (s *service) recordPostReview(ctx context.Context, actorID, postID uuid.UUID, action, reason, ip string) {
	if s.reviewRepo != nil {
		_ = s.reviewRepo.Create(ctx, &domainforum.ForumPostReview{
			ID: uuid.New(), PostID: postID, ReviewerID: actorID, Action: action, Reason: reason, CreatedAt: time.Now().UTC(),
		})
	}
	if s.audit != nil {
		_ = s.audit.LogAction(ctx, actorID, "", "forum_post_"+action, "forum_post", postID, map[string]interface{}{"reason": reason}, ip)
	}
}

func (s *service) enqueuePostRejectedNotification(ctx context.Context, post *domainforum.ForumPost, reason string) {
	payload, err := json.Marshal(map[string]interface{}{"post_id": post.ID, "author_id": post.AuthorID, "reason": reason})
	if err != nil || s.jobQueue == nil {
		return
	}
	if err := s.jobQueue.Enqueue(ctx, notifications.Job{Type: "forum_post_rejected", Payload: payload}); err != nil {
		logger.Error(ctx, "Failed to enqueue forum_post_rejected notification", "post_id", post.ID, "error", err)
	}
}
