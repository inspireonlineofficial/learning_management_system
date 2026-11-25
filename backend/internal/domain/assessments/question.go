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
	QuestionTypeShort     QuestionType = "short_answer"
	QuestionTypeNumeric   QuestionType = "numeric"     // auto-graded with tolerance
	QuestionTypeLong      QuestionType = "long_answer" // graded manually by the teacher
)

// UsesOptions reports whether answers are chosen from options.
func (t QuestionType) UsesOptions() bool {
	return t == QuestionTypeSingle || t == QuestionTypeMultiple || t == QuestionTypeTrueFalse
}

// IsManuallyGraded reports whether a teacher must award the marks.
func (t QuestionType) IsManuallyGraded() bool { return t == QuestionTypeLong }

// QuestionSettings holds per-type configuration stored as JSON.
type QuestionSettings struct {
	// Numeric questions.
	NumericAnswer *float64 `json:"numeric_answer,omitempty"`
	Tolerance     float64  `json:"tolerance,omitempty"`      // >= 0
	ToleranceType string   `json:"tolerance_type,omitempty"` // "absolute" (default) | "percent"
	Unit          string   `json:"unit,omitempty"`           // shown next to the input; optional in answers
	// Long answers.
	MaxWords int `json:"max_words,omitempty"`
	// Short answers.
	CaseSensitive bool `json:"case_sensitive,omitempty"`
}

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
	Hint        string // shown to students only in practice mode
	Settings    QuestionSettings
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
