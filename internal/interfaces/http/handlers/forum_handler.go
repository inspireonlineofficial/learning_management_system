package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	appforum "lms-backend/internal/application/forum"
	domainforum "lms-backend/internal/domain/forum"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// ForumHandler handles HTTP requests for the forum bounded context.
type ForumHandler struct {
	service appforum.Service
}

// NewForumHandler creates a new ForumHandler.
func NewForumHandler(service appforum.Service) *ForumHandler {
	return &ForumHandler{service: service}
}

// getActorNameFromContext extracts the actor name from context (best-effort).
func getActorNameFromContext(r *http.Request) string {
	if name, ok := r.Context().Value("user_name").(string); ok && name != "" {
		return name
	}
	return "unknown"
}

// ─── Public endpoints ─────────────────────────────────────────────────────────

// ListPosts handles GET /v1/forum/posts (public)
// Requirements: 21.1
//
// @Summary      List forum posts
// @Description  Returns a paginated list of forum posts, optionally filtered by course
// @Tags         forum
// @Produce      json
// @Param        page       query  int     false  "Page number"    default(1)
// @Param        limit      query  int     false  "Items per page" default(20)
// @Param        sort       query  string  false  "Sort order (newest, top)"
// @Param        course_id  query  string  false  "Filter by course UUID"
// @Success      200  {object}  forum.PostListResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Router       /v1/forum/posts [get]
func (h *ForumHandler) ListPosts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, limit := parseForumPaginationParams(q)

	sortOrder := domainforum.PostSortOrder(q.Get("sort"))
	if sortOrder != domainforum.PostSortTop {
		sortOrder = domainforum.PostSortNewest
	}

	var courseID *uuid.UUID
	if cid := q.Get("course_id"); cid != "" {
		parsed, err := uuid.Parse(cid)
		if err != nil {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_COURSE_ID", "invalid course_id"))
			return
		}
		courseID = &parsed
	}

	result, err := h.service.ListPosts(r.Context(), appforum.ListPostsCommand{
		CourseID:  courseID,
		SortOrder: sortOrder,
		Page:      page,
		Limit:     limit,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// ─── Authenticated endpoints ──────────────────────────────────────────────────

// CreatePost handles POST /v1/forum/posts (authenticated)
// Requirements: 21.2, 21.3
//
// @Summary      Create a forum post
// @Description  Creates a new forum post for the authenticated user
// @Tags         forum
// @Accept       json
// @Produce      json
// @Param        body  body  object{course_id=string,title=string,body_markdown=string}  true  "Post payload"
// @Success      201  {object}  forum.PostResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/forum/posts [post]
func (h *ForumHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	var req struct {
		CourseID     *string `json:"course_id"`
		Title        string  `json:"title"`
		BodyMarkdown string  `json:"body_markdown"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_BODY", "invalid request body"))
		return
	}

	cmd := appforum.CreatePostCommand{
		AuthorID:     userID,
		Title:        req.Title,
		BodyMarkdown: req.BodyMarkdown,
	}
	if req.CourseID != nil {
		parsed, err := uuid.Parse(*req.CourseID)
		if err != nil {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_COURSE_ID", "invalid course_id"))
			return
		}
		cmd.CourseID = &parsed
	}

	result, err := h.service.CreatePost(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusCreated, result)
}

// UpdatePost handles PATCH /v1/forum/posts/:postId (authenticated)
//
// @Summary      Update a forum post
// @Description  Updates the title or body of an existing forum post owned by the authenticated user
// @Tags         forum
// @Accept       json
// @Produce      json
// @Param        postId  path  string  true  "Post UUID"
// @Param        body    body  object{title=string,body_markdown=string}  true  "Update payload"
// @Success      200  {object}  forum.PostResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/forum/posts/{postId} [patch]
func (h *ForumHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	postID, err := uuid.Parse(r.PathValue("postId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid post ID"))
		return
	}

	var req struct {
		Title        *string `json:"title"`
		BodyMarkdown *string `json:"body_markdown"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_BODY", "invalid request body"))
		return
	}

	result, err := h.service.UpdatePost(r.Context(), appforum.UpdatePostCommand{
		PostID:       postID,
		AuthorID:     userID,
		Title:        req.Title,
		BodyMarkdown: req.BodyMarkdown,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// DeletePost handles DELETE /v1/forum/posts/:postId (authenticated)
//
// @Summary      Delete a forum post
// @Description  Soft-deletes a forum post owned by the authenticated user
// @Tags         forum
// @Produce      json
// @Param        postId  path  string  true  "Post UUID"
// @Success      204  "No Content"
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/forum/posts/{postId} [delete]
func (h *ForumHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	postID, err := uuid.Parse(r.PathValue("postId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid post ID"))
		return
	}

	if err := h.service.DeletePost(r.Context(), appforum.DeletePostCommand{
		PostID:  postID,
		ActorID: userID,
		IsAdmin: false,
	}); err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListComments handles GET /v1/forum/posts/:postId/comments (public)
//
// @Summary      List comments on a post
// @Description  Returns a paginated list of comments for the given forum post
// @Tags         forum
// @Produce      json
// @Param        postId  path   string  true   "Post UUID"
// @Param        page    query  int     false  "Page number"    default(1)
// @Param        limit   query  int     false  "Items per page" default(20)
// @Success      200  {object}  forum.CommentListResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /v1/forum/posts/{postId}/comments [get]
func (h *ForumHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	postID, err := uuid.Parse(r.PathValue("postId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid post ID"))
		return
	}

	q := r.URL.Query()
	page, limit := parseForumPaginationParams(q)

	result, err := h.service.ListComments(r.Context(), appforum.ListCommentsCommand{
		PostID: postID,
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// CreateComment handles POST /v1/forum/posts/:postId/comments (authenticated)
// Requirements: 21.2, 21.3
//
// @Summary      Create a comment
// @Description  Adds a new comment to the specified forum post
// @Tags         forum
// @Accept       json
// @Produce      json
// @Param        postId  path  string                          true  "Post UUID"
// @Param        body    body  object{body_markdown=string}    true  "Comment payload"
// @Success      201  {object}  forum.CommentResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/forum/posts/{postId}/comments [post]
func (h *ForumHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	postID, err := uuid.Parse(r.PathValue("postId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid post ID"))
		return
	}

	var req struct {
		BodyMarkdown string `json:"body_markdown"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_BODY", "invalid request body"))
		return
	}

	result, err := h.service.CreateComment(r.Context(), appforum.CreateCommentCommand{
		PostID:       postID,
		AuthorID:     userID,
		BodyMarkdown: req.BodyMarkdown,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusCreated, result)
}

// UpdateComment handles PATCH /v1/forum/posts/:postId/comments/:commentId (authenticated)
//
// @Summary      Update a comment
// @Description  Updates the body of an existing comment owned by the authenticated user
// @Tags         forum
// @Accept       json
// @Produce      json
// @Param        postId     path  string                        true  "Post UUID"
// @Param        commentId  path  string                        true  "Comment UUID"
// @Param        body       body  object{body_markdown=string}  true  "Update payload"
// @Success      200  {object}  forum.CommentResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/forum/posts/{postId}/comments/{commentId} [patch]
func (h *ForumHandler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	commentID, err := uuid.Parse(r.PathValue("commentId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid comment ID"))
		return
	}

	var req struct {
		BodyMarkdown string `json:"body_markdown"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_BODY", "invalid request body"))
		return
	}

	result, err := h.service.UpdateComment(r.Context(), appforum.UpdateCommentCommand{
		CommentID:    commentID,
		AuthorID:     userID,
		BodyMarkdown: req.BodyMarkdown,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// DeleteComment handles DELETE /v1/forum/posts/:postId/comments/:commentId (authenticated)
//
// @Summary      Delete a comment
// @Description  Soft-deletes a comment owned by the authenticated user
// @Tags         forum
// @Produce      json
// @Param        postId     path  string  true  "Post UUID"
// @Param        commentId  path  string  true  "Comment UUID"
// @Success      204  "No Content"
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/forum/posts/{postId}/comments/{commentId} [delete]
func (h *ForumHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	commentID, err := uuid.Parse(r.PathValue("commentId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid comment ID"))
		return
	}

	if err := h.service.DeleteComment(r.Context(), appforum.DeleteCommentCommand{
		CommentID: commentID,
		ActorID:   userID,
		IsAdmin:   false,
	}); err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ToggleUpvote handles POST /v1/forum/posts/:postId/upvote (authenticated)
// Requirements: 21.4
//
// @Summary      Toggle upvote on a post
// @Description  Adds or removes the authenticated user's upvote on the specified post
// @Tags         forum
// @Produce      json
// @Param        postId  path  string  true  "Post UUID"
// @Success      200  {object}  forum.UpvoteResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/forum/posts/{postId}/upvote [post]
func (h *ForumHandler) ToggleUpvote(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	postID, err := uuid.Parse(r.PathValue("postId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid post ID"))
		return
	}

	result, err := h.service.ToggleUpvote(r.Context(), appforum.ToggleUpvoteCommand{
		PostID: postID,
		UserID: userID,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// FlagPost handles POST /v1/forum/posts/:postId/flag (authenticated)
// Requirements: 21.5, 21.6
//
// @Summary      Flag a post
// @Description  Submits a content flag on the specified forum post
// @Tags         forum
// @Accept       json
// @Produce      json
// @Param        postId  path  string                              true  "Post UUID"
// @Param        body    body  object{reason=string,note=string}   true  "Flag payload"
// @Success      201  {object}  forum.FlagResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/forum/posts/{postId}/flag [post]
func (h *ForumHandler) FlagPost(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	postID, err := uuid.Parse(r.PathValue("postId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid post ID"))
		return
	}

	var req struct {
		Reason string  `json:"reason"`
		Note   *string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_BODY", "invalid request body"))
		return
	}

	result, err := h.service.FlagContent(r.Context(), appforum.FlagContentCommand{
		ReporterID: userID,
		TargetType: domainforum.FlagTargetPost,
		TargetID:   postID,
		Reason:     domainforum.FlagReason(req.Reason),
		Note:       req.Note,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusCreated, result)
}

// ─── Admin endpoints ──────────────────────────────────────────────────────────

// GetModerationQueue handles GET /v1/admin/moderation (admin)
// Requirements: 21.7
//
// @Summary      Get moderation queue
// @Description  Returns a paginated list of flagged content awaiting moderation
// @Tags         forum
// @Produce      json
// @Param        page   query  int  false  "Page number"    default(1)
// @Param        limit  query  int  false  "Items per page" default(20)
// @Success      200  {object}  forum.ModerationQueueResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/moderation [get]
func (h *ForumHandler) GetModerationQueue(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, limit := parseForumPaginationParams(q)

	result, err := h.service.GetModerationQueue(r.Context(), appforum.GetModerationQueueCommand{
		Page:  page,
		Limit: limit,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// ModerateContent handles POST /v1/admin/moderation/:flagId/action (admin)
// Requirements: 21.7, 21.8
//
// @Summary      Perform moderation action
// @Description  Executes a moderation action (e.g. remove content or ban user) on a flagged item
// @Tags         forum
// @Accept       json
// @Produce      json
// @Param        flagId  path  string                              true  "Flag UUID"
// @Param        body    body  object{action=string,reason=string} true  "Moderation action payload"
// @Success      200  {object}  forum.ModerationActionResponse
// @Failure      400  {object}  ValidationErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/admin/moderation/{flagId}/action [post]
func (h *ForumHandler) ModerateContent(w http.ResponseWriter, r *http.Request) {
	actorID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}

	flagID, err := uuid.Parse(r.PathValue("flagId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid flag ID"))
		return
	}

	var req struct {
		Action string `json:"action"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_BODY", "invalid request body"))
		return
	}

	result, err := h.service.ModerateContent(r.Context(), appforum.ModerateContentCommand{
		ActorID:   actorID,
		ActorName: getActorNameFromContext(r),
		FlagID:    flagID,
		Action:    req.Action,
		Reason:    req.Reason,
		IPAddress: r.RemoteAddr,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

func (h *ForumHandler) ListPostsForReview(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, limit := parseForumPaginationParams(q)
	status := domainforum.PostStatus(q.Get("status"))
	if status == "" {
		status = domainforum.PostStatusPending
	}
	result, err := h.service.ListPostsForReview(r.Context(), appforum.ListPostsForReviewCommand{
		Status: status,
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

func (h *ForumHandler) ApprovePost(w http.ResponseWriter, r *http.Request) {
	actorID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}
	postID, err := uuid.Parse(r.PathValue("postId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid post ID"))
		return
	}
	result, err := h.service.ApprovePost(r.Context(), appforum.ReviewPostCommand{
		ActorID:   actorID,
		PostID:    postID,
		IPAddress: requestIP(r),
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

func (h *ForumHandler) RejectPost(w http.ResponseWriter, r *http.Request) {
	actorID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, apperrors.ErrUnauthorized)
		return
	}
	postID, err := uuid.Parse(r.PathValue("postId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid post ID"))
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_BODY", "invalid request body"))
		return
	}
	result, err := h.service.RejectPost(r.Context(), appforum.ReviewPostCommand{
		ActorID:   actorID,
		PostID:    postID,
		Reason:    req.Reason,
		IPAddress: requestIP(r),
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	writeJSONResponse(w, http.StatusOK, result)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// parseForumPaginationParams extracts page and limit from query params.
func parseForumPaginationParams(q interface{ Get(string) string }) (page, limit int) {
	page = 1
	limit = 20
	if p := q.Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	return
}
