package search

import (
	"context"
	"database/sql"
	"fmt"

	"lms-backend/pkg/apperrors"

	tsclient "lms-backend/internal/infrastructure/typesense"

	"github.com/google/uuid"
	"github.com/typesense/typesense-go/typesense/api"
)

// SearchCommand defines the global search request. Requirements: 5.1–5.12
type SearchCommand struct {
	Query  string     // min 2 chars
	Type   string     // optional: courses, lessons, forum, books
	Limit  int        // per-type limit, default 10, max 30
	UserID *uuid.UUID // optional: authenticated user for enrollment-aware metadata
}

// CourseResult is a search result for a course.
type CourseResult struct {
	ID               uuid.UUID `json:"id"`
	Title            string    `json:"title"`
	Slug             string    `json:"slug"`
	ShortDescription string    `json:"short_description"`
	Subject          string    `json:"subject"`
	Level            string    `json:"level"`
	IsEnrolled       bool      `json:"is_enrolled,omitempty"`
}

// LessonResult is a search result for a lesson.
type LessonResult struct {
	ID            uuid.UUID `json:"id"`
	Title         string    `json:"title"`
	CourseID      uuid.UUID `json:"course_id"`
	CourseTitle   string    `json:"course_title"`
	IsFreePreview bool      `json:"is_free_preview"`
	IsEnrolled    bool      `json:"is_enrolled,omitempty"`
}

// ForumResult is a search result for a forum post.
type ForumResult struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
	Body  string    `json:"body_excerpt"`
}

// BookResult is a search result for a book.
type BookResult struct {
	ID     uuid.UUID `json:"id"`
	Title  string    `json:"title"`
	Author string    `json:"author"`
	Format string    `json:"format"`
}

// SearchResponse groups results by type. Requirements: 5.3
type SearchResponse struct {
	Courses []CourseResult `json:"courses"`
	Lessons []LessonResult `json:"lessons"`
	Forum   []ForumResult  `json:"forum"`
	Books   []BookResult   `json:"books"`
}

// Service defines the global search use case.
type Service interface {
	Search(ctx context.Context, cmd SearchCommand) (*SearchResponse, error)
}

type service struct {
	searcher     tsclient.Searcher
	enrollmentDB *sql.DB
}

// NewService creates a new search service backed by Typesense.
// Requirements: 5.1
func NewService(searcher tsclient.Searcher, enrollmentDB *sql.DB) Service {
	return &service{searcher: searcher, enrollmentDB: enrollmentDB}
}

// Search performs a global full-text search across courses, lessons, forum posts, and books.
// Requirements: 5.1–5.12
func (s *service) Search(ctx context.Context, cmd SearchCommand) (*SearchResponse, error) {
	// Requirements: 5.2
	if len(cmd.Query) < 2 {
		return nil, apperrors.NewSimpleValidationError("VALIDATION_ERROR", "query must be at least 2 characters")
	}

	// Requirements: 5.5, 5.6
	limit := cmd.Limit
	if limit < 1 {
		limit = 10
	}
	if limit > 30 {
		limit = 30
	}

	resp := &SearchResponse{
		Courses: []CourseResult{},
		Lessons: []LessonResult{},
		Forum:   []ForumResult{},
		Books:   []BookResult{},
	}

	filterType := cmd.Type

	// Requirements: 5.3, 5.4, 5.7
	if filterType == "" || filterType == "courses" {
		courses, err := s.searchCourses(ctx, cmd.Query, limit, cmd.UserID)
		if err != nil {
			return nil, fmt.Errorf("search courses: %w", err)
		}
		resp.Courses = courses
	}

	// Requirements: 5.3, 5.4, 5.8
	if filterType == "" || filterType == "lessons" {
		lessons, err := s.searchLessons(ctx, cmd.Query, limit, cmd.UserID)
		if err != nil {
			return nil, fmt.Errorf("search lessons: %w", err)
		}
		resp.Lessons = lessons
	}

	// Requirements: 5.3, 5.4, 5.9
	if filterType == "" || filterType == "forum" {
		posts, err := s.searchForum(ctx, cmd.Query, limit)
		if err != nil {
			return nil, fmt.Errorf("search forum: %w", err)
		}
		resp.Forum = posts
	}

	// Requirements: 5.3, 5.4, 5.10
	if filterType == "" || filterType == "books" {
		books, err := s.searchBooks(ctx, cmd.Query, limit)
		if err != nil {
			return nil, fmt.Errorf("search books: %w", err)
		}
		resp.Books = books
	}

	return resp, nil
}

func (s *service) searchCourses(ctx context.Context, query string, limit int, userID *uuid.UUID) ([]CourseResult, error) {
	filterBy := "status:=published"
	params := &api.SearchCollectionParams{
		Q:        query,
		QueryBy:  "title,short_description",
		FilterBy: &filterBy,
		PerPage:  &limit,
	}

	result, err := s.searcher.SearchCollection(ctx, "courses", params)
	if err != nil {
		return nil, err
	}

	var results []CourseResult
	if result.Hits != nil {
		for _, hit := range *result.Hits {
			if hit.Document == nil {
				continue
			}
			doc := *hit.Document
			r := CourseResult{
				Title:            docString(doc, "title"),
				Slug:             docString(doc, "slug"),
				ShortDescription: docString(doc, "short_description"),
				Subject:          docString(doc, "subject"),
				Level:            docString(doc, "level"),
			}
			if id, err := uuid.Parse(docString(doc, "id")); err == nil {
				r.ID = id
			}
			results = append(results, r)
		}
	}

	// Requirements: 5.11
	if userID != nil && len(results) > 0 {
		enrolledSet, err := s.getEnrolledCourseIDs(ctx, *userID)
		if err == nil {
			for i := range results {
				results[i].IsEnrolled = enrolledSet[results[i].ID]
			}
		}
	}

	if results == nil {
		results = []CourseResult{}
	}
	return results, nil
}

func (s *service) searchLessons(ctx context.Context, query string, limit int, userID *uuid.UUID) ([]LessonResult, error) {
	filterBy := "status:=published"
	params := &api.SearchCollectionParams{
		Q:        query,
		QueryBy:  "title",
		FilterBy: &filterBy,
		PerPage:  &limit,
	}

	result, err := s.searcher.SearchCollection(ctx, "lessons", params)
	if err != nil {
		return nil, err
	}

	var results []LessonResult
	if result.Hits != nil {
		for _, hit := range *result.Hits {
			if hit.Document == nil {
				continue
			}
			doc := *hit.Document
			r := LessonResult{
				Title:         docString(doc, "title"),
				CourseTitle:   docString(doc, "course_title"),
				IsFreePreview: docBool(doc, "is_free_preview"),
			}
			if id, err := uuid.Parse(docString(doc, "id")); err == nil {
				r.ID = id
			}
			if cid, err := uuid.Parse(docString(doc, "course_id")); err == nil {
				r.CourseID = cid
			}
			results = append(results, r)
		}
	}

	// Requirements: 5.11
	if userID != nil && len(results) > 0 {
		enrolledSet, err := s.getEnrolledCourseIDs(ctx, *userID)
		if err == nil {
			for i := range results {
				results[i].IsEnrolled = enrolledSet[results[i].CourseID]
			}
		}
	}

	if results == nil {
		results = []LessonResult{}
	}
	return results, nil
}

func (s *service) searchForum(ctx context.Context, query string, limit int) ([]ForumResult, error) {
	filterBy := "status:=active"
	params := &api.SearchCollectionParams{
		Q:        query,
		QueryBy:  "title,body_excerpt",
		FilterBy: &filterBy,
		PerPage:  &limit,
	}

	result, err := s.searcher.SearchCollection(ctx, "forum_posts", params)
	if err != nil {
		return nil, err
	}

	var results []ForumResult
	if result.Hits != nil {
		for _, hit := range *result.Hits {
			if hit.Document == nil {
				continue
			}
			doc := *hit.Document
			r := ForumResult{
				Title: docString(doc, "title"),
				Body:  docString(doc, "body_excerpt"),
			}
			if id, err := uuid.Parse(docString(doc, "id")); err == nil {
				r.ID = id
			}
			results = append(results, r)
		}
	}

	if results == nil {
		results = []ForumResult{}
	}
	return results, nil
}

func (s *service) searchBooks(ctx context.Context, query string, limit int) ([]BookResult, error) {
	filterBy := "is_active:=true"
	params := &api.SearchCollectionParams{
		Q:        query,
		QueryBy:  "title,author",
		FilterBy: &filterBy,
		PerPage:  &limit,
	}

	result, err := s.searcher.SearchCollection(ctx, "books", params)
	if err != nil {
		return nil, err
	}

	var results []BookResult
	if result.Hits != nil {
		for _, hit := range *result.Hits {
			if hit.Document == nil {
				continue
			}
			doc := *hit.Document
			r := BookResult{
				Title:  docString(doc, "title"),
				Author: docString(doc, "author"),
				Format: docString(doc, "format"),
			}
			if id, err := uuid.Parse(docString(doc, "id")); err == nil {
				r.ID = id
			}
			results = append(results, r)
		}
	}

	if results == nil {
		results = []BookResult{}
	}
	return results, nil
}

// getEnrolledCourseIDs returns the set of course IDs the user is actively enrolled in.
// Requirements: 5.11
func (s *service) getEnrolledCourseIDs(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]bool, error) {
	rows, err := s.enrollmentDB.QueryContext(ctx,
		`SELECT course_id FROM enrollments WHERE student_id = $1 AND status = 'active'`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	set := make(map[uuid.UUID]bool)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		set[id] = true
	}
	return set, rows.Err()
}

// docString safely extracts a string value from a Typesense document map.
func docString(doc map[string]interface{}, key string) string {
	if v, ok := doc[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// docBool safely extracts a bool value from a Typesense document map.
func docBool(doc map[string]interface{}, key string) bool {
	if v, ok := doc[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
