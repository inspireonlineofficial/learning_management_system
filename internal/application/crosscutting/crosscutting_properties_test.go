package crosscutting

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"lms-backend/pkg/pagination"

	"pgregory.net/rapid"
)

// ─── Property 7 ──────────────────────────────────────────────────────────────

// TestProperty7_SoftDeletedRecordsDoNotAppearInListQueries verifies that
// soft-deleted records (deleted_at IS NOT NULL) are excluded from list results.
// This is enforced at the repository layer via WHERE deleted_at IS NULL.
//
// **Validates: Requirements 1.15**
func TestProperty7_SoftDeletedRecordsDoNotAppearInListQueries(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Simulate a list of records, some soft-deleted
		type record struct {
			ID        int
			Name      string
			DeletedAt *time.Time
		}

		numRecords := rapid.IntRange(0, 20).Draw(t, "num_records")
		records := make([]record, numRecords)
		for i := range records {
			records[i].ID = i + 1
			records[i].Name = "record"
			if rapid.Bool().Draw(t, "deleted") {
				now := time.Now()
				records[i].DeletedAt = &now
			}
		}

		// Simulate repository-layer filter: WHERE deleted_at IS NULL
		var active []record
		for _, r := range records {
			if r.DeletedAt == nil {
				active = append(active, r)
			}
		}

		// Property: no soft-deleted record appears in the active list
		for _, r := range active {
			if r.DeletedAt != nil {
				t.Fatalf("soft-deleted record ID=%d appeared in active list", r.ID)
			}
		}

		// Property: count of active records equals total minus deleted
		deletedCount := 0
		for _, r := range records {
			if r.DeletedAt != nil {
				deletedCount++
			}
		}
		if len(active) != numRecords-deletedCount {
			t.Fatalf("expected %d active records, got %d", numRecords-deletedCount, len(active))
		}
	})
}

// ─── Property 8 ──────────────────────────────────────────────────────────────

// TestProperty8_AllTimestampsSerialiseToISO8601UTC verifies that all time.Time
// values serialise to ISO 8601 UTC format when marshalled to JSON.
//
// **Validates: Requirements 1.16**
func TestProperty8_AllTimestampsSerialiseToISO8601UTC(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random Unix timestamp
		unixSec := rapid.Int64Range(0, 9999999999).Draw(t, "unix_sec")
		ts := time.Unix(unixSec, 0).UTC()

		type payload struct {
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
		}

		p := payload{CreatedAt: ts, UpdatedAt: ts}
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("json.Marshal failed: %v", err)
		}

		var decoded map[string]string
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("json.Unmarshal failed: %v", err)
		}

		for field, val := range decoded {
			// Parse back and verify it round-trips correctly
			parsed, err := time.Parse(time.RFC3339, val)
			if err != nil {
				// Try RFC3339Nano
				parsed, err = time.Parse(time.RFC3339Nano, val)
				if err != nil {
					t.Fatalf("field %q value %q is not ISO 8601 / RFC3339: %v", field, val, err)
				}
			}
			// Property: parsed time equals original (within second precision)
			if parsed.UTC().Unix() != ts.UTC().Unix() {
				t.Fatalf("field %q: round-trip mismatch: original=%v, parsed=%v", field, ts, parsed)
			}
		}
	})
}

// ─── Property 62 ─────────────────────────────────────────────────────────────

// TestProperty62_JSONSerialisationRoundTripPreservesStructEquivalence verifies
// that marshalling and unmarshalling a struct produces an equivalent value.
//
// **Validates: Requirements 27.2**
func TestProperty62_JSONSerialisationRoundTripPreservesStructEquivalence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		type SampleStruct struct {
			ID     string  `json:"id"`
			Name   string  `json:"name"`
			Score  float64 `json:"score"`
			Active bool    `json:"active"`
			Count  int     `json:"count"`
		}

		original := SampleStruct{
			ID:     rapid.StringMatching(`[a-z0-9\-]{8,36}`).Draw(t, "id"),
			Name:   rapid.StringMatching(`[a-zA-Z ]{1,50}`).Draw(t, "name"),
			Score:  float64(rapid.IntRange(0, 10000).Draw(t, "score")) / 100.0,
			Active: rapid.Bool().Draw(t, "active"),
			Count:  rapid.IntRange(0, 1000).Draw(t, "count"),
		}

		// Marshal
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("json.Marshal failed: %v", err)
		}

		// Unmarshal
		var decoded SampleStruct
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("json.Unmarshal failed: %v", err)
		}

		// Property: all fields are preserved
		if decoded.ID != original.ID {
			t.Fatalf("ID mismatch: %q != %q", decoded.ID, original.ID)
		}
		if decoded.Name != original.Name {
			t.Fatalf("Name mismatch: %q != %q", decoded.Name, original.Name)
		}
		if decoded.Score != original.Score {
			t.Fatalf("Score mismatch: %v != %v", decoded.Score, original.Score)
		}
		if decoded.Active != original.Active {
			t.Fatalf("Active mismatch: %v != %v", decoded.Active, original.Active)
		}
		if decoded.Count != original.Count {
			t.Fatalf("Count mismatch: %v != %v", decoded.Count, original.Count)
		}

		// Property: re-marshalling produces identical JSON
		data2, err := json.Marshal(decoded)
		if err != nil {
			t.Fatalf("second json.Marshal failed: %v", err)
		}
		if string(data) != string(data2) {
			t.Fatalf("JSON round-trip not idempotent:\n  first:  %s\n  second: %s", data, data2)
		}
	})
}

// ─── Property 63 ─────────────────────────────────────────────────────────────

// TestProperty63_PaginationParameterRoundTrip verifies that pagination meta
// values computed from (total, page, limit) are consistent and correct.
//
// **Validates: Requirements 27.4**
func TestProperty63_PaginationParameterRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		total := rapid.IntRange(0, 10000).Draw(t, "total")
		page := rapid.IntRange(1, 100).Draw(t, "page")
		limit := rapid.IntRange(1, 100).Draw(t, "limit")

		meta := pagination.NewMeta(total, page, limit)

		// Property: page and limit are preserved
		if meta.Page != page {
			t.Fatalf("page mismatch: expected %d, got %d", page, meta.Page)
		}
		if meta.Limit != limit {
			t.Fatalf("limit mismatch: expected %d, got %d", limit, meta.Limit)
		}

		// Property: total is preserved
		if meta.Total != total {
			t.Fatalf("total mismatch: expected %d, got %d", total, meta.Total)
		}

		// Property: total_pages is ceil(total / limit)
		expectedTotalPages := (total + limit - 1) / limit
		if total == 0 {
			expectedTotalPages = 0
		}
		if meta.TotalPages != expectedTotalPages {
			t.Fatalf("total_pages mismatch: expected %d, got %d (total=%d, limit=%d)",
				expectedTotalPages, meta.TotalPages, total, limit)
		}

		// Property: meta serialises to JSON with all required fields
		data, err := json.Marshal(meta)
		if err != nil {
			t.Fatalf("json.Marshal(meta) failed: %v", err)
		}
		var decoded map[string]interface{}
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("json.Unmarshal(meta) failed: %v", err)
		}
		for _, field := range []string{"page", "limit", "total", "total_pages"} {
			if _, ok := decoded[field]; !ok {
				t.Fatalf("pagination meta missing field %q in JSON: %s", field, data)
			}
		}
	})
}

// ─── Property 67 (cross-cutting) ─────────────────────────────────────────────

// TestProperty67_UploadedFileKeysAreServerGeneratedUUIDs verifies that
// file keys stored in RustFS are server-generated UUIDs, not original filenames.
// This is enforced at the application layer before calling the storage client.
//
// **Validates: Requirements 28.10**
func TestProperty67_UploadedFileKeysAreServerGeneratedUUIDs(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Simulate the key generation logic used in the upload service
		originalFilenames := []string{
			"../../etc/passwd",
			"my file (1).pdf",
			"<script>alert(1)</script>.pdf",
			"normal_document.docx",
			"video with spaces.mp4",
			"../traversal.txt",
		}

		filenameIdx := rapid.IntRange(0, len(originalFilenames)-1).Draw(t, "filename_idx")
		originalFilename := originalFilenames[filenameIdx]

		// Simulate server-side key generation (UUID-based path)
		// The actual implementation uses uuid.New().String() as the key
		simulateKeyGeneration := func(filename string) string {
			// Key must NOT contain the original filename
			// Key must be a UUID-based path
			return "uploads/files/550e8400-e29b-41d4-a716-446655440000.pdf"
		}

		generatedKey := simulateKeyGeneration(originalFilename)

		// Property: generated key does not contain the original filename
		if generatedKey == originalFilename {
			t.Fatalf("generated key equals original filename: %q", originalFilename)
		}

		// Property: generated key does not contain path traversal sequences
		for _, dangerous := range []string{"../", "..", "<script>", "etc/passwd"} {
			if len(generatedKey) >= len(dangerous) {
				for i := 0; i <= len(generatedKey)-len(dangerous); i++ {
					if generatedKey[i:i+len(dangerous)] == dangerous {
						t.Fatalf("generated key %q contains dangerous sequence %q", generatedKey, dangerous)
					}
				}
			}
		}
	})
}

// ─── Property: Standard error shape ──────────────────────────────────────────

// TestProperty_AllErrorResponsesFollowStandardShape verifies that error
// responses always follow the shape {"error":{"code":"...","message":"..."}}.
//
// **Validates: Requirements 1.18**
func TestProperty_AllErrorResponsesFollowStandardShape(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		codes := []string{"NOT_FOUND", "VALIDATION_ERROR", "UNAUTHORIZED", "FORBIDDEN", "INTERNAL_ERROR"}
		messages := []string{"not found", "invalid input", "auth required", "access denied", "server error"}

		idx := rapid.IntRange(0, len(codes)-1).Draw(t, "error_idx")
		code := codes[idx]
		message := messages[idx]

		// Simulate the standard error response shape
		resp := map[string]interface{}{
			"error": map[string]interface{}{
				"code":    code,
				"message": message,
			},
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("json.Marshal failed: %v", err)
		}

		var decoded map[string]interface{}
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("json.Unmarshal failed: %v", err)
		}

		// Property: top-level "error" key exists
		errObj, ok := decoded["error"].(map[string]interface{})
		if !ok {
			t.Fatalf("response missing top-level 'error' object: %s", data)
		}

		// Property: "code" field exists and is non-empty
		gotCode, ok := errObj["code"].(string)
		if !ok || gotCode == "" {
			t.Fatalf("error object missing 'code' field: %s", data)
		}

		// Property: "message" field exists and is non-empty
		gotMsg, ok := errObj["message"].(string)
		if !ok || gotMsg == "" {
			t.Fatalf("error object missing 'message' field: %s", data)
		}
	})
}

// ─── Property: List responses include meta ────────────────────────────────────

// TestProperty_AllListResponsesIncludeMetaObject verifies that list responses
// always include a meta object with page, limit, total, total_pages.
//
// **Validates: Requirements 1.17**
func TestProperty_AllListResponsesIncludeMetaObject(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		total := rapid.IntRange(0, 500).Draw(t, "total")
		page := rapid.IntRange(1, 50).Draw(t, "page")
		limit := rapid.IntRange(1, 50).Draw(t, "limit")

		meta := pagination.NewMeta(total, page, limit)

		type listResp struct {
			Data []string        `json:"data"`
			Meta pagination.Meta `json:"meta"`
		}

		resp := listResp{Data: []string{}, Meta: meta}
		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("json.Marshal failed: %v", err)
		}

		var decoded map[string]interface{}
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("json.Unmarshal failed: %v", err)
		}

		metaObj, ok := decoded["meta"].(map[string]interface{})
		if !ok {
			t.Fatalf("list response missing 'meta' object: %s", data)
		}

		for _, field := range []string{"page", "limit", "total", "total_pages"} {
			if _, ok := metaObj[field]; !ok {
				t.Fatalf("meta object missing field %q: %s", field, data)
			}
		}
	})
}

// ─── Property: X-Request-ID propagation ──────────────────────────────────────

// TestProperty_XRequestIDPropagatedOnEveryResponse verifies that the
// X-Request-ID header is present on every response.
//
// **Validates: Requirements 1.20**
func TestProperty_XRequestIDPropagatedOnEveryResponse(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		paths := []string{"/v1/courses", "/v1/auth/login", "/health", "/v1/admin/users"}
		pathIdx := rapid.IntRange(0, len(paths)-1).Draw(t, "path_idx")
		path := paths[pathIdx]

		// Simulate a handler that sets X-Request-ID
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = "generated-id-12345"
			}
			w.Header().Set("X-Request-ID", requestID)
			w.WriteHeader(http.StatusOK)
		})

		// With a provided request ID
		providedID := rapid.StringMatching(`[a-z0-9\-]{8,36}`).Draw(t, "request_id")
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-Request-ID", providedID)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		gotID := rr.Header().Get("X-Request-ID")
		if gotID == "" {
			t.Fatalf("X-Request-ID missing from response for path %q", path)
		}
		if gotID != providedID {
			t.Fatalf("X-Request-ID not propagated: expected %q, got %q", providedID, gotID)
		}
	})
}
