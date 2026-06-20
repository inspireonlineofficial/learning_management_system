package courses

import (
	"time"

	"github.com/google/uuid"
)

// CourseStatus represents the lifecycle state of a course
type CourseStatus string

const (
	CourseStatusDraft     CourseStatus = "draft"
	CourseStatusPending   CourseStatus = "pending"
	CourseStatusPublished CourseStatus = "published"
	CourseStatusRejected  CourseStatus = "rejected"
)

// CourseLevel represents the difficulty level
type CourseLevel string

const (
	CourseLevelBeginner     CourseLevel = "beginner"
	CourseLevelIntermediate CourseLevel = "intermediate"
	CourseLevelAdvanced     CourseLevel = "advanced"
)

// PriceType represents whether a course is free or paid
type PriceType string

const (
	PriceTypeFree PriceType = "free"
	PriceTypePaid PriceType = "paid"
)

// Course is the aggregate root for the courses bounded context
type Course struct {
	ID                       uuid.UUID
	TeacherID                uuid.UUID
	Title                    string
	Slug                     string
	ShortDescription         string
	Description              string
	Subject                  string
	Level                    CourseLevel
	PriceType                PriceType
	Price                    float64
	Currency                 string
	Prerequisites            string
	Visibility               string
	LearningOutcomes         string
	Requirements             string
	TargetAudience           string
	EstimatedDurationMinutes int
	ThumbnailURL             string
	Status                   CourseStatus
	RatingAverage            float64
	RatingCount              int
	TotalEnrolled            int
	PublishedAt              *time.Time
	CreatedAt                time.Time
	UpdatedAt                time.Time
	DeletedAt                *time.Time
}

// CourseStatusTransition defines valid state transitions
type CourseStatusTransition struct {
	From CourseStatus
	To   CourseStatus
}

// ValidTransitions defines the allowed state machine transitions
var ValidTransitions = map[CourseStatusTransition]bool{
	{CourseStatusDraft, CourseStatusPending}:     true,
	{CourseStatusRejected, CourseStatusPending}:  true,
	{CourseStatusPending, CourseStatusPublished}: true,
	{CourseStatusPending, CourseStatusRejected}:  true,
}

// CanTransitionTo checks if a status transition is valid
func (c *Course) CanTransitionTo(newStatus CourseStatus) bool {
	return ValidTransitions[CourseStatusTransition{From: c.Status, To: newStatus}]
}

// IsOwnedBy checks if the course belongs to the given teacher
func (c *Course) IsOwnedBy(teacherID uuid.UUID) bool {
	return c.TeacherID == teacherID
}

// IsEditable returns true if the course can be edited
func (c *Course) IsEditable() bool {
	return c.Status != CourseStatusPending
}

// HasPublishedLessons checks if the course has at least one published lesson
func (c *Course) HasPublishedLessons(lessonCount int) bool {
	return lessonCount > 0
}
