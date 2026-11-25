package typesense

import (
	"context"
	"errors"
	"fmt"

	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
)

// Searcher is the read interface consumed by search.Service.
type Searcher interface {
	SearchCollection(ctx context.Context, collection string, params *api.SearchCollectionParams) (*api.SearchResult, error)
}

// Indexer is the write interface consumed by domain services that mutate content.
type Indexer interface {
	UpsertCourse(ctx context.Context, doc CourseDocument) error
	DeleteCourse(ctx context.Context, id string) error
	UpsertLesson(ctx context.Context, doc LessonDocument) error
	DeleteLesson(ctx context.Context, id string) error
	UpsertForumPost(ctx context.Context, doc ForumPostDocument) error
	DeleteForumPost(ctx context.Context, id string) error
	UpsertBook(ctx context.Context, doc BookDocument) error
	DeleteBook(ctx context.Context, id string) error
}

// Client wraps the typesense-go SDK and implements both Searcher and Indexer.
type Client struct {
	ts *typesense.Client
}

// NewClient constructs a Client. Returns a descriptive error if host or apiKey
// is empty, since the SDK constructor does not validate these itself.
func NewClient(host, port, apiKey string) (*Client, error) {
	if host == "" {
		return nil, errors.New("typesense: host must not be empty")
	}
	if apiKey == "" {
		return nil, errors.New("typesense: apiKey must not be empty")
	}
	ts := typesense.NewClient(
		typesense.WithServer(fmt.Sprintf("http://%s:%s", host, port)),
		typesense.WithAPIKey(apiKey),
	)
	return &Client{ts: ts}, nil
}

// SearchCollection executes a search against the named collection.
func (c *Client) SearchCollection(ctx context.Context, collection string, params *api.SearchCollectionParams) (*api.SearchResult, error) {
	return c.ts.Collection(collection).Documents().Search(ctx, params)
}

// UpsertCourse inserts or updates a course document in the "courses" collection.
func (c *Client) UpsertCourse(ctx context.Context, doc CourseDocument) error {
	_, err := c.ts.Collection("courses").Documents().Upsert(ctx, doc)
	return err
}

// DeleteCourse removes a course document from the "courses" collection.
func (c *Client) DeleteCourse(ctx context.Context, id string) error {
	_, err := c.ts.Collection("courses").Document(id).Delete(ctx)
	return err
}

// UpsertLesson inserts or updates a lesson document in the "lessons" collection.
func (c *Client) UpsertLesson(ctx context.Context, doc LessonDocument) error {
	_, err := c.ts.Collection("lessons").Documents().Upsert(ctx, doc)
	return err
}

// DeleteLesson removes a lesson document from the "lessons" collection.
func (c *Client) DeleteLesson(ctx context.Context, id string) error {
	_, err := c.ts.Collection("lessons").Document(id).Delete(ctx)
	return err
}

// UpsertForumPost inserts or updates a forum post document in the "forum_posts" collection.
func (c *Client) UpsertForumPost(ctx context.Context, doc ForumPostDocument) error {
	_, err := c.ts.Collection("forum_posts").Documents().Upsert(ctx, doc)
	return err
}

// DeleteForumPost removes a forum post document from the "forum_posts" collection.
func (c *Client) DeleteForumPost(ctx context.Context, id string) error {
	_, err := c.ts.Collection("forum_posts").Document(id).Delete(ctx)
	return err
}

// UpsertBook inserts or updates a book document in the "books" collection.
func (c *Client) UpsertBook(ctx context.Context, doc BookDocument) error {
	_, err := c.ts.Collection("books").Documents().Upsert(ctx, doc)
	return err
}

// DeleteBook removes a book document from the "books" collection.
func (c *Client) DeleteBook(ctx context.Context, id string) error {
	_, err := c.ts.Collection("books").Document(id).Delete(ctx)
	return err
}
