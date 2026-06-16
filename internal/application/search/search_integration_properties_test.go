//go:build integration

// Feature: typesense-search-integration
package search_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	tsclient "lms-backend/internal/infrastructure/typesense"

	"github.com/google/uuid"
	"github.com/typesense/typesense-go/typesense/api"
	"pgregory.net/rapid"
)

// newIntegrationClient creates a real Typesense client using env vars with fallbacks.
// Returns nil and skips the test if the client cannot connect.
func newIntegrationClient(t *testing.T) *tsclient.Client {
	t.Helper()
	host := os.Getenv("TYPESENSE_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("TYPESENSE_PORT")
	if port == "" {
		port = "8108"
	}
	apiKey := os.Getenv("TYPESENSE_API_KEY")
	if apiKey == "" {
		apiKey = "xyz"
	}

	client, err := tsclient.NewClient(host, port, apiKey)
	if err != nil {
		t.Skipf("requires running Typesense: %v", err)
	}

	// Verify connectivity by ensuring collections exist.
	ctx := context.Background()
	if err := client.EnsureCollections(ctx); err != nil {
		t.Skipf("requires running Typesense: EnsureCollections failed: %v", err)
	}

	return client
}

// courseDocGenerator returns a rapid generator for CourseDocument with a unique title.
func courseDocGenerator() *rapid.Generator[tsclient.CourseDocument] {
	return rapid.Custom(func(t *rapid.T) tsclient.CourseDocument {
		id := uuid.New().String()
		title := fmt.Sprintf("test-course-%s", uuid.New().String())
		slug := rapid.StringMatching(`^[a-z][a-z0-9-]{3,20}$`).Draw(t, "slug")
		subject := rapid.SampledFrom([]string{"math", "science", "history", "art"}).Draw(t, "subject")
		level := rapid.SampledFrom([]string{"beginner", "intermediate", "advanced"}).Draw(t, "level")
		rating := rapid.Float32Range(0, 5).Draw(t, "rating")
		return tsclient.CourseDocument{
			ID:               id,
			Title:            title,
			Slug:             slug,
			ShortDescription: "A test course description",
			Subject:          subject,
			Level:            level,
			Status:           "published",
			RatingAverage:    rating,
		}
	})
}

// containsID checks whether a search result contains a document with the given ID.
func containsID(result *api.SearchResult, id string) bool {
	if result == nil || result.Hits == nil {
		return false
	}
	for _, hit := range *result.Hits {
		if hit.Document == nil {
			continue
		}
		if docID, ok := (*hit.Document)["id"]; ok {
			if fmt.Sprintf("%v", docID) == id {
				return true
			}
		}
	}
	return false
}

// ─── Property 1: Upsert then search round-trip ────────────────────────────────
// Feature: typesense-search-integration, Property 1

// TestProperty1_UpsertThenSearchRoundTrip verifies that upserting a course document
// and then searching by its exact title returns a result containing that document's ID.
//
// **Validates: Requirements 9.1**
func TestProperty1_UpsertThenSearchRoundTrip(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		doc := courseDocGenerator().Draw(t, "doc")

		if err := client.UpsertCourse(ctx, doc); err != nil {
			t.Fatalf("UpsertCourse failed: %v", err)
		}
		t.Cleanup(func() {
			// Best-effort cleanup; ignore errors.
			_ = client.DeleteCourse(context.Background(), doc.ID)
		})

		perPage := 10
		filterBy := fmt.Sprintf("title:=%s", doc.Title)
		result, err := client.SearchCollection(ctx, "courses", &api.SearchCollectionParams{
			Q:        doc.Title,
			QueryBy:  "title",
			FilterBy: &filterBy,
			PerPage:  &perPage,
		})
		if err != nil {
			t.Fatalf("SearchCollection failed: %v", err)
		}

		if !containsID(result, doc.ID) {
			t.Fatalf("expected document ID %q in search results for title %q, but it was absent", doc.ID, doc.Title)
		}
	})
}

// ─── Property 2: Delete then search exclusion ─────────────────────────────────
// Feature: typesense-search-integration, Property 2

// TestProperty2_DeleteThenSearchExclusion verifies that after deleting a course document,
// searching by its exact title does not return that document's ID.
//
// **Validates: Requirements 9.2**
func TestProperty2_DeleteThenSearchExclusion(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		doc := courseDocGenerator().Draw(t, "doc")

		if err := client.UpsertCourse(ctx, doc); err != nil {
			t.Fatalf("UpsertCourse failed: %v", err)
		}

		if err := client.DeleteCourse(ctx, doc.ID); err != nil {
			t.Fatalf("DeleteCourse failed: %v", err)
		}

		perPage := 10
		filterBy := fmt.Sprintf("title:=%s", doc.Title)
		result, err := client.SearchCollection(ctx, "courses", &api.SearchCollectionParams{
			Q:        doc.Title,
			QueryBy:  "title",
			FilterBy: &filterBy,
			PerPage:  &perPage,
		})
		if err != nil {
			t.Fatalf("SearchCollection failed: %v", err)
		}

		if containsID(result, doc.ID) {
			t.Fatalf("expected document ID %q to be absent after deletion, but it was found in results", doc.ID)
		}
	})
}

// ─── Property 3: Upsert idempotence ───────────────────────────────────────────
// Feature: typesense-search-integration, Property 3

// TestProperty3_UpsertIdempotence verifies that upserting the same document twice
// (same ID, updated title) results in exactly one document with the latest field values.
//
// **Validates: Requirements 9.3**
func TestProperty3_UpsertIdempotence(t *testing.T) {
	client := newIntegrationClient(t)
	ctx := context.Background()

	rapid.Check(t, func(t *rapid.T) {
		doc := courseDocGenerator().Draw(t, "doc")

		// First upsert.
		if err := client.UpsertCourse(ctx, doc); err != nil {
			t.Fatalf("first UpsertCourse failed: %v", err)
		}

		// Second upsert with updated title (same ID).
		updatedTitle := fmt.Sprintf("updated-%s", uuid.New().String())
		doc.Title = updatedTitle
		if err := client.UpsertCourse(ctx, doc); err != nil {
			t.Fatalf("second UpsertCourse failed: %v", err)
		}
		t.Cleanup(func() {
			_ = client.DeleteCourse(context.Background(), doc.ID)
		})

		// Search by the updated title — should find exactly one hit with the updated title.
		perPage := 10
		filterBy := fmt.Sprintf("title:=%s", updatedTitle)
		result, err := client.SearchCollection(ctx, "courses", &api.SearchCollectionParams{
			Q:        updatedTitle,
			QueryBy:  "title",
			FilterBy: &filterBy,
			PerPage:  &perPage,
		})
		if err != nil {
			t.Fatalf("SearchCollection failed: %v", err)
		}

		if result.Hits == nil {
			t.Fatalf("expected hits to be non-nil")
		}

		// Assert exactly one hit with the correct ID and updated title.
		hits := *result.Hits
		matchCount := 0
		for _, hit := range hits {
			if hit.Document == nil {
				continue
			}
			docMap := *hit.Document
			if fmt.Sprintf("%v", docMap["id"]) == doc.ID {
				matchCount++
				if fmt.Sprintf("%v", docMap["title"]) != updatedTitle {
					t.Fatalf("expected title %q for document ID %q, got %q", updatedTitle, doc.ID, docMap["title"])
				}
			}
		}

		if matchCount != 1 {
			t.Fatalf("expected exactly 1 document with ID %q, found %d", doc.ID, matchCount)
		}
	})
}
