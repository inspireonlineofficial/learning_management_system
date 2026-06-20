package assessments

import (
	"time"

	"github.com/google/uuid"
)

// QuizResponse represents a quiz in API responses
type QuizResponse struct {
	ID                         uuid.UUID  `json:"id"`
	CourseID                   uuid.UUID  `json:"course_id"`
	LessonID                   *uuid.UUID `json:"lesson_id"`
	Title                      string     `json:"title"`
	TimeLimitSeconds           int        `json:"time_limit_seconds"`
	MaxAttempts                int        `json:"max_attempts"`
	PassingScorePercent        float64    `json:"passing_score_percent"`
	ShuffleQuestions           bool       `json:"shuffle_questions"`
	ShowAnswersAfterSubmission bool       `json:"show_answers_after_submission"`
	CreatedAt                  time.Time  `json:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`
}

// QuizTeacherResponse represents a quiz with questions for teacher view
type QuizTeacherResponse struct {
	QuizResponse
	Questions []QuestionTeacherResponse `json:"questions"`
}

// QuestionTeacherResponse represents a question with answer keys for teacher view
type QuestionTeacherResponse struct {
	ID               uuid.UUID                       `json:"id"`
	QuizID           uuid.UUID                       `json:"quiz_id"`
	Body             string                          `json:"body"`
	Type             string                          `json:"type"`
	ContentType      string                          `json:"content_type"`
	ImageURL         string                          `json:"image_url"`
	Marks            float64                         `json:"marks"`
	IsRequired       bool                            `json:"is_required"`
	Position         int                             `json:"position"`
	Explanation      string                          `json:"explanation"`
	CorrectOptionIDs []uuid.UUID                     `json:"correct_option_ids"`
	Options          []QuestionOptionTeacherResponse `json:"options"`
}

// QuestionOptionTeacherResponse represents a question option for teacher view
type QuestionOptionTeacherResponse struct {
	ID          uuid.UUID `json:"id"`
	Body        string    `json:"body"`
	ContentType string    `json:"content_type"`
	ImageURL    string    `json:"image_url"`
	IsCorrect   bool      `json:"is_correct"`
	Position    int       `json:"position"`
}

// QuizAttemptResponse represents a quiz attempt in API responses
type QuizAttemptResponse struct {
	ID        uuid.UUID                 `json:"id"`
	QuizID    uuid.UUID                 `json:"quiz_id"`
	StartedAt time.Time                 `json:"started_at"`
	ExpiresAt *time.Time                `json:"expires_at,omitempty"`
	Questions []QuestionStudentResponse `json:"questions"`
	Answers   map[string]interface{}    `json:"answers,omitempty"`
}

// QuestionStudentResponse represents a question for student view (no answer keys)
type QuestionStudentResponse struct {
	ID          uuid.UUID                       `json:"id"`
	QuizID      uuid.UUID                       `json:"quiz_id"`
	Body        string                          `json:"body"`
	Type        string                          `json:"type"`
	ContentType string                          `json:"content_type"`
	ImageURL    string                          `json:"image_url"`
	Marks       float64                         `json:"marks"`
	IsRequired  bool                            `json:"is_required"`
	Position    int                             `json:"position"`
	Options     []QuestionOptionStudentResponse `json:"options"`
}

// QuestionOptionStudentResponse represents a question option for student view
type QuestionOptionStudentResponse struct {
	ID          uuid.UUID `json:"id"`
	Body        string    `json:"body"`
	ContentType string    `json:"content_type"`
	ImageURL    string    `json:"image_url"`
	Position    int       `json:"position"`
}

// SubmitAttemptResponse represents the result of a quiz submission
type SubmitAttemptResponse struct {
	ScorePercent     float64                  `json:"score_percent"`
	Passed           bool                     `json:"passed"`
	TimeTakenSeconds int                      `json:"time_taken_seconds"`
	PointsAwarded    int                      `json:"points_awarded"`
	QuestionResults  []QuestionResultResponse `json:"question_results,omitempty"`
}

// QuestionResultResponse represents the result for a single question
type QuestionResultResponse struct {
	QuestionID      uuid.UUID   `json:"question_id"`
	IsCorrect       bool        `json:"is_correct"`
	SelectedOptions []uuid.UUID `json:"selected_options"`
	CorrectOptions  []uuid.UUID `json:"correct_options"`
	Explanation     string      `json:"explanation,omitempty"`
}

// AssignmentResponse represents an assignment in API responses
type AssignmentResponse struct {
	ID                  uuid.UUID `json:"id"`
	CourseID            uuid.UUID `json:"course_id"`
	Title               string    `json:"title"`
	Description         string    `json:"description"`
	DueAt               time.Time `json:"due_at"`
	SubmissionType      string    `json:"submission_type"`
	MaxFileSizeMB       int       `json:"max_file_size_mb"`
	AllowLateSubmission bool      `json:"allow_late_submission"`
	TotalMarks          float64   `json:"total_marks"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// AssignmentSubmissionResponse represents a submission in API responses
type AssignmentSubmissionResponse struct {
	ID           uuid.UUID                `json:"id"`
	AssignmentID uuid.UUID                `json:"assignment_id"`
	StudentID    uuid.UUID                `json:"student_id"`
	Status       string                   `json:"status"`
	TextContent  string                   `json:"text_content"`
	Files        []SubmissionFileResponse `json:"files"`
	LatestGrade  *SubmissionGradeResponse `json:"latest_grade,omitempty"`
	SubmittedAt  *time.Time               `json:"submitted_at,omitempty"`
	IsLate       bool                     `json:"is_late"`
	CreatedAt    time.Time                `json:"created_at"`
	UpdatedAt    time.Time                `json:"updated_at"`
}

// SubmissionFileResponse represents a submission file with presigned URL
type SubmissionFileResponse struct {
	ID               uuid.UUID `json:"id"`
	OriginalFilename string    `json:"original_filename"`
	MimeType         string    `json:"mime_type"`
	SizeBytes        int64     `json:"size_bytes"`
	DownloadURL      string    `json:"download_url"` // Presigned URL, 2h TTL
}

// SubmissionGradeResponse represents a grade in API responses
type SubmissionGradeResponse struct {
	ID                uuid.UUID `json:"id"`
	Score             float64   `json:"score"`
	Feedback          string    `json:"feedback"`
	RevisionRequested bool      `json:"revision_requested"`
	RevisionNotes     string    `json:"revision_notes,omitempty"`
	GradedBy          uuid.UUID `json:"graded_by"`
	GradedAt          time.Time `json:"graded_at"`
}

// StudentQuizSummaryResponse represents a student-visible quiz catalog entry.
type StudentQuizSummaryResponse struct {
	ID                  uuid.UUID             `json:"id"`
	CourseID            uuid.UUID             `json:"course_id"`
	LessonID            *uuid.UUID            `json:"lesson_id"`
	Title               string                `json:"title"`
	TimeLimitSeconds    int                   `json:"time_limit_seconds"`
	MaxAttempts         int                   `json:"max_attempts"`
	PassingScorePercent float64               `json:"passing_score_percent"`
	AttemptsUsed        int                   `json:"attempts_used"`
	LatestAttempt       *StudentAttemptResult `json:"latest_attempt,omitempty"`
}

// StudentAttemptResult represents a completed quiz attempt snapshot.
type StudentAttemptResult struct {
	ID               uuid.UUID  `json:"id"`
	QuizID           uuid.UUID  `json:"quiz_id"`
	Status           string     `json:"status"`
	StartedAt        time.Time  `json:"started_at"`
	SubmittedAt      *time.Time `json:"submitted_at,omitempty"`
	ScorePercent     *float64   `json:"score_percent,omitempty"`
	Passed           *bool      `json:"passed,omitempty"`
	TimeTakenSeconds *int       `json:"time_taken_seconds,omitempty"`
	PointsAwarded    int        `json:"points_awarded"`
}

// StudentQuizDetailResponse represents a student-visible quiz detail.
type StudentQuizDetailResponse struct {
	StudentQuizSummaryResponse
	ShowAnswersAfterSubmission bool `json:"show_answers_after_submission"`
}

// StudentAssignmentDetailResponse includes assignment metadata plus the caller's submission.
type StudentAssignmentDetailResponse struct {
	AssignmentResponse
	Submission *AssignmentSubmissionResponse `json:"submission,omitempty"`
}

// TeacherAssignmentSubmissionListResponse wraps teacher grading queue results.
type TeacherAssignmentSubmissionListResponse struct {
	Assignment  AssignmentResponse             `json:"assignment"`
	Submissions []AssignmentSubmissionResponse `json:"submissions"`
	Meta        struct {
		Page       int `json:"page"`
		Limit      int `json:"limit"`
		Total      int `json:"total"`
		TotalPages int `json:"total_pages"`
	} `json:"meta"`
}
