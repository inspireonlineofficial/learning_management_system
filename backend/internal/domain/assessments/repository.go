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
	// SaveDraftRevision stores the draft only if the attempt is still in
	// progress and revision is newer than the stored one. It returns false
	// when the write was stale or the attempt is no longer in progress.
	SaveDraftRevision(ctx context.Context, attemptID uuid.UUID, draftAnswers []byte, revision int64) (bool, error)
	// Finalize records a submission exactly once: it updates the attempt only
	// while it is still in progress and reports whether this call won.
	Finalize(ctx context.Context, attempt *QuizAttempt) (bool, error)
	// SaveGrading stores manual marks for a submitted attempt.
	SaveGrading(ctx context.Context, attempt *QuizAttempt) error
	// ListByQuiz returns attempts for a quiz joined with student identity.
	ListByQuiz(ctx context.Context, quizID uuid.UUID) ([]*QuizAttemptWithStudent, error)
	// CountByQuizIDs returns attempt and pending-grading counts per quiz.
	CountByQuizIDs(ctx context.Context, quizIDs []uuid.UUID) (map[uuid.UUID]QuizAttemptCounts, error)
}

// QuizAttemptWithStudent is an attempt plus the student's display identity,
// used by teacher monitoring.
type QuizAttemptWithStudent struct {
	QuizAttempt
	StudentName   string
	StudentEmail  string
	AttemptNumber int
}

// QuizAttemptCounts summarises attempts for one quiz.
type QuizAttemptCounts struct {
	Attempts       int
	Submitted      int
	InProgress     int
	PendingGrading int
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
