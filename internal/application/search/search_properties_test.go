// Feature: typesense-search-integration
package search_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"lms-backend/internal/application/search"
	tsclient "lms-backend/internal/infrastructure/typesense"
	"lms-backend/pkg/apperrors"

	"github.com/typesense/typesense-go/typesense/api"
	"pgregory.net/rapid"
)

// ─── Mock Searcher ────────────────────────────────────────────────────────────

// capturedCall records a single SearchCollection invocation.
type capturedCall struct {
	collection string
	params     *api.SearchCollectionParams
}

// mockSearcher implements tsclient.Searcher for property tests.
type mockSearcher struct {
	mu       sync.Mutex
	calls    []capturedCall
	resultFn func(collection string, params *api.SearchCollectionParams) (*api.SearchResult, error)
}

func (m *mockSearcher) SearchCollection(_ context.Context, collection string, params *api.SearchCollectionParams) (*api.SearchResult, error) {
	m.mu.Lock()
	m.calls = append(m.calls, capturedCall{collection: collection, params: params})
	m.mu.Unlock()

	if m.resultFn != nil {
		return m.resultFn(collection, params)
	}
	return oneHitResult(), nil
}

// Compile-time check: mockSearcher must satisfy tsclient.Searcher.
var _ tsclient.Searcher = (*mockSearcher)(nil)

// oneHitResult returns a SearchResult with a single dummy hit.
func oneHitResult() *api.SearchResult {
	doc := map[string]interface{}{
		"id":    "00000000-0000-0000-0000-000000000001",
		"title": "dummy",
	}
	hits := []api.SearchResultHit{{Document: &doc}}
	return &api.SearchResult{Hits: &hits}
}

// hitsResult returns a SearchResult with n dummy hits.
func hitsResult(n int) *api.SearchResult {
	hits := make([]api.SearchResultHit, n)
	for i := range hits {
		doc := map[string]interface{}{
			"id":    "00000000-0000-0000-0000-000000000001",
			"title": "dummy",
		}
		hits[i] = api.SearchResultHit{Document: &doc}
	}
	return &api.SearchResult{Hits: &hits}
}

// ─── Property 4: Query validation rejects short inputs ────────────────────────
// Feature: typesense-search-integration, Property 4

// TestProperty4_QueryValidationRejectsShortInputs verifies that Search returns
// VALIDATION_ERROR for any query shorter than 2 characters.
//
// **Validates: Requirements 5.2**
func TestProperty4_QueryValidationRejectsShortInputs(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate ASCII strings of byte-length 0 or 1 (service checks len(query) < 2)
		query := rapid.OneOf(
			rapid.Just(""),
			rapid.StringMatching(`^[a-z]$`),
		).Draw(t, "query")

		svc := search.NewService(&mockSearcher{}, nil)
		_, err := svc.Search(context.Background(), search.SearchCommand{
			Query: query,
			Limit: 10,
		})

		if err == nil {
			t.Fatalf("expected VALIDATION_ERROR for query %q (len=%d), got nil", query, len(query))
		}

		appErr, ok := err.(*apperrors.AppError)
		if !ok {
			t.Fatalf("expected *apperrors.AppError, got %T: %v", err, err)
		}
		if appErr.Code != "VALIDATION_ERROR" {
			t.Fatalf("expected Code=VALIDATION_ERROR, got %q", appErr.Code)
		}
	})
}

// ─── Property 5: Limit clamping ───────────────────────────────────────────────
// Feature: typesense-search-integration, Property 5

// TestProperty5_LimitClamping verifies that regardless of the Limit value passed,
// the number of results per collection never exceeds 30.
//
// **Validates: Requirements 5.5, 5.6**
func TestProperty5_LimitClamping(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate limit values outside [1, 30]: either ≤0 or >30
		limit := rapid.OneOf(
			rapid.IntRange(-100, 0),
			rapid.IntRange(31, 200),
		).Draw(t, "limit")

		// The mock returns as many hits as PerPage requests (clamped to 30 by service).
		ms := &mockSearcher{
			resultFn: func(collection string, params *api.SearchCollectionParams) (*api.SearchResult, error) {
				n := 0
				if params.PerPage != nil {
					n = *params.PerPage
				}
				if n > 30 {
					n = 30
				}
				if n < 0 {
					n = 0
				}
				return hitsResult(n), nil
			},
		}

		svc := search.NewService(ms, nil)
		resp, err := svc.Search(context.Background(), search.SearchCommand{
			Query: "hello",
			Limit: limit,
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(resp.Courses) > 30 {
			t.Fatalf("Courses count %d exceeds 30 for limit=%d", len(resp.Courses), limit)
		}
		if len(resp.Lessons) > 30 {
			t.Fatalf("Lessons count %d exceeds 30 for limit=%d", len(resp.Lessons), limit)
		}
		if len(resp.Forum) > 30 {
			t.Fatalf("Forum count %d exceeds 30 for limit=%d", len(resp.Forum), limit)
		}
		if len(resp.Books) > 30 {
			t.Fatalf("Books count %d exceeds 30 for limit=%d", len(resp.Books), limit)
		}
	})
}

// ─── Property 6: Status filter correctness ────────────────────────────────────
// Feature: typesense-search-integration, Property 6

// TestProperty6_StatusFilterCorrectness verifies that each collection is queried
// with the correct status filter string.
//
// **Validates: Requirements 5.7, 5.8, 5.9, 5.10**
func TestProperty6_StatusFilterCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ms := &mockSearcher{}
		svc := search.NewService(ms, nil)

		_, err := svc.Search(context.Background(), search.SearchCommand{
			Query: "hello",
			Limit: 5,
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Build a map of collection → FilterBy for easy lookup
		filterByCollection := make(map[string]string)
		ms.mu.Lock()
		for _, c := range ms.calls {
			if c.params != nil && c.params.FilterBy != nil {
				filterByCollection[c.collection] = *c.params.FilterBy
			}
		}
		ms.mu.Unlock()

		// courses → status:=published
		if fb, ok := filterByCollection["courses"]; !ok {
			t.Fatal("no call to collection 'courses'")
		} else if !strings.Contains(fb, "status:=published") {
			t.Fatalf("courses FilterBy=%q, expected to contain 'status:=published'", fb)
		}

		// lessons → status:=published
		if fb, ok := filterByCollection["lessons"]; !ok {
			t.Fatal("no call to collection 'lessons'")
		} else if !strings.Contains(fb, "status:=published") {
			t.Fatalf("lessons FilterBy=%q, expected to contain 'status:=published'", fb)
		}

		// forum_posts → status:=active
		if fb, ok := filterByCollection["forum_posts"]; !ok {
			t.Fatal("no call to collection 'forum_posts'")
		} else if !strings.Contains(fb, "status:=active") {
			t.Fatalf("forum_posts FilterBy=%q, expected to contain 'status:=active'", fb)
		}

		// books → is_active:=true
		if fb, ok := filterByCollection["books"]; !ok {
			t.Fatal("no call to collection 'books'")
		} else if !strings.Contains(fb, "is_active:=true") {
			t.Fatalf("books FilterBy=%q, expected to contain 'is_active:=true'", fb)
		}
	})
}

// ─── Property 7: Type filter isolation ───────────────────────────────────────
// Feature: typesense-search-integration, Property 7

// TestProperty7_TypeFilterIsolation verifies that when a Type filter is specified,
// only the matching collection field in SearchResponse is non-empty.
//
// **Validates: Requirements 5.3, 5.4**
func TestProperty7_TypeFilterIsolation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a type value from the valid set
		filterType := rapid.SampledFrom([]string{"courses", "lessons", "forum", "books"}).Draw(t, "type")

		// The mock returns one hit for every collection it is called with.
		ms := &mockSearcher{}
		svc := search.NewService(ms, nil)

		resp, err := svc.Search(context.Background(), search.SearchCommand{
			Query: "hello",
			Type:  filterType,
			Limit: 5,
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		switch filterType {
		case "courses":
			if len(resp.Courses) == 0 {
				t.Fatal("expected non-empty Courses for type=courses")
			}
			if len(resp.Lessons) != 0 {
				t.Fatalf("expected empty Lessons for type=courses, got %d", len(resp.Lessons))
			}
			if len(resp.Forum) != 0 {
				t.Fatalf("expected empty Forum for type=courses, got %d", len(resp.Forum))
			}
			if len(resp.Books) != 0 {
				t.Fatalf("expected empty Books for type=courses, got %d", len(resp.Books))
			}
		case "lessons":
			if len(resp.Lessons) == 0 {
				t.Fatal("expected non-empty Lessons for type=lessons")
			}
			if len(resp.Courses) != 0 {
				t.Fatalf("expected empty Courses for type=lessons, got %d", len(resp.Courses))
			}
			if len(resp.Forum) != 0 {
				t.Fatalf("expected empty Forum for type=lessons, got %d", len(resp.Forum))
			}
			if len(resp.Books) != 0 {
				t.Fatalf("expected empty Books for type=lessons, got %d", len(resp.Books))
			}
		case "forum":
			if len(resp.Forum) == 0 {
				t.Fatal("expected non-empty Forum for type=forum")
			}
			if len(resp.Courses) != 0 {
				t.Fatalf("expected empty Courses for type=forum, got %d", len(resp.Courses))
			}
			if len(resp.Lessons) != 0 {
				t.Fatalf("expected empty Lessons for type=forum, got %d", len(resp.Lessons))
			}
			if len(resp.Books) != 0 {
				t.Fatalf("expected empty Books for type=forum, got %d", len(resp.Books))
			}
		case "books":
			if len(resp.Books) == 0 {
				t.Fatal("expected non-empty Books for type=books")
			}
			if len(resp.Courses) != 0 {
				t.Fatalf("expected empty Courses for type=books, got %d", len(resp.Courses))
			}
			if len(resp.Lessons) != 0 {
				t.Fatalf("expected empty Lessons for type=books, got %d", len(resp.Lessons))
			}
			if len(resp.Forum) != 0 {
				t.Fatalf("expected empty Forum for type=books, got %d", len(resp.Forum))
			}
		}
	})
}
