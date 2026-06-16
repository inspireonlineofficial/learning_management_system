package typesense

import (
	"context"
	"errors"
	"net/http"

	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
)

// collectionSchemas defines the four Typesense collection schemas.
var collectionSchemas = []*api.CollectionSchema{
	{
		Name: "courses",
		Fields: []api.Field{
			{Name: "id", Type: "string"},
			{Name: "title", Type: "string"},
			{Name: "slug", Type: "string"},
			{Name: "short_description", Type: "string"},
			{Name: "subject", Type: "string"},
			{Name: "level", Type: "string"},
			{Name: "status", Type: "string"},
			{Name: "rating_average", Type: "float"},
		},
	},
	{
		Name: "lessons",
		Fields: []api.Field{
			{Name: "id", Type: "string"},
			{Name: "title", Type: "string"},
			{Name: "course_id", Type: "string"},
			{Name: "course_title", Type: "string"},
			{Name: "is_free_preview", Type: "bool"},
			{Name: "status", Type: "string"},
		},
	},
	{
		Name: "forum_posts",
		Fields: []api.Field{
			{Name: "id", Type: "string"},
			{Name: "title", Type: "string"},
			{Name: "body_excerpt", Type: "string"},
			{Name: "status", Type: "string"},
			{Name: "created_at", Type: "int64"},
		},
	},
	{
		Name: "books",
		Fields: []api.Field{
			{Name: "id", Type: "string"},
			{Name: "title", Type: "string"},
			{Name: "author", Type: "string"},
			{Name: "format", Type: "string"},
			{Name: "is_active", Type: "bool"},
		},
	},
}

// EnsureCollections creates each of the four Typesense collections.
// If a collection already exists (HTTP 409 Conflict), the error is swallowed
// and the next collection is processed.
func (c *Client) EnsureCollections(ctx context.Context) error {
	for _, schema := range collectionSchemas {
		_, err := c.ts.Collections().Create(ctx, schema)
		if err != nil {
			var httpErr *typesense.HTTPError
			if errors.As(err, &httpErr) && httpErr.Status == http.StatusConflict {
				// Collection already exists — continue.
				continue
			}
			return err
		}
	}
	return nil
}
