package handlers_test

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"

	"pgregory.net/rapid"
)

// swaggerSpec holds the parsed swagger.json loaded once for all property tests.
var (
	swaggerSpec     map[string]interface{}
	swaggerSpecOnce sync.Once
	swaggerSpecErr  error
)

func loadSwaggerSpec(t *testing.T) map[string]interface{} {
	t.Helper()
	swaggerSpecOnce.Do(func() {
		data, err := os.ReadFile("../../../../docs/swagger.json")
		if err != nil {
			swaggerSpecErr = err
			return
		}
		var spec map[string]interface{}
		if err := json.Unmarshal(data, &spec); err != nil {
			swaggerSpecErr = err
			return
		}
		swaggerSpec = spec
	})
	if swaggerSpecErr != nil {
		t.Fatalf("failed to load docs/swagger.json: %v", swaggerSpecErr)
	}
	return swaggerSpec
}

// getPaths returns the "paths" map from the spec.
func getPaths(spec map[string]interface{}) map[string]interface{} {
	paths, _ := spec["paths"].(map[string]interface{})
	return paths
}

// getOperation returns the operation object for a given path+method, or nil.
func getOperation(spec map[string]interface{}, path, method string) map[string]interface{} {
	paths := getPaths(spec)
	if paths == nil {
		return nil
	}
	pathItem, ok := paths[path].(map[string]interface{})
	if !ok {
		return nil
	}
	op, _ := pathItem[strings.ToLower(method)].(map[string]interface{})
	return op
}

// routeEntry represents a known registered route (path + HTTP method).
type routeEntry struct {
	Path   string
	Method string
}

// knownRoutes is the full set of routes registered in server.go.
var knownRoutes = []routeEntry{
	// JWKS
	{"/v1/.well-known/jwks.json", "get"},
	// Auth public
	{"/v1/auth/register", "post"},
	{"/v1/auth/verify-otp", "post"},
	{"/v1/auth/resend-otp", "post"},
	{"/v1/auth/login", "post"},
	{"/v1/auth/refresh", "post"},
	{"/v1/auth/logout", "post"},
	{"/v1/auth/forgot-password", "post"},
	{"/v1/auth/reset-password", "post"},
	// Admin auth
	{"/v1/auth/admin/login", "post"},
	{"/v1/auth/admin/verify-otp", "post"},
	{"/v1/auth/admin/resend-otp", "post"},
	// OAuth
	{"/v1/auth/oauth/{provider}", "get"},
	{"/v1/auth/oauth/{provider}/callback", "get"},
	// Authenticated user
	{"/v1/auth/me", "patch"},
	{"/v1/auth/me/settings", "get"},
	{"/v1/auth/me/settings", "patch"},
	{"/v1/auth/me/change-password", "post"},
	{"/v1/auth/me/oauth/connect", "post"},
	{"/v1/auth/me/oauth/{provider}", "delete"},
	// Courses public
	{"/v1/courses", "get"},
	{"/v1/courses/{courseId}", "get"},
	{"/v1/courses/{courseId}/reviews", "get"},
	{"/v1/courses/{courseId}/reviews", "post"},
	{"/v1/public/slides", "get"},
	// Teacher courses
	{"/v1/teacher/courses", "get"},
	{"/v1/teacher/courses", "post"},
	{"/v1/teacher/courses/{courseId}", "patch"},
	{"/v1/teacher/courses/{courseId}/submit", "post"},
	{"/v1/teacher/courses/{courseId}/preview", "get"},
	{"/v1/teacher/courses/{courseId}/modules", "post"},
	{"/v1/teacher/modules/{moduleId}", "patch"},
	{"/v1/teacher/modules/{moduleId}", "delete"},
	{"/v1/teacher/modules/{moduleId}/chapters", "post"},
	{"/v1/teacher/chapters/{chapterId}", "patch"},
	{"/v1/teacher/chapters/{chapterId}", "delete"},
	{"/v1/teacher/chapters/{chapterId}/lessons", "post"},
	{"/v1/teacher/lessons/{lessonId}", "patch"},
	{"/v1/teacher/lessons/{lessonId}", "delete"},
	{"/v1/teacher/content/reorder", "patch"},
	{"/v1/admin/courses", "get"},
	{"/v1/admin/courses/{courseId}", "get"},
	{"/v1/admin/courses/{courseId}", "delete"},
	{"/v1/admin/slides", "get"},
	{"/v1/admin/slides", "post"},
	{"/v1/admin/slides/{slideId}", "patch"},
	{"/v1/admin/slides/reorder", "post"},
	{"/v1/admin/slides/{slideId}/deactivate", "post"},
	// Admin courses
	{"/v1/admin/courses", "get"},
	{"/v1/admin/courses/{courseId}", "get"},
	{"/v1/admin/courses/{courseId}", "delete"},
	{"/v1/admin/courses/{courseId}/review", "post"},
	// Promotional slides
	{"/v1/admin/slides", "get"},
	{"/v1/admin/slides", "post"},
	{"/v1/admin/slides/{slideId}", "patch"},
	{"/v1/admin/slides/reorder", "post"},
	{"/v1/admin/slides/{slideId}/deactivate", "post"},
	// Uploads
	{"/v1/uploads/video", "post"},
	{"/v1/uploads/video/{videoId}/status", "get"},
	{"/v1/uploads/file", "post"},
	// Enrollments
	{"/v1/enrollments", "post"},
	{"/v1/student/enrollments", "get"},
	{"/v1/stream/lessons/{lessonId}/signed-url", "get"},
	{"/v1/enrollments/{courseId}/lessons/{lessonId}/progress", "post"},
	// Teacher quizzes
	{"/v1/teacher/courses/{courseId}/quizzes", "post"},
	{"/v1/teacher/assignments/{assignmentId}/submissions", "get"},
	{"/v1/teacher/assignments/{assignmentId}/submissions/{submissionId}", "get"},
	{"/v1/student/assessments", "get"},
	{"/v1/student/assessments/{quizId}", "get"},
	{"/v1/student/assessments/{quizId}/attempts/{attemptId}", "get"},
	{"/v1/student/assignments", "get"},
	{"/v1/student/assignments/{assignmentId}", "get"},
	{"/v1/teacher/courses/{courseId}/quizzes", "get"},
	// Teacher assignments
	{"/v1/teacher/courses/{courseId}/assignments", "post"},
	{"/v1/teacher/assignments/{assignmentId}/submissions", "get"},
	{"/v1/teacher/assignments/{assignmentId}/submissions/{submissionId}", "get"},
	{"/v1/teacher/assignments/{assignmentId}/submissions/{submissionId}/grade", "post"},
	// Student quizzes
	{"/v1/student/assessments", "get"},
	{"/v1/student/assessments/{quizId}", "get"},
	{"/v1/student/assessments/{quizId}/attempts/{attemptId}", "get"},
	{"/v1/quizzes/{quizId}/attempts", "post"},
	{"/v1/quizzes/{quizId}/attempts/{attemptId}/submit", "post"},
	// Student assignments
	{"/v1/student/assignments", "get"},
	{"/v1/student/assignments/{assignmentId}", "get"},
	{"/v1/assignments/{assignmentId}/submissions", "post"},
	// Points
	{"/v1/student/points", "get"},
	{"/v1/student/points/history", "get"},
	{"/v1/student/leaderboard/opt-out", "patch"},
	{"/v1/leaderboard", "get"},
	{"/v1/admin/points/config", "patch"},
	// Certificates
	{"/v1/student/certificates/{courseId}", "get"},
	{"/v1/certificates/verify/{verificationId}", "get"},
	// Bookshop public
	{"/v1/bookshop/books", "get"},
	{"/v1/bookshop/books/{bookId}", "get"},
	{"/v1/bookshop/books/{bookId}/preview", "get"},
	// Bookshop student
	{"/v1/student/bookshop/reader/{bookId}/access", "get"},
	{"/v1/student/bookshop/reader/{bookId}/bookmark", "post"},
	{"/v1/student/bookshop/orders", "get"},
	{"/v1/student/bookshop/orders/{orderId}", "get"},
	// Bookshop admin
	{"/v1/admin/bookshop/books", "get"},
	{"/v1/admin/bookshop/books", "post"},
	{"/v1/admin/bookshop/books/{bookId}", "patch"},
	{"/v1/admin/bookshop/books/{bookId}/cover", "post"},
	{"/v1/admin/bookshop/orders", "get"},
	{"/v1/student/bookshop/orders/{orderId}", "get"},
	{"/v1/admin/bookshop/orders/{orderId}", "patch"},
	{"/v1/admin/bookshop/refunds", "post"},
	// Forum public
	{"/v1/forum/posts", "get"},
	{"/v1/forum/posts/{postId}/comments", "get"},
	// Forum authenticated
	{"/v1/forum/posts", "post"},
	{"/v1/forum/posts/{postId}", "patch"},
	{"/v1/admin/forum/posts", "get"},
	{"/v1/admin/forum/posts/{postId}/approve", "post"},
	{"/v1/admin/forum/posts/{postId}/reject", "post"},
	{"/v1/forum/posts/{postId}", "delete"},
	{"/v1/forum/posts/{postId}/comments", "post"},
	{"/v1/forum/posts/{postId}/comments/{commentId}", "patch"},
	{"/v1/forum/posts/{postId}/comments/{commentId}", "delete"},
	{"/v1/forum/posts/{postId}/upvote", "post"},
	{"/v1/forum/posts/{postId}/flag", "post"},
	// Forum admin
	{"/v1/admin/moderation", "get"},
	{"/v1/admin/moderation/{flagId}/action", "post"},
	{"/v1/admin/forum/posts", "get"},
	{"/v1/admin/forum/posts/{postId}/approve", "post"},
	{"/v1/admin/forum/posts/{postId}/reject", "post"},
	// Notifications
	{"/v1/notifications", "get"},
	{"/v1/notifications/{notificationId}/read", "patch"},
	{"/v1/notifications/read-all", "patch"},
	{"/v1/admin/notifications/templates", "get"},
	{"/v1/admin/notifications/templates/{templateId}", "patch"},
	{"/v1/admin/notifications/broadcast", "post"},
	// Analytics
	{"/v1/admin/analytics/overview", "get"},
	{"/v1/admin/analytics/courses/{courseId}", "get"},
	{"/v1/admin/analytics/courses/{courseId}/students", "get"},
	{"/v1/admin/analytics/students/{studentId}", "get"},
	{"/v1/teacher/analytics", "get"},
	// Purchase approvals
	{"/v1/purchase-requests", "post"},
	{"/v1/student/purchase-requests", "get"},
	{"/v1/admin/purchase-requests", "get"},
	{"/v1/admin/purchase-requests/export", "get"},
	{"/v1/admin/purchase-requests/{requestId}/approve", "post"},
	{"/v1/admin/purchase-requests/{requestId}/reject", "post"},
	// System config
	{"/v1/admin/system/settings", "get"},
	{"/v1/admin/system/settings", "patch"},
	{"/v1/admin/system/settings/history", "get"},
	{"/v1/admin/system/settings/rollback/{historyId}", "post"},
	// Live sessions
	{"/v1/teacher/live-sessions", "post"},
	{"/v1/teacher/live-sessions", "get"},
	{"/v1/teacher/live-sessions/{sessionId}", "get"},
	{"/v1/teacher/live-sessions/{sessionId}/start", "post"},
	{"/v1/teacher/live-sessions/{sessionId}/end", "post"},
	{"/v1/teacher/live-sessions/{sessionId}", "patch"},
	{"/v1/teacher/live-sessions/{sessionId}/attendance", "get"},
	{"/v1/live-sessions/{sessionId}/join", "post"},
	{"/v1/student/live-sessions", "get"},
	{"/v1/student/live-sessions/{sessionId}", "get"},
	// RBAC
	{"/v1/admin/rbac/roles", "get"},
	{"/v1/admin/rbac/roles/{roleId}", "patch"},
	// Audit
	{"/v1/admin/audit-logs", "get"},
	// Search
	{"/v1/search", "get"},
}

// knownAuthRoutes is the set of routes that require BearerAuth (wrapped with withAuth or withAuthAndRole).
var knownAuthRoutes = []routeEntry{
	{"/v1/auth/me", "patch"},
	{"/v1/auth/me/change-password", "post"},
	{"/v1/auth/me/oauth/connect", "post"},
	{"/v1/auth/me/oauth/{provider}", "delete"},
	{"/v1/courses/{courseId}/reviews", "post"},
	{"/v1/teacher/courses", "get"},
	{"/v1/teacher/courses", "post"},
	{"/v1/teacher/courses/{courseId}", "patch"},
	{"/v1/teacher/courses/{courseId}/modules", "post"},
	{"/v1/teacher/modules/{moduleId}", "patch"},
	{"/v1/teacher/modules/{moduleId}", "delete"},
	{"/v1/teacher/modules/{moduleId}/chapters", "post"},
	{"/v1/teacher/chapters/{chapterId}", "patch"},
	{"/v1/teacher/chapters/{chapterId}", "delete"},
	{"/v1/teacher/chapters/{chapterId}/lessons", "post"},
	{"/v1/teacher/lessons/{lessonId}", "patch"},
	{"/v1/teacher/lessons/{lessonId}", "delete"},
	{"/v1/teacher/content/reorder", "patch"},
	{"/v1/enrollments", "post"},
	{"/v1/student/enrollments", "get"},
	{"/v1/stream/lessons/{lessonId}/signed-url", "get"},
	{"/v1/teacher/courses/{courseId}/quizzes", "post"},
	{"/v1/quizzes/{quizId}/attempts", "post"},
	{"/v1/student/points", "get"},
	{"/v1/leaderboard", "get"},
	{"/v1/admin/points/config", "patch"},
	{"/v1/student/certificates/{courseId}", "get"},
	{"/v1/admin/bookshop/books", "get"},
	{"/v1/admin/bookshop/books", "post"},
	{"/v1/admin/bookshop/books/{bookId}/cover", "post"},
	{"/v1/admin/bookshop/orders", "get"},
	{"/v1/student/bookshop/orders", "get"},
	{"/v1/student/purchase-requests", "get"},
	{"/v1/forum/posts", "post"},
	{"/v1/forum/posts/{postId}", "patch"},
	{"/v1/notifications", "get"},
	{"/v1/admin/notifications/templates", "get"},
	{"/v1/admin/notifications/broadcast", "post"},
	{"/v1/admin/analytics/overview", "get"},
	{"/v1/teacher/analytics", "get"},
	{"/v1/purchase-requests", "post"},
	{"/v1/admin/purchase-requests", "get"},
	{"/v1/admin/purchase-requests/export", "get"},
	{"/v1/admin/purchase-requests/{requestId}/approve", "post"},
	{"/v1/admin/purchase-requests/{requestId}/reject", "post"},
	{"/v1/admin/system/settings", "get"},
	{"/v1/teacher/live-sessions", "post"},
	{"/v1/teacher/live-sessions", "get"},
	{"/v1/student/live-sessions", "get"},
	{"/v1/teacher/live-sessions/{sessionId}", "get"},
	{"/v1/student/live-sessions/{sessionId}", "get"},
	{"/v1/admin/rbac/roles", "get"},
	{"/v1/admin/audit-logs", "get"},
}

// knownPaginatedRoutes is the set of endpoints that accept page+limit query params.
var knownPaginatedRoutes = []routeEntry{
	{"/v1/courses", "get"},
	{"/v1/courses/{courseId}/reviews", "get"},
	{"/v1/bookshop/books", "get"},
	{"/v1/forum/posts", "get"},
	{"/v1/forum/posts/{postId}/comments", "get"},
	{"/v1/student/points/history", "get"},
	{"/v1/leaderboard", "get"},
	{"/v1/student/bookshop/orders", "get"},
	{"/v1/student/purchase-requests", "get"},
	{"/v1/admin/bookshop/books", "get"},
	{"/v1/admin/bookshop/orders", "get"},
	{"/v1/notifications", "get"},
	{"/v1/admin/moderation", "get"},
	{"/v1/admin/audit-logs", "get"},
	{"/v1/admin/purchase-requests", "get"},
	{"/v1/admin/analytics/courses/{courseId}/students", "get"},
	{"/v1/admin/system/settings/history", "get"},
}

// TestProperty1_AllRegisteredRoutesAppearInSpec verifies that every route
// registered in server.go has a corresponding path+method entry in swagger.json.
//
// Feature: swagger-api-documentation, Property 1: All registered routes appear in the spec
// Validates: Requirements 4.1–4.5, 5.1–5.5, 6.1–6.3, 7.1–7.4, 8.1–8.3, 9.1–9.2,
// 10.1–10.3, 11.1–11.3, 12.1–12.2, 13.1–13.2, 14.1–14.4, 15.1, 16.1–16.2, 17.1–17.2, 18.1–18.3
func TestProperty1_AllRegisteredRoutesAppearInSpec(t *testing.T) {
	spec := loadSwaggerSpec(t)

	rapid.Check(t, func(t *rapid.T) {
		route := rapid.SampledFrom(knownRoutes).Draw(t, "route")

		op := getOperation(spec, route.Path, route.Method)
		if op == nil {
			t.Fatalf("route %s %s not found in swagger.json paths", strings.ToUpper(route.Method), route.Path)
		}
	})
}

// TestProperty2_EveryEndpointOperationIsFullyDocumented verifies that every
// operation in swagger.json has a non-empty summary, description, at least one
// tag, produces application/json, at least one 2xx response, and at least one
// 4xx response.
//
// Feature: swagger-api-documentation, Property 2: Every endpoint operation is fully documented
// Validates: Requirements 21.1, 21.5
func TestProperty2_EveryEndpointOperationIsFullyDocumented(t *testing.T) {
	spec := loadSwaggerSpec(t)
	paths := getPaths(spec)

	// Collect all operations as (path, method, operation) triples.
	type opEntry struct {
		Path   string
		Method string
		Op     map[string]interface{}
	}
	var ops []opEntry
	for path, pathItemRaw := range paths {
		pathItem, ok := pathItemRaw.(map[string]interface{})
		if !ok {
			continue
		}
		for _, method := range []string{"get", "post", "put", "patch", "delete"} {
			if opRaw, exists := pathItem[method]; exists {
				if op, ok := opRaw.(map[string]interface{}); ok {
					ops = append(ops, opEntry{Path: path, Method: method, Op: op})
				}
			}
		}
	}

	if len(ops) == 0 {
		t.Fatal("no operations found in swagger.json")
	}

	rapid.Check(t, func(t *rapid.T) {
		entry := rapid.SampledFrom(ops).Draw(t, "operation")
		op := entry.Op
		path := entry.Path
		method := entry.Method

		// Assert non-empty summary
		summary, _ := op["summary"].(string)
		if strings.TrimSpace(summary) == "" {
			t.Fatalf("%s %s: missing or empty summary", strings.ToUpper(method), path)
		}

		// Assert non-empty description
		description, _ := op["description"].(string)
		if strings.TrimSpace(description) == "" {
			t.Fatalf("%s %s: missing or empty description", strings.ToUpper(method), path)
		}

		// Assert at least one tag
		tagsRaw, _ := op["tags"].([]interface{})
		if len(tagsRaw) == 0 {
			t.Fatalf("%s %s: no tags defined", strings.ToUpper(method), path)
		}

		// Assert produces includes application/json
		producesRaw, _ := op["produces"].([]interface{})
		hasJSON := false
		for _, p := range producesRaw {
			if s, ok := p.(string); ok && s == "application/json" {
				hasJSON = true
				break
			}
		}
		if !hasJSON {
			t.Fatalf("%s %s: produces does not include application/json", strings.ToUpper(method), path)
		}

		// Assert at least one success response (2xx or 3xx redirect) and one 4xx response.
		// OAuth redirect endpoints legitimately return 302, so 3xx counts as a success response.
		responsesRaw, _ := op["responses"].(map[string]interface{})
		hasSuccess := false
		has4xx := false
		for code := range responsesRaw {
			if strings.HasPrefix(code, "2") || strings.HasPrefix(code, "3") {
				hasSuccess = true
			}
			if strings.HasPrefix(code, "4") {
				has4xx = true
			}
		}
		if !hasSuccess {
			t.Fatalf("%s %s: no 2xx/3xx success response defined", strings.ToUpper(method), path)
		}
		if !has4xx {
			t.Fatalf("%s %s: no 4xx response defined", strings.ToUpper(method), path)
		}
	})
}

// TestProperty3_AllParametersAreDocumented verifies that for every operation
// whose path contains {param} segments, each segment has a corresponding
// parameter entry with in: path.
//
// Feature: swagger-api-documentation, Property 3: All parameters are documented
// Validates: Requirements 21.2, 21.3
func TestProperty3_AllParametersAreDocumented(t *testing.T) {
	spec := loadSwaggerSpec(t)
	paths := getPaths(spec)

	// Collect operations that have path parameters.
	type opEntry struct {
		Path       string
		Method     string
		Op         map[string]interface{}
		PathParams []string
	}
	var ops []opEntry
	for path, pathItemRaw := range paths {
		// Extract {param} segments from path
		var params []string
		for _, segment := range strings.Split(path, "/") {
			if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
				params = append(params, segment[1:len(segment)-1])
			}
		}
		if len(params) == 0 {
			continue
		}
		pathItem, ok := pathItemRaw.(map[string]interface{})
		if !ok {
			continue
		}
		for _, method := range []string{"get", "post", "put", "patch", "delete"} {
			if opRaw, exists := pathItem[method]; exists {
				if op, ok := opRaw.(map[string]interface{}); ok {
					ops = append(ops, opEntry{Path: path, Method: method, Op: op, PathParams: params})
				}
			}
		}
	}

	if len(ops) == 0 {
		t.Fatal("no operations with path parameters found in swagger.json")
	}

	rapid.Check(t, func(t *rapid.T) {
		entry := rapid.SampledFrom(ops).Draw(t, "operation")
		op := entry.Op
		path := entry.Path
		method := entry.Method

		// Build set of documented in:path parameter names
		documented := make(map[string]bool)
		paramsRaw, _ := op["parameters"].([]interface{})
		for _, pRaw := range paramsRaw {
			p, ok := pRaw.(map[string]interface{})
			if !ok {
				continue
			}
			if inVal, _ := p["in"].(string); inVal == "path" {
				if name, _ := p["name"].(string); name != "" {
					documented[name] = true
				}
			}
		}

		// Assert every {param} in the path is documented
		for _, param := range entry.PathParams {
			if !documented[param] {
				t.Fatalf("%s %s: path parameter {%s} is not documented with in:path", strings.ToUpper(method), path, param)
			}
		}
	})
}

// TestProperty4_AuthenticatedEndpointsDeclareBearer verifies that every
// operation corresponding to an authenticated route declares BearerAuth security.
//
// Feature: swagger-api-documentation, Property 4: Authenticated endpoints declare BearerAuth security
// Validates: Requirements 21.4
func TestProperty4_AuthenticatedEndpointsDeclareBearer(t *testing.T) {
	spec := loadSwaggerSpec(t)

	rapid.Check(t, func(t *rapid.T) {
		route := rapid.SampledFrom(knownAuthRoutes).Draw(t, "auth_route")

		op := getOperation(spec, route.Path, route.Method)
		if op == nil {
			t.Fatalf("authenticated route %s %s not found in swagger.json", strings.ToUpper(route.Method), route.Path)
		}

		// Assert security array contains BearerAuth
		securityRaw, _ := op["security"].([]interface{})
		hasBearerAuth := false
		for _, secEntryRaw := range securityRaw {
			secEntry, ok := secEntryRaw.(map[string]interface{})
			if !ok {
				continue
			}
			if _, exists := secEntry["BearerAuth"]; exists {
				hasBearerAuth = true
				break
			}
		}
		if !hasBearerAuth {
			t.Fatalf("authenticated route %s %s does not declare BearerAuth security", strings.ToUpper(route.Method), route.Path)
		}
	})
}

// refFromResponse extracts the $ref string from a response object, if present.
func refFromResponse(response map[string]interface{}) string {
	schema, ok := response["schema"].(map[string]interface{})
	if !ok {
		return ""
	}
	ref, _ := schema["$ref"].(string)
	return ref
}

// TestProperty5_ErrorResponsesReferenceCorrectSchemas verifies that 400
// responses reference ValidationErrorResponse and 401/403/404 responses
// reference ErrorResponse.
//
// Feature: swagger-api-documentation, Property 5: Error responses reference correct schemas
// Validates: Requirements 19.1–19.5
func TestProperty5_ErrorResponsesReferenceCorrectSchemas(t *testing.T) {
	spec := loadSwaggerSpec(t)
	paths := getPaths(spec)

	// Collect operations that have a 400 response.
	type opEntry struct {
		Path   string
		Method string
		Op     map[string]interface{}
	}
	var ops []opEntry
	for path, pathItemRaw := range paths {
		pathItem, ok := pathItemRaw.(map[string]interface{})
		if !ok {
			continue
		}
		for _, method := range []string{"get", "post", "put", "patch", "delete"} {
			if opRaw, exists := pathItem[method]; exists {
				if op, ok := opRaw.(map[string]interface{}); ok {
					responses, _ := op["responses"].(map[string]interface{})
					if _, has400 := responses["400"]; has400 {
						ops = append(ops, opEntry{Path: path, Method: method, Op: op})
					}
				}
			}
		}
	}

	if len(ops) == 0 {
		t.Fatal("no operations with 400 responses found in swagger.json")
	}

	const (
		validationErrRef = "#/definitions/handlers.ValidationErrorResponse"
		errorRef         = "#/definitions/handlers.ErrorResponse"
	)

	rapid.Check(t, func(t *rapid.T) {
		entry := rapid.SampledFrom(ops).Draw(t, "operation")
		op := entry.Op
		path := entry.Path
		method := entry.Method

		responses, _ := op["responses"].(map[string]interface{})

		// Assert 400 references ValidationErrorResponse
		if resp400Raw, ok := responses["400"]; ok {
			resp400, _ := resp400Raw.(map[string]interface{})
			ref := refFromResponse(resp400)
			if ref != validationErrRef {
				t.Fatalf("%s %s: 400 response references %q, want %q", strings.ToUpper(method), path, ref, validationErrRef)
			}
		}

		// Assert 401 references ErrorResponse
		if resp401Raw, ok := responses["401"]; ok {
			resp401, _ := resp401Raw.(map[string]interface{})
			ref := refFromResponse(resp401)
			if ref != errorRef {
				t.Fatalf("%s %s: 401 response references %q, want %q", strings.ToUpper(method), path, ref, errorRef)
			}
		}

		// Assert 403 references ErrorResponse
		if resp403Raw, ok := responses["403"]; ok {
			resp403, _ := resp403Raw.(map[string]interface{})
			ref := refFromResponse(resp403)
			if ref != errorRef {
				t.Fatalf("%s %s: 403 response references %q, want %q", strings.ToUpper(method), path, ref, errorRef)
			}
		}

		// Assert 404 references ErrorResponse
		if resp404Raw, ok := responses["404"]; ok {
			resp404, _ := resp404Raw.(map[string]interface{})
			ref := refFromResponse(resp404)
			if ref != errorRef {
				t.Fatalf("%s %s: 404 response references %q, want %q", strings.ToUpper(method), path, ref, errorRef)
			}
		}
	})
}

// TestProperty6_PaginatedEndpointsDocumentPaginationConsistently verifies that
// known paginated endpoints document page and limit query parameters with the
// correct types and default values.
//
// Feature: swagger-api-documentation, Property 6: Paginated endpoints document pagination consistently
// Validates: Requirements 20.1, 20.2
func TestProperty6_PaginatedEndpointsDocumentPaginationConsistently(t *testing.T) {
	spec := loadSwaggerSpec(t)

	rapid.Check(t, func(t *rapid.T) {
		route := rapid.SampledFrom(knownPaginatedRoutes).Draw(t, "paginated_route")

		op := getOperation(spec, route.Path, route.Method)
		if op == nil {
			t.Fatalf("paginated route %s %s not found in swagger.json", strings.ToUpper(route.Method), route.Path)
		}

		paramsRaw, _ := op["parameters"].([]interface{})

		var pageParam, limitParam map[string]interface{}
		for _, pRaw := range paramsRaw {
			p, ok := pRaw.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := p["name"].(string)
			switch name {
			case "page":
				pageParam = p
			case "limit":
				limitParam = p
			}
		}

		// Assert page param exists with type integer and default 1
		if pageParam == nil {
			t.Fatalf("%s %s: missing 'page' query parameter", strings.ToUpper(route.Method), route.Path)
		}
		pageType, _ := pageParam["type"].(string)
		if pageType != "integer" {
			t.Fatalf("%s %s: 'page' param type is %q, want \"integer\"", strings.ToUpper(route.Method), route.Path, pageType)
		}
		pageDefault := pageParam["default"]
		// JSON numbers unmarshal as float64
		pageDefaultNum, _ := pageDefault.(float64)
		if pageDefaultNum != 1 {
			t.Fatalf("%s %s: 'page' param default is %v, want 1", strings.ToUpper(route.Method), route.Path, pageDefault)
		}

		// Assert limit param exists with type integer and default 20
		if limitParam == nil {
			t.Fatalf("%s %s: missing 'limit' query parameter", strings.ToUpper(route.Method), route.Path)
		}
		limitType, _ := limitParam["type"].(string)
		if limitType != "integer" {
			t.Fatalf("%s %s: 'limit' param type is %q, want \"integer\"", strings.ToUpper(route.Method), route.Path, limitType)
		}
		limitDefault := limitParam["default"]
		limitDefaultNum, _ := limitDefault.(float64)
		if limitDefaultNum != 20 {
			t.Fatalf("%s %s: 'limit' param default is %v, want 20", strings.ToUpper(route.Method), route.Path, limitDefault)
		}
	})
}
