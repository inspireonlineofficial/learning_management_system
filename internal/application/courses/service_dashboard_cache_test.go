package courses

import (
	"context"
	"errors"
	"testing"
	"time"

	"lms-backend/internal/domain/courses"
	domainenrollments "lms-backend/internal/domain/enrollments"
	tsclient "lms-backend/internal/infrastructure/typesense"

	"github.com/google/uuid"
)

// stubDashboardCache records every InvalidateStudentDashboard call so we
// can assert the courses service triggers it for each enrolled student
// when a course is deleted.
type stubDashboardCache struct {
	calls []uuid.UUID
	err   error
}

func (s *stubDashboardCache) InvalidateStudentDashboard(_ context.Context, studentID uuid.UUID) error {
	s.calls = append(s.calls, studentID)
	return s.err
}

// fakeEnrollmentRepo only needs FindByCourseID for the invalidation path.
// The other methods satisfy the interface but are never called.
type fakeEnrollmentRepo struct {
	enrollments []*domainenrollments.Enrollment
}

func (f *fakeEnrollmentRepo) FindByCourseID(_ context.Context, _ uuid.UUID, _, _ int) ([]*domainenrollments.Enrollment, int, error) {
	return f.enrollments, len(f.enrollments), nil
}
func (f *fakeEnrollmentRepo) Create(context.Context, *domainenrollments.Enrollment) error {
	return nil
}
func (f *fakeEnrollmentRepo) FindByID(context.Context, uuid.UUID) (*domainenrollments.Enrollment, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeEnrollmentRepo) FindByStudentAndCourse(context.Context, uuid.UUID, uuid.UUID) (*domainenrollments.Enrollment, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeEnrollmentRepo) FindByStudentID(context.Context, uuid.UUID, int, int) ([]*domainenrollments.Enrollment, int, error) {
	return nil, 0, nil
}
func (f *fakeEnrollmentRepo) Update(context.Context, *domainenrollments.Enrollment) error {
	return nil
}
func (f *fakeEnrollmentRepo) UpdateProgressPercent(context.Context, uuid.UUID, float64) error {
	return nil
}
func (f *fakeEnrollmentRepo) RecalculateProgressPercent(context.Context, uuid.UUID) error {
	return nil
}
func (f *fakeEnrollmentRepo) CountTotalLessons(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}
func (f *fakeEnrollmentRepo) Exists(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}

// TestAdminDeleteCourseInvalidatesDashboardForEveryStudent verifies that
// when an admin deletes a course, the dashboard cache for each enrolled
// student is invalidated. Without this hook the cached
// /v1/student/dashboard response keeps surfacing the deleted course under
// continue_learning for up to 5 minutes (the cacheTTL set in
// application/analytics/service.go).
func TestAdminDeleteCourseInvalidatesDashboardForEveryStudent(t *testing.T) {
	teacherID := uuid.New()
	courseID := uuid.New()
	studentA := uuid.New()
	studentB := uuid.New()

	courseRepo := newCourseRepoStub(courseID, teacherID)
	indexer := newIndexerStub()

	svc := &service{
		courseRepo: courseRepo,
		enrollmentRepo: &fakeEnrollmentRepo{enrollments: []*domainenrollments.Enrollment{
			{ID: uuid.New(), StudentID: studentA, CourseID: courseID, EnrolledAt: time.Now()},
			{ID: uuid.New(), StudentID: studentB, CourseID: courseID, EnrolledAt: time.Now()},
		}},
		indexer: indexer,
	}
	cache := &stubDashboardCache{}
	svc.SetDashboardCache(cache)

	if err := svc.AdminDeleteCourse(context.Background(), AdminDeleteCourseCommand{
		CourseID: courseID,
		AdminID:  uuid.New(),
	}); err != nil {
		t.Fatalf("AdminDeleteCourse returned error: %v", err)
	}

	if len(cache.calls) != 2 {
		t.Fatalf("expected 2 invalidations (one per enrolled student), got %d", len(cache.calls))
	}
	got := map[uuid.UUID]bool{cache.calls[0]: true, cache.calls[1]: true}
	if !got[studentA] || !got[studentB] {
		t.Errorf("expected invalidations for both enrolled students, got %v", cache.calls)
	}
	if courseRepo.softDeleteCalls != 1 {
		t.Errorf("expected SoftDelete to be called once, got %d", courseRepo.softDeleteCalls)
	}
	if !indexer.deleteCalled {
		t.Errorf("expected indexer.DeleteCourse to be called")
	}
}

// TestAdminDeleteCourseNoDashboardCache is a regression guard: when no
// dashboard cache is wired (e.g. tests or a bootstrap environment), the
// delete still succeeds and does not panic.
func TestAdminDeleteCourseNoDashboardCache(t *testing.T) {
	teacherID := uuid.New()
	courseID := uuid.New()

	courseRepo := newCourseRepoStub(courseID, teacherID)
	svc := &service{
		courseRepo:     courseRepo,
		enrollmentRepo: &fakeEnrollmentRepo{},
		indexer:        newIndexerStub(),
	}

	if err := svc.AdminDeleteCourse(context.Background(), AdminDeleteCourseCommand{
		CourseID: courseID,
		AdminID:  uuid.New(),
	}); err != nil {
		t.Fatalf("AdminDeleteCourse returned error: %v", err)
	}
}

// ─── Tiny stubs ────────────────────────────────────────────────────────────

type courseRepoStub struct {
	course          *courses.Course
	softDeleteCalls int
}

func newCourseRepoStub(id, teacherID uuid.UUID) *courseRepoStub {
	return &courseRepoStub{course: &courses.Course{ID: id, TeacherID: teacherID, Status: courses.CourseStatusPublished}}
}

func (r *courseRepoStub) FindByID(_ context.Context, id uuid.UUID) (*courses.Course, error) {
	if id == r.course.ID {
		return r.course, nil
	}
	return nil, errors.New("course not found")
}
func (r *courseRepoStub) SoftDelete(_ context.Context, _ uuid.UUID) error {
	r.softDeleteCalls++
	return nil
}
func (r *courseRepoStub) Create(context.Context, *courses.Course) error { return nil }
func (r *courseRepoStub) FindBySlug(context.Context, string) (*courses.Course, error) {
	return nil, nil
}
func (r *courseRepoStub) FindByTeacherID(context.Context, uuid.UUID, int, int) ([]*courses.Course, int, error) {
	return nil, 0, nil
}
func (r *courseRepoStub) Update(context.Context, *courses.Course) error { return nil }
func (r *courseRepoStub) List(context.Context, courses.CourseFilters, int, int) ([]*courses.Course, int, error) {
	return nil, 0, nil
}
func (r *courseRepoStub) CountPublishedLessons(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}

type indexerStub struct {
	deleteCalled bool
}

func newIndexerStub() *indexerStub { return &indexerStub{} }

func (i *indexerStub) DeleteCourse(_ context.Context, _ string) error {
	i.deleteCalled = true
	return nil
}
func (i *indexerStub) UpsertCourse(context.Context, tsclient.CourseDocument) error { return nil }
func (i *indexerStub) DeleteLesson(context.Context, string) error                  { return nil }
func (i *indexerStub) UpsertLesson(context.Context, tsclient.LessonDocument) error { return nil }
func (i *indexerStub) DeleteForumPost(context.Context, string) error               { return nil }
func (i *indexerStub) UpsertForumPost(context.Context, tsclient.ForumPostDocument) error {
	return nil
}
func (i *indexerStub) DeleteBook(context.Context, string) error { return nil }
func (i *indexerStub) UpsertBook(context.Context, tsclient.BookDocument) error {
	return nil
}
