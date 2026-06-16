package typesense

import (
	"context"

	"github.com/typesense/typesense-go/typesense/api"
)

// NoopSearcher implements Searcher with a no-op method for use before the real
// Typesense client is wired in main.go.
type NoopSearcher struct{}

func (n *NoopSearcher) SearchCollection(_ context.Context, _ string, _ *api.SearchCollectionParams) (*api.SearchResult, error) {
	return &api.SearchResult{}, nil
}
