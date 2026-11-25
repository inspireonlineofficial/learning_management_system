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
	IsFree                     bool       `json:"is_free"`
	IsPublished                bool       `json:"is_published"`
	CreatedAt                  time.Time  `json:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`
	Description                string     `json:"description"`
	Instructions               string     `json:"instructions"`
	Mode                       string     `json:"mode"`
	ShuffleOptions             bool       `json:"shuffle_options"`
	PartialCredit              bool       `json:"partial_credit"`
	AvailableFrom              *time.Time `json:"available_from"`
	AvailableUntil             *time.Time `json:"available_until"`
	CalculatorPolicy           string     `json:"calculator_policy"`
	ReferenceMaterial          string     `json:"reference_material"`
	// Lifecycle as seen now: draft | scheduled | open | closed.
	Lifecycle      string  `json:"lifecycle"`
	QuestionCount  int     `json:"question_count"`
	TotalMarks     float64 `json:"total_marks"`
	AttemptCount   int     `json:"attempt_count"`
	SubmittedCount int     `json:"submitted_count"`
	InProgress     int     `json:"in_progress_count"`
	PendingGrading int     `json:"pending_grading_count"`
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
	Hint             string                          `json:"hint"`
	Settings         QuestionSettingsResponse        `json:"settings"`
	CorrectOptionIDs []uuid.UUID                     `json:"correct_option_ids"`
	Options          []QuestionOptionTeacherResponse `json:"options"`
}

// QuestionSettingsResponse is the teacher view of per-type settings.
type QuestionSettingsResponse struct {
	NumericAnswer *float64 `json:"numeric_answer,omitempty"`
	Tolerance     float64  `json:"tolerance,omitempty"`
	ToleranceType string   `json:"tolerance_type,omitempty"`
	Unit          string   `json:"unit,omitempty"`
	MaxWords      int      `json:"max_words,omitempty"`
	CaseSensitive bool     `json:"case_sensitive,omitempty"`
}

// StudentQuestionSettings is what students may see of the settings: never
// the numeric answer or tolerance.
type StudentQuestionSettings struct {
	Unit     string `json:"unit,omitempty"`
	MaxWords int    `json:"max_words,omitempty"`
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
	// ServerNow lets the client correct for a wrong device clock: remaining
	// time = expires_at - server_now, counted down locally.
	ServerNow         time.Time `json:"server_now"`
	Status            string    `json:"status"`
	DraftRevision     int64     `json:"draft_revision"`
	Saved             *bool     `json:"saved,omitempty"` // autosave only: false = stale write ignored
	QuizTitle         string    `json:"quiz_title"`
	Mode              string    `json:"mode"`
	Instructions      string    `json:"instructions,omitempty"`
	TimeLimitSeconds  int       `json:"time_limit_seconds"`
	CalculatorPolicy  string    `json:"calculator_policy"`
	ReferenceMaterial string    `json:"reference_material,omitempty"`
	TotalMarks        float64   `json:"total_marks"`
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
	Hint        string                          `json:"hint,omitempty"` // practice mode only
	Settings    StudentQuestionSettings         `json:"settings"`
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
	AttemptID        uuid.UUID                `json:"attempt_id"`
	QuizID           uuid.UUID                `json:"quiz_id"`
	Status           string                   `json:"status"`
	// GradingStatus is "pending" while manually graded answers await marks;
	// score_percent and passed are then provisional (auto-graded part only).
	GradingStatus   string           `json:"grading_status"`
	ScorePoints     float64          `json:"score_points"`
	MaxPoints       float64          `json:"max_points"`
	AnswersReleased bool             `json:"answers_released"`
	Review          []QuestionReview `json:"review,omitempty"`
}

// QuestionReview is one question of a submitted attempt as the student may
// see it. Fields that reveal the answer key are omitted unless the teacher
// releases answers (show_answers_after_submission).
type QuestionReview struct {
	QuestionID        uuid.UUID       `json:"question_id"`
	Position          int             `json:"position"`
	Type              string          `json:"type"`
	Body              string          `json:"body"`
	ContentType       string          `json:"content_type,omitempty"`
	ImageURL          string          `json:"image_url,omitempty"`
	Marks             float64         `json:"marks"`
	Options           []OutcomeOption `json:"options,omitempty"`
	Unit              string          `json:"unit,omitempty"`
	SelectedOptionIDs []uuid.UUID     `json:"selected_option_ids"`
	TextAnswer        string          `json:"text_answer,omitempty"`
	AnswerImageURL    string          `json:"answer_image_url,omitempty"`
	Answered          bool            `json:"answered"`
	// Released only.
	Awarded          *float64    `json:"awarded,omitempty"`
	Status           string      `json:"status,omitempty"`
	CorrectOptionIDs []uuid.UUID `json:"correct_option_ids,omitempty"`
	AcceptedAnswers  []string    `json:"accepted_answers,omitempty"`
	NumericAnswer    *float64    `json:"numeric_answer,omitempty"`
	Tolerance        float64     `json:"tolerance,omitempty"`
	ToleranceType    string      `json:"tolerance_type,omitempty"`
	Explanation      string      `json:"explanation,omitempty"`
	Feedback         string      `json:"feedback,omitempty"`
	// Pending is true while a manually graded answer awaits marks.
	Pending bool `json:"pending,omitempty"`
}

// CheckAnswerResponse is instant feedback in practice mode.
type CheckAnswerResponse struct {
	QuestionID       uuid.UUID   `json:"question_id"`
	Status           string      `json:"status"`
	Correct          bool        `json:"correct"`
	CorrectOptionIDs []uuid.UUID `json:"correct_option_ids,omitempty"`
	AcceptedAnswers  []string    `json:"accepted_answers,omitempty"`
	NumericAnswer    *float64    `json:"numeric_answer,omitempty"`
	Explanation      string      `json:"explanation,omitempty"`
}

// QuestionResultResponse represents the result for a single question
type QuestionResultResponse struct {
	QuestionID      uuid.UUID   `json:"question_id"`
	IsCorrect       bool        `json:"is_correct"`
	SelectedOptions []uuid.UUID `json:"selected_options"`
	CorrectOptions  []uuid.UUID `json:"correct_options"`
	TextAnswer      string      `json:"text_answer,omitempty"`
	ImageURL        string      `json:"image_url,omitempty"`
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
	IsFree              bool                  `json:"is_free"`
	IsLocked            bool                  `json:"is_locked"`
	AttemptsUsed        int                   `json:"attempts_used"`
	LatestAttempt       *StudentAttemptResult `json:"latest_attempt,omitempty"`
	CourseTitle         string                `json:"course_title,omitempty"`
	Description         string                `json:"description"`
	Instructions        string                `json:"instructions"`
	Mode                string                `json:"mode"`
	QuestionCount       int                   `json:"question_count"`
	TotalMarks          float64               `json:"total_marks"`
	AvailableFrom       *time.Time            `json:"available_from"`
	AvailableUntil      *time.Time            `json:"available_until"`
	// Availability is open | not_open | closed.
	Availability        string     `json:"availability"`
	CalculatorPolicy    string     `json:"calculator_policy"`
	ReferenceMaterial   string     `json:"reference_material"`
	RemainingAttempts   *int       `json:"remaining_attempts"`
	InProgressAttemptID *uuid.UUID `json:"in_progress_attempt_id,omitempty"`
	BestScorePercent    *float64   `json:"best_score_percent,omitempty"`
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
	GradingStatus    string     `json:"grading_status"`
	ScorePoints      *float64   `json:"score_points,omitempty"`
	MaxPoints        *float64   `json:"max_points,omitempty"`
	// Result-page fields (filled for submitted attempts).
	QuizTitle           string           `json:"quiz_title,omitempty"`
	PassingScorePercent float64          `json:"passing_score_percent,omitempty"`
	AnswersReleased     bool             `json:"answers_released"`
	ReviewAvailable     bool             `json:"review_available"`
	Review              []QuestionReview `json:"review,omitempty"`
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

// TeacherAttemptRow is one attempt in the teacher monitoring table.
type TeacherAttemptRow struct {
	AttemptID        uuid.UUID  `json:"attempt_id"`
	StudentID        uuid.UUID  `json:"student_id"`
	StudentName      string     `json:"student_name"`
	StudentEmail     string     `json:"student_email"`
	AttemptNumber    int        `json:"attempt_number"`
	Status           string     `json:"status"`
	GradingStatus    string     `json:"grading_status"`
	StartedAt        time.Time  `json:"started_at"`
	SubmittedAt      *time.Time `json:"submitted_at,omitempty"`
	TimeTakenSeconds *int       `json:"time_taken_seconds,omitempty"`
	ScorePercent     *float64   `json:"score_percent,omitempty"`
	ScorePoints      *float64   `json:"score_points,omitempty"`
	MaxPoints        *float64   `json:"max_points,omitempty"`
	Passed           *bool      `json:"passed,omitempty"`
}

// QuizAttemptSummary aggregates attempts for the monitoring header. Scores
// use each student's latest finally-graded attempt.
type QuizAttemptSummary struct {
	EnrolledStudents   int      `json:"enrolled_students"`
	StudentsAttempted  int      `json:"students_attempted"`
	StudentsNotStarted int      `json:"students_not_started"`
	TotalAttempts      int      `json:"total_attempts"`
	InProgress         int      `json:"in_progress"`
	Submitted          int      `json:"submitted"`
	PendingGrading     int      `json:"pending_grading"`
	AverageScore       *float64 `json:"average_score,omitempty"`
	HighestScore       *float64 `json:"highest_score,omitempty"`
	LowestScore        *float64 `json:"lowest_score,omitempty"`
	PassRate           *float64 `json:"pass_rate,omitempty"`
	GradedAttempts     int      `json:"graded_attempts"`
}

// QuizAttemptsResponse is the teacher monitoring view of one quiz.
type QuizAttemptsResponse struct {
	Quiz     QuizResponse        `json:"quiz"`
	Summary  QuizAttemptSummary  `json:"summary"`
	Attempts []TeacherAttemptRow `json:"attempts"`
	Meta     PageMeta            `json:"meta"`
}

// PageMeta describes a page of results.
type PageMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// TeacherAttemptDetailResponse is one attempt with every question outcome.
type TeacherAttemptDetailResponse struct {
	TeacherAttemptRow
	QuizTitle       string            `json:"quiz_title"`
	ReviewAvailable bool              `json:"review_available"`
	Results         []QuestionOutcome `json:"results"`
	DraftAnswers    map[string]any    `json:"draft_answers,omitempty"` // in-progress attempts
}

// QuizAnalyticsResponse is built only from persisted submitted attempts.
type QuizAnalyticsResponse struct {
	QuizID            uuid.UUID `json:"quiz_id"`
	SubmittedAttempts int       `json:"submitted_attempts"`
	GradedAttempts    int       `json:"graded_attempts"`
	// SmallSample warns that fewer than 5 graded attempts exist, so rates
	// should not be read as reliable.
	SmallSample        bool                `json:"small_sample"`
	AverageScore       *float64            `json:"average_score,omitempty"`
	MedianScore        *float64            `json:"median_score,omitempty"`
	AverageTimeSeconds *float64            `json:"average_time_seconds,omitempty"`
	ScoreDistribution  []ScoreBucket       `json:"score_distribution"`
	Questions          []QuestionAnalytics `json:"questions"`
}

// ScoreBucket counts graded attempts with a score in [From, To).
type ScoreBucket struct {
	From  int `json:"from"`
	To    int `json:"to"`
	Count int `json:"count"`
}

// QuestionAnalytics summarises one question across submitted attempts.
type QuestionAnalytics struct {
	QuestionID      uuid.UUID        `json:"question_id"`
	Position        int              `json:"position"`
	Type            string           `json:"type"`
	Body            string           `json:"body"`
	Marks           float64          `json:"marks"`
	Responses       int              `json:"responses"`
	Correct         int              `json:"correct"`
	Partial         int              `json:"partial"`
	Incorrect       int              `json:"incorrect"`
	Unanswered      int              `json:"unanswered"`
	Pending         int              `json:"pending"`
	AverageAwarded  float64          `json:"average_awarded"`
	AccuracyPercent *float64         `json:"accuracy_percent,omitempty"`
	OptionCounts    []OptionAnalytic `json:"option_counts,omitempty"`
	CommonWrong     []TextAnalytic   `json:"common_wrong_answers,omitempty"`
}

// OptionAnalytic counts how often an option was chosen.
type OptionAnalytic struct {
	OptionID  uuid.UUID `json:"option_id"`
	Body      string    `json:"body"`
	IsCorrect bool      `json:"is_correct"`
	Count     int       `json:"count"`
}

// TextAnalytic counts a normalised wrong text answer.
type TextAnalytic struct {
	Answer string `json:"answer"`
	Count  int    `json:"count"`
}
