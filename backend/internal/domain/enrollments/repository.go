package enrollments

import (
	"context"

	"github.com/google/uuid"
)

// EnrollmentRepository defines the interface for enrollment persistence
type EnrollmentRepository interface {
	Create(ctx context.Context, enrollment *Enrollment) error
	FindByID(ctx context.Context, id uuid.UUID) (*Enrollment, error)
	FindByStudentAndCourse(ctx context.Context, studentID, courseID uuid.UUID) (*Enrollment, error)
	FindByStudentID(ctx context.Context, studentID uuid.UUID, page, limit int) ([]*Enrollment, int, error)
	FindByCourseID(ctx context.Context, courseID uuid.UUID, page, limit int) ([]*Enrollment, int, error)
	Update(ctx context.Context, enrollment *Enrollment) error
	UpdateProgressPercent(ctx context.Context, enrollmentID uuid.UUID, progressPercent float64) error
	RecalculateProgressPercent(ctx context.Context, enrollmentID uuid.UUID) error
	CountTotalLessons(ctx context.Context, courseID uuid.UUID) (int, error)
	Exists(ctx context.Context, studentID, courseID uuid.UUID) (bool, error)
}

// LessonProgressRepository defines the interface for lesson progress persistence
type LessonProgressRepository interface {
	Upsert(ctx context.Context, progress *LessonProgress) error
	FindByID(ctx context.Context, id uuid.UUID) (*LessonProgress, error)
	FindByEnrollmentAndLesson(ctx context.Context, enrollmentID, lessonID uuid.UUID) (*LessonProgress, error)
	FindByEnrollmentID(ctx context.Context, enrollmentID uuid.UUID) ([]*LessonProgress, error)
	CountCompletedLessons(ctx context.Context, enrollmentID uuid.UUID) (int, error)
}
