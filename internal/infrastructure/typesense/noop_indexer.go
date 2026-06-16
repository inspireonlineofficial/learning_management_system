package typesense

import "context"

// NoopIndexer implements Indexer with no-op methods for use in services not yet
// wired to a real Typesense client. All methods return nil.
type NoopIndexer struct{}

func (n *NoopIndexer) UpsertCourse(_ context.Context, _ CourseDocument) error       { return nil }
func (n *NoopIndexer) DeleteCourse(_ context.Context, _ string) error               { return nil }
func (n *NoopIndexer) UpsertLesson(_ context.Context, _ LessonDocument) error       { return nil }
func (n *NoopIndexer) DeleteLesson(_ context.Context, _ string) error               { return nil }
func (n *NoopIndexer) UpsertForumPost(_ context.Context, _ ForumPostDocument) error { return nil }
func (n *NoopIndexer) DeleteForumPost(_ context.Context, _ string) error            { return nil }
func (n *NoopIndexer) UpsertBook(_ context.Context, _ BookDocument) error           { return nil }
func (n *NoopIndexer) DeleteBook(_ context.Context, _ string) error                 { return nil }
