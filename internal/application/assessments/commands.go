package assessments

import "github.com/google/uuid"

// CreateQuizCommand represents the command to create a new quiz
type CreateQuizCommand struct {
	CourseID                   uuid.UUID
	TeacherID                  uuid.UUID
	LessonID                   *uuid.UUID // nullable - can be module-level
	Title                      string
	TimeLimitSeconds           int
	MaxAttempts                int
	PassingScorePercent        float64
	ShuffleQuestions           bool
	ShowAnswersAfterSubmission bool
	Questions                  []CreateQuestionCommand
}

// UpdateQuizCommand replaces editable quiz metadata and question contents.
type UpdateQuizCommand struct {
	QuizID                     uuid.UUID
	TeacherID                  uuid.UUID
	LessonID                   *uuid.UUID
	Title                      string
	TimeLimitSeconds           int
	MaxAttempts                int
	PassingScorePercent        float64
	ShuffleQuestions           bool
	ShowAnswersAfterSubmission bool
	Questions                  []CreateQuestionCommand
}

// CreateQuestionCommand represents a question to be created with a quiz
type CreateQuestionCommand struct {
	Body        string
	Type        string // "single", "multiple", "true_false"
	Position    int
	Explanation string
	Options     []CreateQuestionOptionCommand
}

// CreateStandaloneQuestionCommand creates a question for an existing quiz.
type CreateStandaloneQuestionCommand struct {
	QuizID    uuid.UUID
	TeacherID uuid.UUID
	Question  CreateQuestionCommand
}

// UpdateQuestionCommand replaces a question and its answer options.
type UpdateQuestionCommand struct {
	QuestionID uuid.UUID
	TeacherID  uuid.UUID
	Question   CreateQuestionCommand
}

// CreateQuestionOptionCommand represents an option for a question
type CreateQuestionOptionCommand struct {
	Body      string
	IsCorrect bool
	Position  int
}

// StartAttemptCommand represents the command to start a quiz attempt
type StartAttemptCommand struct {
	QuizID    uuid.UUID
	StudentID uuid.UUID
}

// SubmitAttemptCommand represents the command to submit a quiz attempt
type SubmitAttemptCommand struct {
	AttemptID uuid.UUID
	StudentID uuid.UUID
	Answers   []QuizAnswerCommand
}

// QuizAnswerCommand represents a student's answer to a question
type QuizAnswerCommand struct {
	QuestionID      uuid.UUID
	SelectedOptions []uuid.UUID
}

// CreateAssignmentCommand represents the command to create a new assignment
type CreateAssignmentCommand struct {
	CourseID            uuid.UUID
	TeacherID           uuid.UUID
	Title               string
	Description         string
	DueAt               string // ISO 8601 format
	SubmissionType      string // "file", "text", "both"
	MaxFileSizeMB       int
	AllowLateSubmission bool
	TotalMarks          float64
}

// UpdateAssignmentCommand replaces editable assignment metadata.
type UpdateAssignmentCommand struct {
	AssignmentID        uuid.UUID
	TeacherID           uuid.UUID
	Title               string
	Description         string
	DueAt               string
	SubmissionType      string
	MaxFileSizeMB       int
	AllowLateSubmission bool
	TotalMarks          float64
}

// SubmitAssignmentCommand represents the command to submit an assignment
type SubmitAssignmentCommand struct {
	AssignmentID uuid.UUID
	StudentID    uuid.UUID
	TextContent  string
	Files        []SubmissionFileCommand
	IsDraft      bool
}

// SubmissionFileCommand represents a file to be uploaded with a submission
type SubmissionFileCommand struct {
	OriginalFilename string
	MimeType         string
	SizeBytes        int64
	Content          []byte
}

// GradeSubmissionCommand represents the command to grade a submission
type GradeSubmissionCommand struct {
	SubmissionID      uuid.UUID
	GradedBy          uuid.UUID
	Score             float64
	Feedback          string
	RevisionRequested bool
	RevisionNotes     string
}
