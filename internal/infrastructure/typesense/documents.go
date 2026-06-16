package typesense

// CourseDocument maps to the "courses" collection.
type CourseDocument struct {
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	Slug             string  `json:"slug"`
	ShortDescription string  `json:"short_description"`
	Subject          string  `json:"subject"`
	Level            string  `json:"level"`
	Status           string  `json:"status"`
	RatingAverage    float32 `json:"rating_average"`
}

// LessonDocument maps to the "lessons" collection.
type LessonDocument struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	CourseID      string `json:"course_id"`
	CourseTitle   string `json:"course_title"`
	IsFreePreview bool   `json:"is_free_preview"`
	Status        string `json:"status"`
}

// ForumPostDocument maps to the "forum_posts" collection.
type ForumPostDocument struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	BodyExcerpt string `json:"body_excerpt"` // first 200 chars of body_markdown
	Status      string `json:"status"`
	CreatedAt   int64  `json:"created_at"` // Unix timestamp
}

// BookDocument maps to the "books" collection.
type BookDocument struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Author   string `json:"author"`
	Format   string `json:"format"`
	IsActive bool   `json:"is_active"`
}
