package forum

import (
	"context"
	"strings"
	"testing"
	"time"

	domainforum "lms-backend/internal/domain/forum"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// ─── Mock implementations ─────────────────────────────────────────────────────

type mockPostRepo struct {
	posts map[uuid.UUID]*domainforum.ForumPost
}

func newMockPostRepo() *mockPostRepo {
	return &mockPostRepo{posts: make(map[uuid.UUID]*domainforum.ForumPost)}
}

func (m *mockPostRepo) Create(ctx context.Context, post *domainforum.ForumPost) error {
	m.posts[post.ID] = post
	return nil
}

func (m *mockPostRepo) FindByID(ctx context.Context, id uuid.UUID) (*domainforum.ForumPost, error) {
	p, ok := m.posts[id]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (m *mockPostRepo) Update(ctx context.Context, post *domainforum.ForumPost) error {
	m.posts[post.ID] = post
	return nil
}

func (m *mockPostRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	if p, ok := m.posts[id]; ok {
		now := time.Now().UTC()
		p.DeletedAt = &now
	}
	return nil
}

func (m *mockPostRepo) List(ctx context.Context, filter domainforum.PostFilter, page, limit int) ([]*domainforum.ForumPost, int, error) {
	var result []*domainforum.ForumPost
	for _, p := range m.posts {
		if p.Status != domainforum.PostStatusActive || p.DeletedAt != nil {
			continue
		}
		result = append(result, p)
	}
	return result, len(result), nil
}

func (m *mockPostRepo) ListWithFlagCountGTE(ctx context.Context, threshold int, page, limit int) ([]*domainforum.ForumPost, int, error) {
	var result []*domainforum.ForumPost
	for _, p := range m.posts {
		if p.FlagCount >= threshold && p.Status == domainforum.PostStatusActive && p.DeletedAt == nil {
			result = append(result, p)
		}
	}
	return result, len(result), nil
}

func (m *mockPostRepo) IncrementFlagCount(ctx context.Context, postID uuid.UUID) error {
	if p, ok := m.posts[postID]; ok {
		p.FlagCount++
	}
	return nil
}

type mockCommentRepo struct {
	comments map[uuid.UUID]*domainforum.ForumComment
}

func newMockCommentRepo() *mockCommentRepo {
	return &mockCommentRepo{comments: make(map[uuid.UUID]*domainforum.ForumComment)}
}

func (m *mockCommentRepo) Create(ctx context.Context, c *domainforum.ForumComment) error {
	m.comments[c.ID] = c
	return nil
}

func (m *mockCommentRepo) FindByID(ctx context.Context, id uuid.UUID) (*domainforum.ForumComment, error) {
	c, ok := m.comments[id]
	if !ok {
		return nil, nil
	}
	return c, nil
}

func (m *mockCommentRepo) Update(ctx context.Context, c *domainforum.ForumComment) error {
	m.comments[c.ID] = c
	return nil
}

func (m *mockCommentRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	if c, ok := m.comments[id]; ok {
		now := time.Now().UTC()
		c.DeletedAt = &now
	}
	return nil
}

func (m *mockCommentRepo) ListByPostID(ctx context.Context, postID uuid.UUID, page, limit int) ([]*domainforum.ForumComment, int, error) {
	var result []*domainforum.ForumComment
	for _, c := range m.comments {
		if c.PostID == postID && c.Status == domainforum.CommentStatusActive && c.DeletedAt == nil {
			result = append(result, c)
		}
	}
	return result, len(result), nil
}

func (m *mockCommentRepo) ListWithFlagCountGTE(ctx context.Context, threshold int, page, limit int) ([]*domainforum.ForumComment, int, error) {
	var result []*domainforum.ForumComment
	for _, c := range m.comments {
		if c.FlagCount >= threshold && c.Status == domainforum.CommentStatusActive && c.DeletedAt == nil {
			result = append(result, c)
		}
	}
	return result, len(result), nil
}

func (m *mockCommentRepo) IncrementFlagCount(ctx context.Context, commentID uuid.UUID) error {
	if c, ok := m.comments[commentID]; ok {
		c.FlagCount++
	}
	return nil
}

type mockUpvoteRepo struct {
	upvotes map[string]bool // "postID:userID"
}

func newMockUpvoteRepo() *mockUpvoteRepo {
	return &mockUpvoteRepo{upvotes: make(map[string]bool)}
}

func (m *mockUpvoteRepo) Exists(ctx context.Context, postID, userID uuid.UUID) (bool, error) {
	return m.upvotes[postID.String()+":"+userID.String()], nil
}

func (m *mockUpvoteRepo) Create(ctx context.Context, upvote *domainforum.PostUpvote) error {
	m.upvotes[upvote.PostID.String()+":"+upvote.UserID.String()] = true
	return nil
}

func (m *mockUpvoteRepo) Delete(ctx context.Context, postID, userID uuid.UUID) error {
	delete(m.upvotes, postID.String()+":"+userID.String())
	return nil
}

type mockFlagRepo struct {
	flags map[uuid.UUID]*domainforum.ContentFlag
}

func newMockFlagRepo() *mockFlagRepo {
	return &mockFlagRepo{flags: make(map[uuid.UUID]*domainforum.ContentFlag)}
}

func (m *mockFlagRepo) Create(ctx context.Context, flag *domainforum.ContentFlag) error {
	m.flags[flag.ID] = flag
	return nil
}

func (m *mockFlagRepo) FindByID(ctx context.Context, id uuid.UUID) (*domainforum.ContentFlag, error) {
	f, ok := m.flags[id]
	if !ok {
		return nil, nil
	}
	return f, nil
}

func (m *mockFlagRepo) Update(ctx context.Context, flag *domainforum.ContentFlag) error {
	m.flags[flag.ID] = flag
	return nil
}

func (m *mockFlagRepo) ListPending(ctx context.Context, page, limit int) ([]*domainforum.ContentFlag, int, error) {
	var result []*domainforum.ContentFlag
	for _, f := range m.flags {
		if f.Status == domainforum.FlagStatusPending {
			result = append(result, f)
		}
	}
	return result, len(result), nil
}

// newTestService creates a forum service with all mock dependencies.
func newTestService() (Service, *mockPostRepo, *mockCommentRepo, *mockUpvoteRepo, *mockFlagRepo) {
	postRepo := newMockPostRepo()
	commentRepo := newMockCommentRepo()
	upvoteRepo := newMockUpvoteRepo()
	flagRepo := newMockFlagRepo()
	svc := NewService(postRepo, commentRepo, upvoteRepo, flagRepo, nil, nil, nil, nil)
	return svc, postRepo, commentRepo, upvoteRepo, flagRepo
}

// seedPost creates a post directly in the mock repo.
func seedPost(repo *mockPostRepo) *domainforum.ForumPost {
	now := time.Now().UTC()
	post := &domainforum.ForumPost{
		ID:           uuid.New(),
		AuthorID:     uuid.New(),
		Title:        "Test Post",
		BodyMarkdown: "Hello **world**",
		BodyHTML:     "Hello <strong>world</strong>",
		Upvotes:      0,
		FlagCount:    0,
		Status:       domainforum.PostStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	repo.posts[post.ID] = post
	return post
}

// ─── Property 55: Posts with 3+ flags appear in moderation queue ─────────────

// Property55_FlaggedPostsAppearInModerationQueue verifies that after flagging a post
// flagThreshold (3) times, it appears in the moderation queue.
// Validates: Requirements 21.6
func TestProperty55_FlaggedPostsAppearInModerationQueue(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		svc, postRepo, _, _, _ := newTestService()
		ctx := context.Background()

		post := seedPost(postRepo)

		// Flag the post exactly flagThreshold times by different reporters
		for i := 0; i < flagThreshold; i++ {
			_, err := svc.FlagContent(ctx, FlagContentCommand{
				ReporterID: uuid.New(),
				TargetType: domainforum.FlagTargetPost,
				TargetID:   post.ID,
				Reason:     domainforum.FlagReasonSpam,
			})
			if err != nil {
				t.Fatalf("FlagContent failed: %v", err)
			}
		}

		// The post's flag_count should now be >= flagThreshold
		updated, _ := postRepo.FindByID(ctx, post.ID)
		if updated.FlagCount < flagThreshold {
			t.Fatalf("expected flag_count >= %d, got %d", flagThreshold, updated.FlagCount)
		}

		// The moderation queue should contain this post's flags
		queue, err := svc.GetModerationQueue(ctx, GetModerationQueueCommand{Page: 1, Limit: 100})
		if err != nil {
			t.Fatalf("GetModerationQueue failed: %v", err)
		}

		found := false
		for _, item := range queue.Data {
			if item.TargetID == post.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("post with %d flags not found in moderation queue", flagThreshold)
		}
	})
}

// ─── Property 56: Upvote is a toggle ─────────────────────────────────────────

// Property56_UpvoteIsToggle verifies that upvoting twice returns to the original state.
// Validates: Requirements 21.4
func TestProperty56_UpvoteIsToggle(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		svc, postRepo, _, _, _ := newTestService()
		ctx := context.Background()

		post := seedPost(postRepo)
		userID := uuid.New()
		initialUpvotes := post.Upvotes

		// First upvote — should add
		resp1, err := svc.ToggleUpvote(ctx, ToggleUpvoteCommand{PostID: post.ID, UserID: userID})
		if err != nil {
			t.Fatalf("first ToggleUpvote failed: %v", err)
		}
		if !resp1.UserUpvoted {
			t.Fatal("expected user_upvoted=true after first upvote")
		}
		if resp1.Upvotes != initialUpvotes+1 {
			t.Fatalf("expected upvotes=%d after first upvote, got %d", initialUpvotes+1, resp1.Upvotes)
		}

		// Second upvote — should remove (toggle back)
		resp2, err := svc.ToggleUpvote(ctx, ToggleUpvoteCommand{PostID: post.ID, UserID: userID})
		if err != nil {
			t.Fatalf("second ToggleUpvote failed: %v", err)
		}
		if resp2.UserUpvoted {
			t.Fatal("expected user_upvoted=false after second upvote (toggle off)")
		}
		if resp2.Upvotes != initialUpvotes {
			t.Fatalf("expected upvotes=%d after toggle off, got %d", initialUpvotes, resp2.Upvotes)
		}
	})
}

// ─── Property 6: Markdown content is XSS-sanitised ───────────────────────────

// Property6_MarkdownContentIsXSSSanitised verifies that XSS payloads in markdown
// are stripped from body_html before storage and return.
// Validates: Requirements 1.13, 21.3
func TestProperty6_MarkdownContentIsXSSSanitised(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		svc, _, _, _, _ := newTestService()
		ctx := context.Background()

		// Generate a random XSS payload embedded in markdown
		xssPayloads := rapid.SampledFrom([]string{
			`<script>alert('xss')</script>`,
			`<img src=x onerror=alert(1)>`,
			`<a href="javascript:alert(1)">click</a>`,
			`<svg onload=alert(1)>`,
			`<iframe src="javascript:alert(1)"></iframe>`,
			`<body onload=alert(1)>`,
		}).Draw(t, "xss_payload")

		authorID := uuid.New()
		title := "Test Post " + rapid.StringMatching(`[a-zA-Z]{3,10}`).Draw(t, "title_suffix")
		bodyMarkdown := "Hello " + xssPayloads + " world"

		result, err := svc.CreatePost(ctx, CreatePostCommand{
			AuthorID:     authorID,
			Title:        title,
			BodyMarkdown: bodyMarkdown,
		})
		if err != nil {
			t.Fatalf("CreatePost failed: %v", err)
		}

		// body_html must not contain script tags or event handlers
		html := result.BodyHTML
		dangerousPatterns := []string{
			"<script", "</script>",
			"javascript:",
			"onerror=", "onload=", "onclick=", "onmouseover=",
			"<iframe", "<svg onload",
		}
		for _, pattern := range dangerousPatterns {
			if strings.Contains(strings.ToLower(html), strings.ToLower(pattern)) {
				t.Fatalf("XSS payload not sanitised: found %q in body_html: %q", pattern, html)
			}
		}

		// body_markdown is stored as-is (raw input preserved)
		if result.BodyMarkdown != bodyMarkdown {
			t.Fatalf("body_markdown should be stored as-is, got %q", result.BodyMarkdown)
		}
	})
}

// ─── Additional: Comment sanitisation ────────────────────────────────────────

// TestProperty6_CommentMarkdownIsXSSSanitised verifies the same sanitisation for comments.
// Validates: Requirements 1.13, 21.3
func TestProperty6_CommentMarkdownIsXSSSanitised(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		svc, postRepo, _, _, _ := newTestService()
		ctx := context.Background()

		post := seedPost(postRepo)

		xssPayload := `<script>alert('xss')</script>`
		result, err := svc.CreateComment(ctx, CreateCommentCommand{
			PostID:       post.ID,
			AuthorID:     uuid.New(),
			BodyMarkdown: "Comment with " + xssPayload,
		})
		if err != nil {
			t.Fatalf("CreateComment failed: %v", err)
		}

		if strings.Contains(strings.ToLower(result.BodyHTML), "<script") {
			t.Fatalf("script tag not sanitised in comment body_html: %q", result.BodyHTML)
		}
	})
}
