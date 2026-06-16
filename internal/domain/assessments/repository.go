package assessments

import (
	"context"

	"github.com/google/uuid"
)

// QuizRepository defines the interface for quiz persistence
type QuizRepository interface {
	Create(ctx context.Context, quiz *Quiz) error
	FindByID(ctx context.Context, id uuid.UUID) (*Quiz, error)
	FindByCourseID(ctx context.Context, courseID uuid.UUID) ([]*Quiz, error)
	FindByLessonID(ctx context.Context, lessonID uuid.UUID) ([]*Quiz, error)
	Update(ctx context.Context, quiz *Quiz) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// QuestionRepository defines the interface for question persistence
type QuestionRepository interface {
	Create(ctx context.Context, question *Question) error
	CreateBatch(ctx context.Context, questions []*Question) error
	FindByID(ctx context.Context, id uuid.UUID) (*Question, error)
	FindByQuizID(ctx context.Context, quizID uuid.UUID) ([]*Question, error)
	Update(ctx context.Context, question *Question) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// QuestionOptionRepository defines the interface for question option persistence
type QuestionOptionRepository interface {
	Create(ctx context.Context, option *QuestionOption) error
	CreateBatch(ctx context.Context, options []*QuestionOption) error
	FindByQuestionID(ctx context.Context, questionID uuid.UUID) ([]*QuestionOption, error)
	FindByQuestionIDs(ctx context.Context, questionIDs []uuid.UUID) (map[uuid.UUID][]*QuestionOption, error)
	Update(ctx context.Context, option *QuestionOption) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// QuizAttemptRepository defines the interface for quiz attempt persistence
type QuizAttemptRepository interface {
	Create(ctx context.Context, attempt *QuizAttempt) error
	FindByID(ctx context.Context, id uuid.UUID) (*QuizAttempt, error)
	FindByQuizAndStudent(ctx context.Context, quizID, studentID uuid.UUID) ([]*QuizAttempt, error)
	CountAttempts(ctx context.Context, quizID, studentID uuid.UUID) (int, error)
	GetHighestScore(ctx context.Context, quizID, studentID uuid.UUID) (*float64, error)
	Update(ctx context.Context, attempt *QuizAttempt) error
	SaveDraftAnswers(ctx context.Context, attemptID uuid.UUID, draftAnswers []byte) error
	FindInProgressAttempts(ctx context.Context) ([]*QuizAttempt, error)
}

// AssignmentRepository defines the interface for assignment persistence
type AssignmentRepository interface {
	Create(ctx context.Context, assignment *Assignment) error
	FindByID(ctx context.Context, id uuid.UUID) (*Assignment, error)
	FindByCourseID(ctx context.Context, courseID uuid.UUID) ([]*Assignment, error)
	Update(ctx context.Context, assignment *Assignment) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// AssignmentSubmissionRepository defines the interface for assignment submission persistence
type AssignmentSubmissionRepository interface {
	Create(ctx context.Context, submission *AssignmentSubmission) error
	FindByID(ctx context.Context, id uuid.UUID) (*AssignmentSubmission, error)
	FindByAssignmentAndStudent(ctx context.Context, assignmentID, studentID uuid.UUID) (*AssignmentSubmission, error)
	FindByAssignmentID(ctx context.Context, assignmentID uuid.UUID, page, limit int) ([]*AssignmentSubmission, int, error)
	FindDraftByAssignmentAndStudent(ctx context.Context, assignmentID, studentID uuid.UUID) (*AssignmentSubmission, error)
	Update(ctx context.Context, submission *AssignmentSubmission) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SubmissionFileRepository defines the interface for submission file persistence
type SubmissionFileRepository interface {
	Create(ctx context.Context, file *SubmissionFile) error
	CreateBatch(ctx context.Context, files []*SubmissionFile) error
	FindBySubmissionID(ctx context.Context, submissionID uuid.UUID) ([]*SubmissionFile, error)
	CountBySubmissionID(ctx context.Context, submissionID uuid.UUID) (int, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// SubmissionGradeRepository defines the interface for submission grade persistence
// Note: This is an append-only repository - no Update or Delete methods
type SubmissionGradeRepository interface {
	Create(ctx context.Context, grade *SubmissionGrade) error
	FindByID(ctx context.Context, id uuid.UUID) (*SubmissionGrade, error)
	FindBySubmissionID(ctx context.Context, submissionID uuid.UUID) ([]*SubmissionGrade, error)
	GetLatestGrade(ctx context.Context, submissionID uuid.UUID) (*SubmissionGrade, error)
}
