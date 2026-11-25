package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	domainforum "lms-backend/internal/domain/forum"

	"github.com/google/uuid"
)

// ─── ForumPostRepository ──────────────────────────────────────────────────────

// ForumPostRepository implements domainforum.ForumPostRepository.
type ForumPostRepository struct {
	db *sql.DB
}

// NewForumPostRepository creates a new ForumPostRepository.
func NewForumPostRepository(db *sql.DB) *ForumPostRepository {
	return &ForumPostRepository{db: db}
}

func (r *ForumPostRepository) Create(ctx context.Context, post *domainforum.ForumPost) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO forum_posts (id, author_id, course_id, title, body_markdown, body_html, upvotes, flag_count, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		post.ID, post.AuthorID, post.CourseID, post.Title, post.BodyMarkdown, post.BodyHTML,
		post.Upvotes, post.FlagCount, post.Status, post.CreatedAt, post.UpdatedAt,
	)
	return err
}

func (r *ForumPostRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainforum.ForumPost, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, author_id, course_id, title, body_markdown, body_html, upvotes, flag_count, status, created_at, updated_at, deleted_at
		FROM forum_posts WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanForumPost(row)
}

func (r *ForumPostRepository) Update(ctx context.Context, post *domainforum.ForumPost) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE forum_posts SET title=$1, body_markdown=$2, body_html=$3, upvotes=$4, flag_count=$5, status=$6, updated_at=$7
		WHERE id=$8 AND deleted_at IS NULL`,
		post.Title, post.BodyMarkdown, post.BodyHTML, post.Upvotes, post.FlagCount, post.Status, post.UpdatedAt, post.ID,
	)
	return err
}

func (r *ForumPostRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `UPDATE forum_posts SET deleted_at=$1, updated_at=$1 WHERE id=$2 AND deleted_at IS NULL`, now, id)
	return err
}

func (r *ForumPostRepository) List(ctx context.Context, filter domainforum.PostFilter, page, limit int) ([]*domainforum.ForumPost, int, error) {
	offset := (page - 1) * limit

	orderClause := "created_at DESC"
	if filter.SortOrder == domainforum.PostSortTop {
		orderClause = "upvotes DESC, created_at DESC"
	}

	args := []interface{}{}
	where := "status = 'active' AND deleted_at IS NULL"
	argIdx := 1

	if filter.CourseID != nil {
		where += fmt.Sprintf(" AND course_id = $%d", argIdx)
		args = append(args, *filter.CourseID)
		argIdx++
	}

	// Count query
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM forum_posts WHERE %s", where)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data query
	args = append(args, limit, offset)
	dataQuery := fmt.Sprintf(`
		SELECT id, author_id, course_id, title, body_markdown, body_html, upvotes, flag_count, status, created_at, updated_at, deleted_at
		FROM forum_posts WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d`,
		where, orderClause, argIdx, argIdx+1,
	)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []*domainforum.ForumPost
	for rows.Next() {
		p, err := scanForumPostRow(rows)
		if err != nil {
			return nil, 0, err
		}
		posts = append(posts, p)
	}
	return posts, total, rows.Err()
}

func (r *ForumPostRepository) ListWithFlagCountGTE(ctx context.Context, threshold int, page, limit int) ([]*domainforum.ForumPost, int, error) {
	offset := (page - 1) * limit

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM forum_posts WHERE flag_count >= $1 AND status = 'active' AND deleted_at IS NULL`,
		threshold).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, author_id, course_id, title, body_markdown, body_html, upvotes, flag_count, status, created_at, updated_at, deleted_at
		FROM forum_posts WHERE flag_count >= $1 AND status = 'active' AND deleted_at IS NULL
		ORDER BY flag_count DESC, created_at DESC LIMIT $2 OFFSET $3`,
		threshold, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []*domainforum.ForumPost
	for rows.Next() {
		p, err := scanForumPostRow(rows)
		if err != nil {
			return nil, 0, err
		}
		posts = append(posts, p)
	}
	return posts, total, rows.Err()
}

func (r *ForumPostRepository) ListByStatus(ctx context.Context, status domainforum.PostStatus, page, limit int) ([]*domainforum.ForumPost, int, error) {
	offset := (page - 1) * limit
	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM forum_posts WHERE status = $1 AND deleted_at IS NULL`, status).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, author_id, course_id, title, body_markdown, body_html, upvotes, flag_count, status, created_at, updated_at, deleted_at
		FROM forum_posts WHERE status = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC LIMIT $2 OFFSET $3`, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var posts []*domainforum.ForumPost
	for rows.Next() {
		p, err := scanForumPostRow(rows)
		if err != nil {
			return nil, 0, err
		}
		posts = append(posts, p)
	}
	return posts, total, rows.Err()
}

func (r *ForumPostRepository) IncrementFlagCount(ctx context.Context, postID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE forum_posts SET flag_count = flag_count + 1, updated_at = $1 WHERE id = $2 AND deleted_at IS NULL`,
		time.Now().UTC(), postID,
	)
	return err
}

func scanForumPost(row *sql.Row) (*domainforum.ForumPost, error) {
	var p domainforum.ForumPost
	var courseID *uuid.UUID
	var deletedAt *time.Time
	err := row.Scan(&p.ID, &p.AuthorID, &courseID, &p.Title, &p.BodyMarkdown, &p.BodyHTML,
		&p.Upvotes, &p.FlagCount, &p.Status, &p.CreatedAt, &p.UpdatedAt, &deletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.CourseID = courseID
	p.DeletedAt = deletedAt
	return &p, nil
}

func scanForumPostRow(rows *sql.Rows) (*domainforum.ForumPost, error) {
	var p domainforum.ForumPost
	var courseID *uuid.UUID
	var deletedAt *time.Time
	err := rows.Scan(&p.ID, &p.AuthorID, &courseID, &p.Title, &p.BodyMarkdown, &p.BodyHTML,
		&p.Upvotes, &p.FlagCount, &p.Status, &p.CreatedAt, &p.UpdatedAt, &deletedAt)
	if err != nil {
		return nil, err
	}
	p.CourseID = courseID
	p.DeletedAt = deletedAt
	return &p, nil
}

// ─── ForumCommentRepository ───────────────────────────────────────────────────

// ForumCommentRepository implements domainforum.ForumCommentRepository.
type ForumCommentRepository struct {
	db *sql.DB
}

// NewForumCommentRepository creates a new ForumCommentRepository.
func NewForumCommentRepository(db *sql.DB) *ForumCommentRepository {
	return &ForumCommentRepository{db: db}
}

func (r *ForumCommentRepository) Create(ctx context.Context, comment *domainforum.ForumComment) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO forum_comments (id, post_id, author_id, body_markdown, body_html, flag_count, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		comment.ID, comment.PostID, comment.AuthorID, comment.BodyMarkdown, comment.BodyHTML,
		0, comment.Status, comment.CreatedAt, comment.UpdatedAt,
	)
	return err
}

func (r *ForumCommentRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainforum.ForumComment, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, post_id, author_id, body_markdown, body_html, flag_count, status, created_at, updated_at, deleted_at
		FROM forum_comments WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanForumComment(row)
}

func (r *ForumCommentRepository) Update(ctx context.Context, comment *domainforum.ForumComment) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE forum_comments SET body_markdown=$1, body_html=$2, status=$3, updated_at=$4
		WHERE id=$5 AND deleted_at IS NULL`,
		comment.BodyMarkdown, comment.BodyHTML, comment.Status, comment.UpdatedAt, comment.ID,
	)
	return err
}

func (r *ForumCommentRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `UPDATE forum_comments SET deleted_at=$1, updated_at=$1 WHERE id=$2 AND deleted_at IS NULL`, now, id)
	return err
}

func (r *ForumCommentRepository) ListByPostID(ctx context.Context, postID uuid.UUID, page, limit int) ([]*domainforum.ForumComment, int, error) {
	offset := (page - 1) * limit

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM forum_comments WHERE post_id=$1 AND status='active' AND deleted_at IS NULL`,
		postID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, post_id, author_id, body_markdown, body_html, flag_count, status, created_at, updated_at, deleted_at
		FROM forum_comments WHERE post_id=$1 AND status='active' AND deleted_at IS NULL
		ORDER BY created_at ASC LIMIT $2 OFFSET $3`,
		postID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var comments []*domainforum.ForumComment
	for rows.Next() {
		c, err := scanForumCommentRow(rows)
		if err != nil {
			return nil, 0, err
		}
		comments = append(comments, c)
	}
	return comments, total, rows.Err()
}

func (r *ForumCommentRepository) ListWithFlagCountGTE(ctx context.Context, threshold int, page, limit int) ([]*domainforum.ForumComment, int, error) {
	offset := (page - 1) * limit

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM forum_comments WHERE flag_count >= $1 AND status='active' AND deleted_at IS NULL`,
		threshold).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, post_id, author_id, body_markdown, body_html, flag_count, status, created_at, updated_at, deleted_at
		FROM forum_comments WHERE flag_count >= $1 AND status='active' AND deleted_at IS NULL
		ORDER BY flag_count DESC, created_at DESC LIMIT $2 OFFSET $3`,
		threshold, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var comments []*domainforum.ForumComment
	for rows.Next() {
		c, err := scanForumCommentRow(rows)
		if err != nil {
			return nil, 0, err
		}
		comments = append(comments, c)
	}
	return comments, total, rows.Err()
}

func (r *ForumCommentRepository) IncrementFlagCount(ctx context.Context, commentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE forum_comments SET flag_count = flag_count + 1, updated_at = $1 WHERE id = $2 AND deleted_at IS NULL`,
		time.Now().UTC(), commentID,
	)
	return err
}

func scanForumComment(row *sql.Row) (*domainforum.ForumComment, error) {
	var c domainforum.ForumComment
	var deletedAt *time.Time
	err := row.Scan(&c.ID, &c.PostID, &c.AuthorID, &c.BodyMarkdown, &c.BodyHTML, &c.FlagCount, &c.Status, &c.CreatedAt, &c.UpdatedAt, &deletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.DeletedAt = deletedAt
	return &c, nil
}

func scanForumCommentRow(rows *sql.Rows) (*domainforum.ForumComment, error) {
	var c domainforum.ForumComment
	var deletedAt *time.Time
	err := rows.Scan(&c.ID, &c.PostID, &c.AuthorID, &c.BodyMarkdown, &c.BodyHTML, &c.FlagCount, &c.Status, &c.CreatedAt, &c.UpdatedAt, &deletedAt)
	if err != nil {
		return nil, err
	}
	c.DeletedAt = deletedAt
	return &c, nil
}

// ─── PostUpvoteRepository ─────────────────────────────────────────────────────

// PostUpvoteRepository implements domainforum.PostUpvoteRepository.
type PostUpvoteRepository struct {
	db *sql.DB
}

// NewPostUpvoteRepository creates a new PostUpvoteRepository.
func NewPostUpvoteRepository(db *sql.DB) *PostUpvoteRepository {
	return &PostUpvoteRepository{db: db}
}

func (r *PostUpvoteRepository) Exists(ctx context.Context, postID, userID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM post_upvotes WHERE post_id=$1 AND user_id=$2`, postID, userID).Scan(&count)
	return count > 0, err
}

func (r *PostUpvoteRepository) Create(ctx context.Context, upvote *domainforum.PostUpvote) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO post_upvotes (post_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		upvote.PostID, upvote.UserID,
	)
	return err
}

func (r *PostUpvoteRepository) Delete(ctx context.Context, postID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM post_upvotes WHERE post_id=$1 AND user_id=$2`, postID, userID)
	return err
}

// ─── ContentFlagRepository ────────────────────────────────────────────────────

// ContentFlagRepository implements domainforum.ContentFlagRepository.
type ContentFlagRepository struct {
	db *sql.DB
}

// NewContentFlagRepository creates a new ContentFlagRepository.
func NewContentFlagRepository(db *sql.DB) *ContentFlagRepository {
	return &ContentFlagRepository{db: db}
}

func (r *ContentFlagRepository) Create(ctx context.Context, flag *domainforum.ContentFlag) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO content_flags (id, reporter_id, target_type, target_id, reason, note, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		flag.ID, flag.ReporterID, flag.TargetType, flag.TargetID, flag.Reason, flag.Note, flag.Status, flag.CreatedAt,
	)
	return err
}

func (r *ContentFlagRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainforum.ContentFlag, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, reporter_id, target_type, target_id, reason, note, status, created_at
		FROM content_flags WHERE id=$1`, id)
	return scanContentFlag(row)
}

func (r *ContentFlagRepository) Update(ctx context.Context, flag *domainforum.ContentFlag) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE content_flags SET status=$1 WHERE id=$2`,
		flag.Status, flag.ID,
	)
	return err
}

func (r *ContentFlagRepository) ListPending(ctx context.Context, page, limit int) ([]*domainforum.ContentFlag, int, error) {
	offset := (page - 1) * limit

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM content_flags WHERE status='pending'`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, reporter_id, target_type, target_id, reason, note, status, created_at
		FROM content_flags WHERE status='pending'
		ORDER BY created_at ASC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var flags []*domainforum.ContentFlag
	for rows.Next() {
		f, err := scanContentFlagRow(rows)
		if err != nil {
			return nil, 0, err
		}
		flags = append(flags, f)
	}
	return flags, total, rows.Err()
}

func scanContentFlag(row *sql.Row) (*domainforum.ContentFlag, error) {
	var f domainforum.ContentFlag
	err := row.Scan(&f.ID, &f.ReporterID, &f.TargetType, &f.TargetID, &f.Reason, &f.Note, &f.Status, &f.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &f, err
}

func scanContentFlagRow(rows *sql.Rows) (*domainforum.ContentFlag, error) {
	var f domainforum.ContentFlag
	err := rows.Scan(&f.ID, &f.ReporterID, &f.TargetType, &f.TargetID, &f.Reason, &f.Note, &f.Status, &f.CreatedAt)
	return &f, err
}

type ForumPostReviewRepository struct {
	db *sql.DB
}

func NewForumPostReviewRepository(db *sql.DB) *ForumPostReviewRepository {
	return &ForumPostReviewRepository{db: db}
}

func (r *ForumPostReviewRepository) Create(ctx context.Context, review *domainforum.ForumPostReview) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO forum_post_reviews (id, post_id, reviewer_id, action, reason, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		review.ID, review.PostID, review.ReviewerID, review.Action, review.Reason, review.CreatedAt)
	return err
}
