package assessments

import (
	"time"

	"github.com/google/uuid"
)

// QuestionType represents the type of question
type QuestionType string

const (
	QuestionTypeSingle    QuestionType = "single"
	QuestionTypeMultiple  QuestionType = "multiple"
	QuestionTypeTrueFalse QuestionType = "true_false"
)

// Question represents a quiz question entity
type Question struct {
	ID          uuid.UUID
	QuizID      uuid.UUID
	Body        string
	Type        QuestionType
	ContentType string
	ImageURL    string
	Marks       float64
	IsRequired  bool
	Position    int
	Explanation string // teacher/admin only - NEVER in student response
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// QuestionOption represents an answer option for a question
type QuestionOption struct {
	ID          uuid.UUID
	QuestionID  uuid.UUID
	Body        string
	ContentType string
	ImageURL    string
	IsCorrect   bool // NEVER in student response
	Position    int
}

// QuestionStudentView is the student-facing view of a question
// It excludes all answer key fields (explanation, correct_option_ids)
type QuestionStudentView struct {
	ID          uuid.UUID
	QuizID      uuid.UUID
	Body        string
	Type        QuestionType
	ContentType string
	ImageURL    string
	Marks       float64
	IsRequired  bool
	Position    int
	Options     []QuestionOptionStudentView
}

// QuestionOptionStudentView is the student-facing view of a question option
// It excludes the IsCorrect field
type QuestionOptionStudentView struct {
	ID          uuid.UUID
	Body        string
	ContentType string
	ImageURL    string
	Position    int
}

// QuestionTeacherView is the teacher/admin-facing view of a question
// It includes all fields including answer keys
type QuestionTeacherView struct {
	ID               uuid.UUID
	QuizID           uuid.UUID
	Body             string
	Type             QuestionType
	ContentType      string
	ImageURL         string
	Marks            float64
	IsRequired       bool
	Position         int
	Explanation      string
	CorrectOptionIDs []uuid.UUID
	Options          []QuestionOption
}

// ToStudentView converts a Question to a student-safe view
func (q *Question) ToStudentView(options []QuestionOption) QuestionStudentView {
	studentOptions := make([]QuestionOptionStudentView, len(options))
	for i, opt := range options {
		studentOptions[i] = QuestionOptionStudentView{
			ID:          opt.ID,
			Body:        opt.Body,
			ContentType: opt.ContentType,
			ImageURL:    opt.ImageURL,
			Position:    opt.Position,
		}
	}

	return QuestionStudentView{
		ID:          q.ID,
		QuizID:      q.QuizID,
		Body:        q.Body,
		Type:        q.Type,
		ContentType: q.ContentType,
		ImageURL:    q.ImageURL,
		Marks:       q.Marks,
		IsRequired:  q.IsRequired,
		Position:    q.Position,
		Options:     studentOptions,
	}
}

// ToTeacherView converts a Question to a teacher view with answer keys
func (q *Question) ToTeacherView(options []QuestionOption) QuestionTeacherView {
	correctIDs := make([]uuid.UUID, 0)
	for _, opt := range options {
		if opt.IsCorrect {
			correctIDs = append(correctIDs, opt.ID)
		}
	}

	return QuestionTeacherView{
		ID:               q.ID,
		QuizID:           q.QuizID,
		Body:             q.Body,
		Type:             q.Type,
		ContentType:      q.ContentType,
		ImageURL:         q.ImageURL,
		Marks:            q.Marks,
		IsRequired:       q.IsRequired,
		Position:         q.Position,
		Explanation:      q.Explanation,
		CorrectOptionIDs: correctIDs,
		Options:          options,
	}
}
