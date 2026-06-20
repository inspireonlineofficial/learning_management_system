package assessments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"lms-backend/internal/domain/assessments"
	"lms-backend/internal/domain/courses"
	domainenrollments "lms-backend/internal/domain/enrollments"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// Service defines the interface for assessment use cases
type Service interface {
	// Teacher operations - Quiz
	CreateQuiz(ctx context.Context, cmd CreateQuizCommand) (*QuizResponse, error)
	GetTeacherQuizzes(ctx context.Context, courseID, teacherID uuid.UUID) ([]QuizTeacherResponse, error)
	GetTeacherQuiz(ctx context.Context, quizID, teacherID uuid.UUID) (*QuizTeacherResponse, error)
	UpdateQuiz(ctx context.Context, cmd UpdateQuizCommand) (*QuizTeacherResponse, error)
	DeleteQuiz(ctx context.Context, quizID, teacherID uuid.UUID) error
	CreateQuestion(ctx context.Context, cmd CreateStandaloneQuestionCommand) (*QuestionTeacherResponse, error)
	UpdateQuestion(ctx context.Context, cmd UpdateQuestionCommand) (*QuestionTeacherResponse, error)
	DeleteQuestion(ctx context.Context, questionID, teacherID uuid.UUID) error

	// Student operations - Quiz attempts
	ListStudentQuizzes(ctx context.Context, studentID uuid.UUID) ([]StudentQuizSummaryResponse, error)
	GetStudentQuizDetail(ctx context.Context, quizID, studentID uuid.UUID) (*StudentQuizDetailResponse, error)
	GetStudentQuizAttemptResult(ctx context.Context, quizID, attemptID, studentID uuid.UUID) (*StudentAttemptResult, error)
	GetStudentAttempt(ctx context.Context, attemptID, studentID uuid.UUID) (*StudentAttemptResult, error)
	StartAttempt(ctx context.Context, cmd StartAttemptCommand) (*QuizAttemptResponse, error)
	SaveAttemptAnswers(ctx context.Context, cmd SaveAttemptAnswersCommand) (*QuizAttemptResponse, error)
	SubmitAttempt(ctx context.Context, cmd SubmitAttemptCommand) (*SubmitAttemptResponse, error)
	AutoSubmitAttempt(ctx context.Context, attemptID uuid.UUID) (*SubmitAttemptResponse, error)

	// Teacher operations - Assignments
	CreateAssignment(ctx context.Context, cmd CreateAssignmentCommand) (*AssignmentResponse, error)
	ListTeacherCourseAssignments(ctx context.Context, courseID, teacherID uuid.UUID) ([]AssignmentResponse, error)
	GetTeacherAssignment(ctx context.Context, assignmentID, teacherID uuid.UUID) (*AssignmentResponse, error)
	UpdateAssignment(ctx context.Context, cmd UpdateAssignmentCommand) (*AssignmentResponse, error)
	ListTeacherAssignmentSubmissions(ctx context.Context, assignmentID, teacherID uuid.UUID, page, limit int) (*TeacherAssignmentSubmissionListResponse, error)
	GetTeacherAssignmentSubmission(ctx context.Context, assignmentID, submissionID, teacherID uuid.UUID) (*AssignmentSubmissionResponse, error)

	// Student operations - Assignment submissions
	ListStudentAssignments(ctx context.Context, studentID uuid.UUID) ([]StudentAssignmentDetailResponse, error)
	GetStudentAssignmentDetail(ctx context.Context, assignmentID, studentID uuid.UUID) (*StudentAssignmentDetailResponse, error)
	SubmitAssignment(ctx context.Context, cmd SubmitAssignmentCommand) (*AssignmentSubmissionResponse, error)

	// Teacher operations - Grading
	GradeSubmission(ctx context.Context, cmd GradeSubmissionCommand) (*SubmissionGradeResponse, error)
}

type service struct {
	quizRepo           assessments.QuizRepository
	questionRepo       assessments.QuestionRepository
	optionRepo         assessments.QuestionOptionRepository
	attemptRepo        assessments.QuizAttemptRepository
	assignmentRepo     assessments.AssignmentRepository
	submissionRepo     assessments.AssignmentSubmissionRepository
	submissionFileRepo assessments.SubmissionFileRepository
	gradeRepo          assessments.SubmissionGradeRepository
	enrollmentRepo     domainenrollments.EnrollmentRepository
	courseRepo         courses.CourseRepository
	storageClient      StorageClient
	notificationQueue  NotificationQueue
	filesBucket        string
}

// StorageClient defines the interface for object storage operations
type StorageClient interface {
	PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) error
	PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)
	DeleteObject(ctx context.Context, bucket, key string) error
}

// NotificationQueue interface for enqueueing notifications
type NotificationQueue interface {
	EnqueueNotification(ctx context.Context, userID uuid.UUID, notificationType, title, body string) error
}

// NewService creates a new assessment service
func NewService(
	quizRepo assessments.QuizRepository,
	questionRepo assessments.QuestionRepository,
	optionRepo assessments.QuestionOptionRepository,
	attemptRepo assessments.QuizAttemptRepository,
	assignmentRepo assessments.AssignmentRepository,
	submissionRepo assessments.AssignmentSubmissionRepository,
	submissionFileRepo assessments.SubmissionFileRepository,
	gradeRepo assessments.SubmissionGradeRepository,
	args ...interface{},
) Service {
	var enrollmentRepo domainenrollments.EnrollmentRepository
	var courseRepo courses.CourseRepository
	var storageClient StorageClient
	var notificationQueue NotificationQueue
	var filesBucket string

	if len(args) == 5 {
		enrollmentRepo, _ = args[0].(domainenrollments.EnrollmentRepository)
		courseRepo, _ = args[1].(courses.CourseRepository)
		storageClient, _ = args[2].(StorageClient)
		notificationQueue, _ = args[3].(NotificationQueue)
		filesBucket, _ = args[4].(string)
	} else if len(args) == 4 {
		courseRepo, _ = args[0].(courses.CourseRepository)
		storageClient, _ = args[1].(StorageClient)
		notificationQueue, _ = args[2].(NotificationQueue)
		filesBucket, _ = args[3].(string)
	}

	return &service{
		quizRepo:           quizRepo,
		questionRepo:       questionRepo,
		optionRepo:         optionRepo,
		attemptRepo:        attemptRepo,
		assignmentRepo:     assignmentRepo,
		submissionRepo:     submissionRepo,
		submissionFileRepo: submissionFileRepo,
		gradeRepo:          gradeRepo,
		enrollmentRepo:     enrollmentRepo,
		courseRepo:         courseRepo,
		storageClient:      storageClient,
		notificationQueue:  notificationQueue,
		filesBucket:        filesBucket,
	}
}

// CreateQuiz creates a new quiz with questions and options
func (s *service) CreateQuiz(ctx context.Context, cmd CreateQuizCommand) (*QuizResponse, error) {
	// Verify course exists and teacher owns it
	course, err := s.courseRepo.FindByID(ctx, cmd.CourseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}

	if !course.IsOwnedBy(cmd.TeacherID) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to create quiz for this course")
	}

	if err := validateCreateQuiz(cmd); err != nil {
		return nil, err
	}

	// Create quiz
	quiz := &assessments.Quiz{
		ID:                         uuid.New(),
		CourseID:                   cmd.CourseID,
		LessonID:                   cmd.LessonID,
		Title:                      cmd.Title,
		TimeLimitSeconds:           cmd.TimeLimitSeconds,
		MaxAttempts:                cmd.MaxAttempts,
		PassingScorePercent:        cmd.PassingScorePercent,
		ShuffleQuestions:           cmd.ShuffleQuestions,
		ShowAnswersAfterSubmission: cmd.ShowAnswersAfterSubmission,
		IsFree:                     cmd.IsFree,
		IsPublished:                cmd.IsPublished,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}

	err = s.quizRepo.Create(ctx, quiz)
	if err != nil {
		return nil, err
	}

	// Create questions and options
	for _, qCmd := range cmd.Questions {
		question := &assessments.Question{
			ID:          uuid.New(),
			QuizID:      quiz.ID,
			Body:        qCmd.Body,
			Type:        assessments.QuestionType(qCmd.Type),
			ContentType: normalizeContentType(qCmd.ContentType, qCmd.Body, qCmd.ImageURL),
			ImageURL:    strings.TrimSpace(qCmd.ImageURL),
			Marks:       normalizeMarks(qCmd.Marks),
			IsRequired:  qCmd.IsRequired,
			Position:    qCmd.Position,
			Explanation: qCmd.Explanation,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		err = s.questionRepo.Create(ctx, question)
		if err != nil {
			return nil, err
		}

		// Create options for this question
		options := make([]*assessments.QuestionOption, 0, len(qCmd.Options))
		for _, optCmd := range qCmd.Options {
			option := &assessments.QuestionOption{
				ID:          uuid.New(),
				QuestionID:  question.ID,
				Body:        optCmd.Body,
				ContentType: normalizeContentType(optCmd.ContentType, optCmd.Body, optCmd.ImageURL),
				ImageURL:    strings.TrimSpace(optCmd.ImageURL),
				IsCorrect:   optCmd.IsCorrect,
				Position:    optCmd.Position,
			}
			options = append(options, option)
		}

		if len(options) > 0 {
			err = s.optionRepo.CreateBatch(ctx, options)
			if err != nil {
				return nil, err
			}
		}
	}

	return s.toQuizResponse(quiz), nil
}

func validateCreateQuiz(cmd CreateQuizCommand) error {
	if strings.TrimSpace(cmd.Title) == "" {
		return apperrors.NewSimpleValidationError("TITLE_REQUIRED", "quiz title is required")
	}
	if cmd.TimeLimitSeconds <= 0 {
		return apperrors.NewSimpleValidationError("INVALID_TIME_LIMIT", "time limit must be greater than zero")
	}
	if cmd.MaxAttempts < 0 {
		return apperrors.NewSimpleValidationError("INVALID_MAX_ATTEMPTS", "max attempts cannot be negative")
	}
	if cmd.PassingScorePercent < 0 || cmd.PassingScorePercent > 100 {
		return apperrors.NewSimpleValidationError("INVALID_PASSING_SCORE", "passing score must be between 0 and 100")
	}
	if len(cmd.Questions) == 0 {
		return apperrors.NewSimpleValidationError("QUESTIONS_REQUIRED", "at least one question is required")
	}

	for _, question := range cmd.Questions {
		if !hasTextOrImage(question.Body, question.ImageURL) {
			return apperrors.NewSimpleValidationError("QUESTION_CONTENT_REQUIRED", "each question must include text or an image")
		}
		if question.Type != string(assessments.QuestionTypeSingle) &&
			question.Type != string(assessments.QuestionTypeMultiple) &&
			question.Type != string(assessments.QuestionTypeTrueFalse) {
			return apperrors.NewSimpleValidationError("INVALID_QUESTION_TYPE", "question type is invalid")
		}
		if len(question.Options) < 2 {
			return apperrors.NewSimpleValidationError("OPTIONS_REQUIRED", "each question must include at least two options")
		}

		correctOptions := 0
		for _, option := range question.Options {
			if !hasTextOrImage(option.Body, option.ImageURL) {
				return apperrors.NewSimpleValidationError("OPTION_CONTENT_REQUIRED", "each answer option must include text or an image")
			}
			if option.IsCorrect {
				correctOptions++
			}
		}
		if correctOptions == 0 {
			return apperrors.NewSimpleValidationError("CORRECT_OPTION_REQUIRED", "each question must include a correct option")
		}
		if question.Type != string(assessments.QuestionTypeMultiple) && correctOptions != 1 {
			return apperrors.NewSimpleValidationError("INVALID_CORRECT_OPTIONS", "single-answer questions must include exactly one correct option")
		}
	}

	return nil
}

func hasTextOrImage(text, imageURL string) bool {
	return strings.TrimSpace(text) != "" || strings.TrimSpace(imageURL) != ""
}

func normalizeContentType(raw, text, imageURL string) string {
	contentType := strings.TrimSpace(raw)
	if contentType == "text" || contentType == "image" || contentType == "text_image" {
		return contentType
	}
	hasText := strings.TrimSpace(text) != ""
	hasImage := strings.TrimSpace(imageURL) != ""
	if hasText && hasImage {
		return "text_image"
	}
	if hasImage {
		return "image"
	}
	return "text"
}

func normalizeMarks(raw float64) float64 {
	if raw > 0 {
		return raw
	}
	return 1
}

// GetTeacherQuizzes returns all quizzes for a course with answer keys (teacher view)
func (s *service) GetTeacherQuizzes(ctx context.Context, courseID, teacherID uuid.UUID) ([]QuizTeacherResponse, error) {
	// Verify course exists and teacher owns it
	course, err := s.courseRepo.FindByID(ctx, courseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}

	if !course.IsOwnedBy(teacherID) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to view quizzes for this course")
	}

	// Get all quizzes for the course
	quizzes, err := s.quizRepo.FindByCourseID(ctx, courseID)
	if err != nil {
		return nil, err
	}

	// Build teacher responses with questions and answer keys
	responses := make([]QuizTeacherResponse, 0, len(quizzes))
	for _, quiz := range quizzes {
		questions, err := s.questionRepo.FindByQuizID(ctx, quiz.ID)
		if err != nil {
			return nil, err
		}

		// Get question IDs for batch option fetch
		questionIDs := make([]uuid.UUID, len(questions))
		for i, q := range questions {
			questionIDs[i] = q.ID
		}

		// Fetch all options for all questions in one call
		optionsMap, err := s.optionRepo.FindByQuestionIDs(ctx, questionIDs)
		if err != nil {
			return nil, err
		}

		// Build question responses with answer keys
		questionResponses := make([]QuestionTeacherResponse, 0, len(questions))
		for _, question := range questions {
			options := optionsMap[question.ID]
			questionResponses = append(questionResponses, s.toQuestionTeacherResponse(question, options))
		}

		responses = append(responses, QuizTeacherResponse{
			QuizResponse: *s.toQuizResponse(quiz),
			Questions:    questionResponses,
		})
	}

	return responses, nil
}

func (s *service) GetTeacherQuiz(ctx context.Context, quizID, teacherID uuid.UUID) (*QuizTeacherResponse, error) {
	quiz, err := s.quizRepo.FindByID(ctx, quizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureTeacherOwnsCourse(ctx, quiz.CourseID, teacherID); err != nil {
		return nil, err
	}
	return s.toQuizTeacherResponse(ctx, quiz)
}

func (s *service) UpdateQuiz(ctx context.Context, cmd UpdateQuizCommand) (*QuizTeacherResponse, error) {
	quiz, err := s.quizRepo.FindByID(ctx, cmd.QuizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureTeacherOwnsCourse(ctx, quiz.CourseID, cmd.TeacherID); err != nil {
		return nil, err
	}

	createCmd := CreateQuizCommand{
		CourseID:                   quiz.CourseID,
		TeacherID:                  cmd.TeacherID,
		LessonID:                   cmd.LessonID,
		Title:                      cmd.Title,
		TimeLimitSeconds:           cmd.TimeLimitSeconds,
		MaxAttempts:                cmd.MaxAttempts,
		PassingScorePercent:        cmd.PassingScorePercent,
		ShuffleQuestions:           cmd.ShuffleQuestions,
		ShowAnswersAfterSubmission: cmd.ShowAnswersAfterSubmission,
		IsFree:                     cmd.IsFree,
		IsPublished:                cmd.IsPublished,
		Questions:                  cmd.Questions,
	}
	if err := validateCreateQuiz(createCmd); err != nil {
		return nil, err
	}

	quiz.LessonID = cmd.LessonID
	quiz.Title = cmd.Title
	quiz.TimeLimitSeconds = cmd.TimeLimitSeconds
	quiz.MaxAttempts = cmd.MaxAttempts
	quiz.PassingScorePercent = cmd.PassingScorePercent
	quiz.ShuffleQuestions = cmd.ShuffleQuestions
	quiz.ShowAnswersAfterSubmission = cmd.ShowAnswersAfterSubmission
	quiz.IsFree = cmd.IsFree
	quiz.IsPublished = cmd.IsPublished
	quiz.UpdatedAt = time.Now().UTC()

	if err := s.quizRepo.Update(ctx, quiz); err != nil {
		return nil, err
	}

	existingQuestions, err := s.questionRepo.FindByQuizID(ctx, quiz.ID)
	if err != nil {
		return nil, err
	}
	for _, question := range existingQuestions {
		if err := s.questionRepo.Delete(ctx, question.ID); err != nil {
			return nil, err
		}
	}

	for _, qCmd := range cmd.Questions {
		question := &assessments.Question{
			ID:          uuid.New(),
			QuizID:      quiz.ID,
			Body:        qCmd.Body,
			Type:        assessments.QuestionType(qCmd.Type),
			ContentType: normalizeContentType(qCmd.ContentType, qCmd.Body, qCmd.ImageURL),
			ImageURL:    strings.TrimSpace(qCmd.ImageURL),
			Marks:       normalizeMarks(qCmd.Marks),
			IsRequired:  qCmd.IsRequired,
			Position:    qCmd.Position,
			Explanation: qCmd.Explanation,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}
		if err := s.questionRepo.Create(ctx, question); err != nil {
			return nil, err
		}
		options := make([]*assessments.QuestionOption, 0, len(qCmd.Options))
		for _, optCmd := range qCmd.Options {
			options = append(options, &assessments.QuestionOption{
				ID:          uuid.New(),
				QuestionID:  question.ID,
				Body:        optCmd.Body,
				ContentType: normalizeContentType(optCmd.ContentType, optCmd.Body, optCmd.ImageURL),
				ImageURL:    strings.TrimSpace(optCmd.ImageURL),
				IsCorrect:   optCmd.IsCorrect,
				Position:    optCmd.Position,
			})
		}
		if len(options) > 0 {
			if err := s.optionRepo.CreateBatch(ctx, options); err != nil {
				return nil, err
			}
		}
	}

	return s.toQuizTeacherResponse(ctx, quiz)
}

func (s *service) DeleteQuiz(ctx context.Context, quizID, teacherID uuid.UUID) error {
	quiz, err := s.quizRepo.FindByID(ctx, quizID)
	if err != nil {
		return apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureTeacherOwnsCourse(ctx, quiz.CourseID, teacherID); err != nil {
		return err
	}
	return s.quizRepo.Delete(ctx, quizID)
}

func (s *service) CreateQuestion(ctx context.Context, cmd CreateStandaloneQuestionCommand) (*QuestionTeacherResponse, error) {
	quiz, err := s.quizRepo.FindByID(ctx, cmd.QuizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureTeacherOwnsCourse(ctx, quiz.CourseID, cmd.TeacherID); err != nil {
		return nil, err
	}
	if err := validateCreateQuiz(CreateQuizCommand{
		CourseID:                   quiz.CourseID,
		TeacherID:                  cmd.TeacherID,
		Title:                      "validation",
		TimeLimitSeconds:           maxInt(quiz.TimeLimitSeconds, 1),
		MaxAttempts:                quiz.MaxAttempts,
		PassingScorePercent:        quiz.PassingScorePercent,
		ShowAnswersAfterSubmission: quiz.ShowAnswersAfterSubmission,
		Questions:                  []CreateQuestionCommand{cmd.Question},
	}); err != nil {
		return nil, err
	}

	question := &assessments.Question{
		ID:          uuid.New(),
		QuizID:      quiz.ID,
		Body:        cmd.Question.Body,
		Type:        assessments.QuestionType(cmd.Question.Type),
		ContentType: normalizeContentType(cmd.Question.ContentType, cmd.Question.Body, cmd.Question.ImageURL),
		ImageURL:    strings.TrimSpace(cmd.Question.ImageURL),
		Marks:       normalizeMarks(cmd.Question.Marks),
		IsRequired:  cmd.Question.IsRequired,
		Position:    cmd.Question.Position,
		Explanation: cmd.Question.Explanation,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if question.Position <= 0 {
		existing, err := s.questionRepo.FindByQuizID(ctx, quiz.ID)
		if err != nil {
			return nil, err
		}
		question.Position = len(existing) + 1
	}
	if err := s.questionRepo.Create(ctx, question); err != nil {
		return nil, err
	}
	options, err := s.createQuestionOptions(ctx, question.ID, cmd.Question.Options)
	if err != nil {
		return nil, err
	}
	response := s.toQuestionTeacherResponse(question, options)
	return &response, nil
}

func (s *service) UpdateQuestion(ctx context.Context, cmd UpdateQuestionCommand) (*QuestionTeacherResponse, error) {
	question, err := s.questionRepo.FindByID(ctx, cmd.QuestionID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUESTION_NOT_FOUND", "question not found")
	}
	quiz, err := s.quizRepo.FindByID(ctx, question.QuizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureTeacherOwnsCourse(ctx, quiz.CourseID, cmd.TeacherID); err != nil {
		return nil, err
	}
	if err := validateCreateQuiz(CreateQuizCommand{
		CourseID:                   quiz.CourseID,
		TeacherID:                  cmd.TeacherID,
		Title:                      "validation",
		TimeLimitSeconds:           maxInt(quiz.TimeLimitSeconds, 1),
		MaxAttempts:                quiz.MaxAttempts,
		PassingScorePercent:        quiz.PassingScorePercent,
		ShowAnswersAfterSubmission: quiz.ShowAnswersAfterSubmission,
		Questions:                  []CreateQuestionCommand{cmd.Question},
	}); err != nil {
		return nil, err
	}

	question.Body = cmd.Question.Body
	question.Type = assessments.QuestionType(cmd.Question.Type)
	question.ContentType = normalizeContentType(cmd.Question.ContentType, cmd.Question.Body, cmd.Question.ImageURL)
	question.ImageURL = strings.TrimSpace(cmd.Question.ImageURL)
	question.Marks = normalizeMarks(cmd.Question.Marks)
	question.IsRequired = cmd.Question.IsRequired
	question.Position = cmd.Question.Position
	question.Explanation = cmd.Question.Explanation
	question.UpdatedAt = time.Now().UTC()
	if question.Position <= 0 {
		question.Position = 1
	}
	if err := s.questionRepo.Update(ctx, question); err != nil {
		return nil, err
	}

	existingOptions, err := s.optionRepo.FindByQuestionID(ctx, question.ID)
	if err != nil {
		return nil, err
	}
	for _, option := range existingOptions {
		if err := s.optionRepo.Delete(ctx, option.ID); err != nil {
			return nil, err
		}
	}
	options, err := s.createQuestionOptions(ctx, question.ID, cmd.Question.Options)
	if err != nil {
		return nil, err
	}
	response := s.toQuestionTeacherResponse(question, options)
	return &response, nil
}

func (s *service) DeleteQuestion(ctx context.Context, questionID, teacherID uuid.UUID) error {
	question, err := s.questionRepo.FindByID(ctx, questionID)
	if err != nil {
		return apperrors.NewNotFoundError("QUESTION_NOT_FOUND", "question not found")
	}
	quiz, err := s.quizRepo.FindByID(ctx, question.QuizID)
	if err != nil {
		return apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureTeacherOwnsCourse(ctx, quiz.CourseID, teacherID); err != nil {
		return err
	}
	return s.questionRepo.Delete(ctx, questionID)
}

func (s *service) createQuestionOptions(ctx context.Context, questionID uuid.UUID, optionCommands []CreateQuestionOptionCommand) ([]*assessments.QuestionOption, error) {
	options := make([]*assessments.QuestionOption, 0, len(optionCommands))
	for _, optCmd := range optionCommands {
		options = append(options, &assessments.QuestionOption{
			ID:          uuid.New(),
			QuestionID:  questionID,
			Body:        optCmd.Body,
			ContentType: normalizeContentType(optCmd.ContentType, optCmd.Body, optCmd.ImageURL),
			ImageURL:    strings.TrimSpace(optCmd.ImageURL),
			IsCorrect:   optCmd.IsCorrect,
			Position:    optCmd.Position,
		})
	}
	if len(options) > 0 {
		if err := s.optionRepo.CreateBatch(ctx, options); err != nil {
			return nil, err
		}
	}
	return options, nil
}

func (s *service) ensureTeacherOwnsCourse(ctx context.Context, courseID, teacherID uuid.UUID) error {
	course, err := s.courseRepo.FindByID(ctx, courseID)
	if err != nil {
		return apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}
	if !course.IsOwnedBy(teacherID) {
		return apperrors.NewForbiddenError("FORBIDDEN", "not authorized to manage this quiz")
	}
	return nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (s *service) toQuizTeacherResponse(ctx context.Context, quiz *assessments.Quiz) (*QuizTeacherResponse, error) {
	questions, err := s.questionRepo.FindByQuizID(ctx, quiz.ID)
	if err != nil {
		return nil, err
	}
	questionIDs := make([]uuid.UUID, len(questions))
	for i, question := range questions {
		questionIDs[i] = question.ID
	}
	optionsMap, err := s.optionRepo.FindByQuestionIDs(ctx, questionIDs)
	if err != nil {
		return nil, err
	}
	questionResponses := make([]QuestionTeacherResponse, 0, len(questions))
	for _, question := range questions {
		questionResponses = append(questionResponses, s.toQuestionTeacherResponse(question, optionsMap[question.ID]))
	}
	return &QuizTeacherResponse{
		QuizResponse: *s.toQuizResponse(quiz),
		Questions:    questionResponses,
	}, nil
}

func (s *service) toQuizResponse(quiz *assessments.Quiz) *QuizResponse {
	return &QuizResponse{
		ID:                         quiz.ID,
		CourseID:                   quiz.CourseID,
		LessonID:                   quiz.LessonID,
		Title:                      quiz.Title,
		TimeLimitSeconds:           quiz.TimeLimitSeconds,
		MaxAttempts:                quiz.MaxAttempts,
		PassingScorePercent:        quiz.PassingScorePercent,
		ShuffleQuestions:           quiz.ShuffleQuestions,
		ShowAnswersAfterSubmission: quiz.ShowAnswersAfterSubmission,
		IsFree:                     quiz.IsFree,
		IsPublished:                quiz.IsPublished,
		CreatedAt:                  quiz.CreatedAt,
		UpdatedAt:                  quiz.UpdatedAt,
	}
}

func (s *service) toQuestionTeacherResponse(question *assessments.Question, options []*assessments.QuestionOption) QuestionTeacherResponse {
	correctIDs := make([]uuid.UUID, 0)
	optionResponses := make([]QuestionOptionTeacherResponse, 0, len(options))

	for _, opt := range options {
		if opt.IsCorrect {
			correctIDs = append(correctIDs, opt.ID)
		}
		optionResponses = append(optionResponses, QuestionOptionTeacherResponse{
			ID:          opt.ID,
			Body:        opt.Body,
			ContentType: opt.ContentType,
			ImageURL:    opt.ImageURL,
			IsCorrect:   opt.IsCorrect,
			Position:    opt.Position,
		})
	}

	return QuestionTeacherResponse{
		ID:               question.ID,
		QuizID:           question.QuizID,
		Body:             question.Body,
		Type:             string(question.Type),
		ContentType:      question.ContentType,
		ImageURL:         question.ImageURL,
		Marks:            question.Marks,
		IsRequired:       question.IsRequired,
		Position:         question.Position,
		Explanation:      question.Explanation,
		CorrectOptionIDs: correctIDs,
		Options:          optionResponses,
	}
}

// ListStudentQuizzes returns quizzes across the student's enrolled courses.
func (s *service) ListStudentQuizzes(ctx context.Context, studentID uuid.UUID) ([]StudentQuizSummaryResponse, error) {
	courseIDs, err := s.listStudentCourseIDs(ctx, studentID)
	if err != nil {
		return nil, err
	}

	responses := make([]StudentQuizSummaryResponse, 0)
	for _, courseID := range courseIDs {
		quizzes, err := s.quizRepo.FindByCourseID(ctx, courseID)
		if err != nil {
			return nil, err
		}

		for _, quiz := range quizzes {
			attempts, err := s.attemptRepo.FindByQuizAndStudent(ctx, quiz.ID, studentID)
			if err != nil {
				return nil, err
			}

			responses = append(responses, StudentQuizSummaryResponse{
				ID:                  quiz.ID,
				CourseID:            quiz.CourseID,
				LessonID:            quiz.LessonID,
				Title:               quiz.Title,
				TimeLimitSeconds:    quiz.TimeLimitSeconds,
				MaxAttempts:         quiz.MaxAttempts,
				PassingScorePercent: quiz.PassingScorePercent,
				IsFree:              quiz.IsFree,
				IsLocked:            false,
				AttemptsUsed:        len(attempts),
				LatestAttempt:       toStudentAttemptResult(firstAttempt(attempts)),
			})
		}
	}

	return responses, nil
}

// GetStudentQuizDetail returns a student-visible quiz summary without answer keys.
func (s *service) GetStudentQuizDetail(ctx context.Context, quizID, studentID uuid.UUID) (*StudentQuizDetailResponse, error) {
	quiz, err := s.quizRepo.FindByID(ctx, quizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}

	if err := s.ensureStudentCanAccessQuiz(ctx, studentID, quiz); err != nil {
		return nil, err
	}

	attempts, err := s.attemptRepo.FindByQuizAndStudent(ctx, quiz.ID, studentID)
	if err != nil {
		return nil, err
	}

	return &StudentQuizDetailResponse{
		StudentQuizSummaryResponse: StudentQuizSummaryResponse{
			ID:                  quiz.ID,
			CourseID:            quiz.CourseID,
			LessonID:            quiz.LessonID,
			Title:               quiz.Title,
			TimeLimitSeconds:    quiz.TimeLimitSeconds,
			MaxAttempts:         quiz.MaxAttempts,
			PassingScorePercent: quiz.PassingScorePercent,
			IsFree:              quiz.IsFree,
			IsLocked:            false,
			AttemptsUsed:        len(attempts),
			LatestAttempt:       toStudentAttemptResult(firstAttempt(attempts)),
		},
		ShowAnswersAfterSubmission: quiz.ShowAnswersAfterSubmission,
	}, nil
}

// GetStudentQuizAttemptResult returns a completed attempt snapshot for the student.
func (s *service) GetStudentQuizAttemptResult(ctx context.Context, quizID, attemptID, studentID uuid.UUID) (*StudentAttemptResult, error) {
	quiz, err := s.quizRepo.FindByID(ctx, quizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}

	if err := s.ensureStudentCanAccessQuiz(ctx, studentID, quiz); err != nil {
		return nil, err
	}

	attempt, err := s.attemptRepo.FindByID(ctx, attemptID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ATTEMPT_NOT_FOUND", "quiz attempt not found")
	}

	if attempt.QuizID != quizID || attempt.StudentID != studentID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to access this attempt")
	}
	if !attempt.IsSubmitted() {
		return nil, apperrors.NewValidationErrorWithDetails("ATTEMPT_IN_PROGRESS", "quiz attempt is still in progress", nil)
	}

	return toStudentAttemptResult(attempt), nil
}

// GetStudentAttempt returns a student's attempt by ID and includes its quiz ID.
func (s *service) GetStudentAttempt(ctx context.Context, attemptID, studentID uuid.UUID) (*StudentAttemptResult, error) {
	attempt, err := s.attemptRepo.FindByID(ctx, attemptID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ATTEMPT_NOT_FOUND", "quiz attempt not found")
	}
	if attempt.StudentID != studentID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to access this attempt")
	}
	quiz, err := s.quizRepo.FindByID(ctx, attempt.QuizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureStudentCanAccessQuiz(ctx, studentID, quiz); err != nil {
		return nil, err
	}
	return toStudentAttemptResult(attempt), nil
}

// StartAttempt creates a new quiz attempt for a student
func (s *service) StartAttempt(ctx context.Context, cmd StartAttemptCommand) (*QuizAttemptResponse, error) {
	// Get quiz
	quiz, err := s.quizRepo.FindByID(ctx, cmd.QuizID)
	if err != nil {
		return nil, err
	}

	if err := s.ensureStudentCanAccessQuiz(ctx, cmd.StudentID, quiz); err != nil {
		return nil, err
	}

	existingAttempts, err := s.attemptRepo.FindByQuizAndStudent(ctx, cmd.QuizID, cmd.StudentID)
	if err != nil {
		return nil, err
	}
	if latestAttempt := firstAttempt(existingAttempts); latestAttempt != nil && latestAttempt.IsInProgress() {
		if !latestAttempt.HasExpired(quiz.TimeLimitSeconds) {
			return s.buildQuizAttemptResponse(ctx, quiz, latestAttempt)
		}
		if _, err := s.AutoSubmitAttempt(ctx, latestAttempt.ID); err != nil {
			return nil, err
		}
	}

	// Count existing attempts
	attemptsUsed, err := s.attemptRepo.CountAttempts(ctx, cmd.QuizID, cmd.StudentID)
	if err != nil {
		return nil, err
	}

	// Validate attempts_used < max_attempts (0 = unlimited)
	if !quiz.CanAttempt(attemptsUsed) {
		return nil, apperrors.NewValidationErrorWithDetails("MAX_ATTEMPTS_REACHED", "maximum attempts reached for this quiz", nil)
	}

	// Create attempt record
	now := time.Now()
	attempt := &assessments.QuizAttempt{
		ID:        uuid.New(),
		QuizID:    cmd.QuizID,
		StudentID: cmd.StudentID,
		StartedAt: now,
		Status:    assessments.QuizAttemptStatusInProgress,
	}

	err = s.attemptRepo.Create(ctx, attempt)
	if err != nil {
		return nil, err
	}

	return s.buildQuizAttemptResponse(ctx, quiz, attempt)
}

func (s *service) buildQuizAttemptResponse(ctx context.Context, quiz *assessments.Quiz, attempt *assessments.QuizAttempt) (*QuizAttemptResponse, error) {
	questions, err := s.questionRepo.FindByQuizID(ctx, quiz.ID)
	if err != nil {
		return nil, err
	}

	// Shuffle questions if configured
	if quiz.ShuffleQuestions {
		questions = shuffleQuestions(questions)
	}

	// Get question IDs for batch option fetch
	questionIDs := make([]uuid.UUID, len(questions))
	for i, q := range questions {
		questionIDs[i] = q.ID
	}

	// Fetch all options for all questions
	optionsMap, err := s.optionRepo.FindByQuestionIDs(ctx, questionIDs)
	if err != nil {
		return nil, err
	}

	// Build student-facing question responses (no answer keys)
	questionResponses := make([]QuestionStudentResponse, 0, len(questions))
	for _, question := range questions {
		options := optionsMap[question.ID]
		questionResponses = append(questionResponses, s.toQuestionStudentResponse(question, options))
	}

	// Calculate expires_at if time limit is set
	var expiresAt *time.Time
	if quiz.HasTimeLimit() {
		expires := attempt.StartedAt.Add(time.Duration(quiz.TimeLimitSeconds) * time.Second)
		expiresAt = &expires
	}

	return &QuizAttemptResponse{
		ID:        attempt.ID,
		QuizID:    attempt.QuizID,
		StartedAt: attempt.StartedAt,
		ExpiresAt: expiresAt,
		Questions: questionResponses,
		Answers:   decodeDraftAnswers(attempt.DraftAnswers),
	}, nil
}

// SaveAttemptAnswers persists draft answers for an in-progress attempt.
func (s *service) SaveAttemptAnswers(ctx context.Context, cmd SaveAttemptAnswersCommand) (*QuizAttemptResponse, error) {
	attempt, err := s.attemptRepo.FindByID(ctx, cmd.AttemptID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ATTEMPT_NOT_FOUND", "quiz attempt not found")
	}
	if attempt.StudentID != cmd.StudentID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to update this attempt")
	}
	if !attempt.IsInProgress() {
		return nil, apperrors.NewValidationErrorWithDetails("ATTEMPT_ALREADY_SUBMITTED", "attempt has already been submitted", nil)
	}
	quiz, err := s.quizRepo.FindByID(ctx, attempt.QuizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureStudentCanAccessQuiz(ctx, cmd.StudentID, quiz); err != nil {
		return nil, err
	}
	if quiz.HasTimeLimit() && attempt.HasExpired(quiz.TimeLimitSeconds) {
		if _, err := s.AutoSubmitAttempt(ctx, attempt.ID); err != nil {
			return nil, err
		}
		return nil, apperrors.NewValidationErrorWithDetails("ATTEMPT_EXPIRED", "quiz attempt has expired", nil)
	}
	raw, err := json.Marshal(cmd.Answers)
	if err != nil {
		return nil, apperrors.NewSimpleValidationError("INVALID_ANSWERS", "answers must be valid JSON")
	}
	if err := s.attemptRepo.SaveDraftAnswers(ctx, attempt.ID, raw); err != nil {
		return nil, err
	}
	attempt.DraftAnswers = raw
	return s.buildQuizAttemptResponse(ctx, quiz, attempt)
}

// SubmitAttempt scores and records a quiz submission
func (s *service) SubmitAttempt(ctx context.Context, cmd SubmitAttemptCommand) (*SubmitAttemptResponse, error) {
	// Get attempt
	attempt, err := s.attemptRepo.FindByID(ctx, cmd.AttemptID)
	if err != nil {
		return nil, err
	}

	// Verify student owns this attempt
	if attempt.StudentID != cmd.StudentID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to submit this attempt")
	}

	// Verify attempt is in progress
	if !attempt.IsInProgress() {
		return nil, apperrors.NewValidationErrorWithDetails("ATTEMPT_ALREADY_SUBMITTED", "attempt has already been submitted", nil)
	}

	// Get quiz
	quiz, err := s.quizRepo.FindByID(ctx, attempt.QuizID)
	if err != nil {
		return nil, err
	}

	// Check if timer expired - auto-submit if so
	now := time.Now()
	status := assessments.QuizAttemptStatusSubmitted
	if quiz.HasTimeLimit() && attempt.HasExpired(quiz.TimeLimitSeconds) {
		status = assessments.QuizAttemptStatusAutoSubmitted
	}

	// Score the submission server-side
	scoreResult, err := s.scoreAttempt(ctx, attempt.QuizID, cmd.Answers)
	if err != nil {
		return nil, err
	}

	// Calculate time taken
	timeTaken := int(now.Sub(attempt.StartedAt).Seconds())

	// Check if passed
	passed := quiz.IsPassing(scoreResult.ScorePercent)

	// Determine points to award
	pointsAwarded := 0
	if passed {
		// Award points only on first passing attempt
		highestScore, err := s.attemptRepo.GetHighestScore(ctx, attempt.QuizID, attempt.StudentID)
		if err != nil {
			return nil, err
		}

		// Check if this is the first passing attempt
		isFirstPass := highestScore == nil || !quiz.IsPassing(*highestScore)
		if isFirstPass {
			pointsAwarded = 20 // Default points_per_quiz_pass

			// Award bonus for perfect score
			if scoreResult.ScorePercent == 100.0 {
				pointsAwarded += 10 // Default bonus_points_perfect_score
			}
		}
	}

	// Update attempt record
	attempt.SubmittedAt = &now
	attempt.ScorePercent = &scoreResult.ScorePercent
	attempt.Passed = &passed
	attempt.TimeTakenSeconds = &timeTaken
	attempt.PointsAwarded = pointsAwarded
	attempt.Status = status

	err = s.attemptRepo.Update(ctx, attempt)
	if err != nil {
		return nil, err
	}

	// Build response
	response := &SubmitAttemptResponse{
		ScorePercent:     scoreResult.ScorePercent,
		Passed:           passed,
		TimeTakenSeconds: timeTaken,
		PointsAwarded:    pointsAwarded,
	}

	// Include question results only if show_answers_after_submission is true
	if quiz.ShowAnswersAfterSubmission {
		response.QuestionResults = scoreResult.QuestionResults
	}

	return response, nil
}

// AutoSubmitAttempt is called by a worker to auto-submit expired attempts
func (s *service) AutoSubmitAttempt(ctx context.Context, attemptID uuid.UUID) (*SubmitAttemptResponse, error) {
	// Get attempt
	attempt, err := s.attemptRepo.FindByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}

	// Verify attempt is in progress
	if !attempt.IsInProgress() {
		return nil, apperrors.NewValidationErrorWithDetails("ATTEMPT_ALREADY_SUBMITTED", "attempt has already been submitted", nil)
	}

	// Get quiz
	quiz, err := s.quizRepo.FindByID(ctx, attempt.QuizID)
	if err != nil {
		return nil, err
	}

	// Verify timer has expired
	if !quiz.HasTimeLimit() || !attempt.HasExpired(quiz.TimeLimitSeconds) {
		return nil, apperrors.NewValidationErrorWithDetails("ATTEMPT_NOT_EXPIRED", "attempt has not expired", nil)
	}

	// Auto-submit with empty answers (student didn't submit in time)
	now := time.Now()
	timeTaken := int(now.Sub(attempt.StartedAt).Seconds())

	// Score with empty answers
	scoreResult, err := s.scoreAttempt(ctx, attempt.QuizID, []QuizAnswerCommand{})
	if err != nil {
		return nil, err
	}

	passed := quiz.IsPassing(scoreResult.ScorePercent)

	// Update attempt record
	attempt.SubmittedAt = &now
	attempt.ScorePercent = &scoreResult.ScorePercent
	attempt.Passed = &passed
	attempt.TimeTakenSeconds = &timeTaken
	attempt.PointsAwarded = 0 // No points for auto-submitted attempts
	attempt.Status = assessments.QuizAttemptStatusAutoSubmitted

	err = s.attemptRepo.Update(ctx, attempt)
	if err != nil {
		return nil, err
	}

	return &SubmitAttemptResponse{
		ScorePercent:     scoreResult.ScorePercent,
		Passed:           passed,
		TimeTakenSeconds: timeTaken,
		PointsAwarded:    0,
	}, nil
}

// scoreAttempt scores a quiz attempt server-side
func (s *service) scoreAttempt(ctx context.Context, quizID uuid.UUID, answers []QuizAnswerCommand) (*scoreResult, error) {
	// Get all questions for the quiz
	questions, err := s.questionRepo.FindByQuizID(ctx, quizID)
	if err != nil {
		return nil, err
	}

	// Get question IDs for batch option fetch
	questionIDs := make([]uuid.UUID, len(questions))
	for i, q := range questions {
		questionIDs[i] = q.ID
	}

	// Fetch all options for all questions
	optionsMap, err := s.optionRepo.FindByQuestionIDs(ctx, questionIDs)
	if err != nil {
		return nil, err
	}

	// Build answer map for quick lookup
	answerMap := make(map[uuid.UUID][]uuid.UUID)
	for _, answer := range answers {
		answerMap[answer.QuestionID] = answer.SelectedOptions
	}

	// Score each question
	totalQuestions := len(questions)
	correctCount := 0
	questionResults := make([]QuestionResultResponse, 0, totalQuestions)

	for _, question := range questions {
		options := optionsMap[question.ID]

		// Get correct option IDs
		correctIDs := make([]uuid.UUID, 0)
		for _, opt := range options {
			if opt.IsCorrect {
				correctIDs = append(correctIDs, opt.ID)
			}
		}

		// Get student's selected options
		selectedIDs := answerMap[question.ID]
		if selectedIDs == nil {
			selectedIDs = []uuid.UUID{}
		}

		// Check if answer is correct
		isCorrect := areAnswersEqual(selectedIDs, correctIDs)
		if isCorrect {
			correctCount++
		}

		questionResults = append(questionResults, QuestionResultResponse{
			QuestionID:      question.ID,
			IsCorrect:       isCorrect,
			SelectedOptions: selectedIDs,
			CorrectOptions:  correctIDs,
			Explanation:     question.Explanation,
		})
	}

	// Calculate score percentage
	scorePercent := 0.0
	if totalQuestions > 0 {
		scorePercent = (float64(correctCount) / float64(totalQuestions)) * 100.0
	}

	return &scoreResult{
		ScorePercent:    scorePercent,
		QuestionResults: questionResults,
	}, nil
}

// scoreResult holds the scoring result
type scoreResult struct {
	ScorePercent    float64
	QuestionResults []QuestionResultResponse
}

// areAnswersEqual checks if two sets of option IDs are equal
func areAnswersEqual(selected, correct []uuid.UUID) bool {
	if len(selected) != len(correct) {
		return false
	}

	// Create maps for comparison
	selectedMap := make(map[uuid.UUID]bool)
	for _, id := range selected {
		selectedMap[id] = true
	}

	correctMap := make(map[uuid.UUID]bool)
	for _, id := range correct {
		correctMap[id] = true
	}

	// Check if all selected are correct
	for id := range selectedMap {
		if !correctMap[id] {
			return false
		}
	}

	// Check if all correct are selected
	for id := range correctMap {
		if !selectedMap[id] {
			return false
		}
	}

	return true
}

// shuffleQuestions randomizes the order of questions
func shuffleQuestions(questions []*assessments.Question) []*assessments.Question {
	shuffled := make([]*assessments.Question, len(questions))
	copy(shuffled, questions)

	// Fisher-Yates shuffle
	for i := len(shuffled) - 1; i > 0; i-- {
		j := time.Now().UnixNano() % int64(i+1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled
}

func (s *service) toQuestionStudentResponse(question *assessments.Question, options []*assessments.QuestionOption) QuestionStudentResponse {
	optionResponses := make([]QuestionOptionStudentResponse, 0, len(options))
	for _, opt := range options {
		optionResponses = append(optionResponses, QuestionOptionStudentResponse{
			ID:          opt.ID,
			Body:        opt.Body,
			ContentType: opt.ContentType,
			ImageURL:    opt.ImageURL,
			Position:    opt.Position,
		})
	}

	return QuestionStudentResponse{
		ID:          question.ID,
		QuizID:      question.QuizID,
		Body:        question.Body,
		Type:        string(question.Type),
		ContentType: question.ContentType,
		ImageURL:    question.ImageURL,
		Marks:       question.Marks,
		IsRequired:  question.IsRequired,
		Position:    question.Position,
		Options:     optionResponses,
	}
}

// CreateAssignment creates a new assignment for a course
func (s *service) CreateAssignment(ctx context.Context, cmd CreateAssignmentCommand) (*AssignmentResponse, error) {
	// Verify course exists and teacher owns it
	course, err := s.courseRepo.FindByID(ctx, cmd.CourseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}

	if !course.IsOwnedBy(cmd.TeacherID) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to create assignment for this course")
	}

	// Parse due date
	dueAt, err := time.Parse(time.RFC3339, cmd.DueAt)
	if err != nil {
		return nil, apperrors.NewSimpleValidationError("INVALID_DATE", "invalid due_at format, expected ISO 8601")
	}

	// Validate submission type
	var submissionType assessments.SubmissionType
	switch cmd.SubmissionType {
	case "file":
		submissionType = assessments.SubmissionTypeFile
	case "text":
		submissionType = assessments.SubmissionTypeText
	case "both":
		submissionType = assessments.SubmissionTypeBoth
	default:
		return nil, apperrors.NewSimpleValidationError("INVALID_SUBMISSION_TYPE", "submission_type must be 'file', 'text', or 'both'")
	}

	// Validate total marks
	if cmd.TotalMarks <= 0 {
		return nil, apperrors.NewSimpleValidationError("INVALID_TOTAL_MARKS", "total_marks must be greater than 0")
	}

	// Validate max file size
	if cmd.MaxFileSizeMB <= 0 || cmd.MaxFileSizeMB > 50 {
		return nil, apperrors.NewSimpleValidationError("INVALID_FILE_SIZE", "max_file_size_mb must be between 1 and 50")
	}

	// Create assignment
	assignment := &assessments.Assignment{
		ID:                  uuid.New(),
		CourseID:            cmd.CourseID,
		Title:               cmd.Title,
		Description:         cmd.Description,
		DueAt:               dueAt,
		SubmissionType:      submissionType,
		MaxFileSizeMB:       cmd.MaxFileSizeMB,
		AllowLateSubmission: cmd.AllowLateSubmission,
		TotalMarks:          cmd.TotalMarks,
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	}

	if err := s.assignmentRepo.Create(ctx, assignment); err != nil {
		return nil, err
	}

	return &AssignmentResponse{
		ID:                  assignment.ID,
		CourseID:            assignment.CourseID,
		Title:               assignment.Title,
		Description:         assignment.Description,
		DueAt:               assignment.DueAt,
		SubmissionType:      string(assignment.SubmissionType),
		MaxFileSizeMB:       assignment.MaxFileSizeMB,
		AllowLateSubmission: assignment.AllowLateSubmission,
		TotalMarks:          assignment.TotalMarks,
		CreatedAt:           assignment.CreatedAt,
		UpdatedAt:           assignment.UpdatedAt,
	}, nil
}

// ListTeacherCourseAssignments returns assignment metadata for a teacher-owned course.
func (s *service) ListTeacherCourseAssignments(ctx context.Context, courseID, teacherID uuid.UUID) ([]AssignmentResponse, error) {
	course, err := s.courseRepo.FindByID(ctx, courseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}
	if !course.IsOwnedBy(teacherID) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to view assignments for this course")
	}

	assignments, err := s.assignmentRepo.FindByCourseID(ctx, courseID)
	if err != nil {
		return nil, err
	}

	responses := make([]AssignmentResponse, 0, len(assignments))
	for _, assignment := range assignments {
		responses = append(responses, *toAssignmentResponse(assignment))
	}
	return responses, nil
}

// GetTeacherAssignment returns assignment metadata for a teacher-owned assignment.
func (s *service) GetTeacherAssignment(ctx context.Context, assignmentID, teacherID uuid.UUID) (*AssignmentResponse, error) {
	assignment, err := s.assignmentRepo.FindByID(ctx, assignmentID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ASSIGNMENT_NOT_FOUND", "assignment not found")
	}
	if err := s.ensureTeacherOwnsCourse(ctx, assignment.CourseID, teacherID); err != nil {
		return nil, err
	}
	return toAssignmentResponse(assignment), nil
}

// UpdateAssignment replaces editable assignment metadata for a teacher-owned assignment.
func (s *service) UpdateAssignment(ctx context.Context, cmd UpdateAssignmentCommand) (*AssignmentResponse, error) {
	assignment, err := s.assignmentRepo.FindByID(ctx, cmd.AssignmentID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ASSIGNMENT_NOT_FOUND", "assignment not found")
	}
	if err := s.ensureTeacherOwnsCourse(ctx, assignment.CourseID, cmd.TeacherID); err != nil {
		return nil, err
	}

	dueAt, err := time.Parse(time.RFC3339, cmd.DueAt)
	if err != nil {
		return nil, apperrors.NewSimpleValidationError("INVALID_DATE", "invalid due_at format, expected ISO 8601")
	}

	var submissionType assessments.SubmissionType
	switch cmd.SubmissionType {
	case "file":
		submissionType = assessments.SubmissionTypeFile
	case "text":
		submissionType = assessments.SubmissionTypeText
	case "both":
		submissionType = assessments.SubmissionTypeBoth
	default:
		return nil, apperrors.NewSimpleValidationError("INVALID_SUBMISSION_TYPE", "submission_type must be 'file', 'text', or 'both'")
	}

	if strings.TrimSpace(cmd.Title) == "" {
		return nil, apperrors.NewSimpleValidationError("TITLE_REQUIRED", "title is required")
	}
	if cmd.TotalMarks <= 0 {
		return nil, apperrors.NewSimpleValidationError("INVALID_TOTAL_MARKS", "total_marks must be greater than 0")
	}
	if cmd.MaxFileSizeMB <= 0 || cmd.MaxFileSizeMB > 50 {
		return nil, apperrors.NewSimpleValidationError("INVALID_FILE_SIZE", "max_file_size_mb must be between 1 and 50")
	}

	assignment.Title = strings.TrimSpace(cmd.Title)
	assignment.Description = strings.TrimSpace(cmd.Description)
	assignment.DueAt = dueAt
	assignment.SubmissionType = submissionType
	assignment.MaxFileSizeMB = cmd.MaxFileSizeMB
	assignment.AllowLateSubmission = cmd.AllowLateSubmission
	assignment.TotalMarks = cmd.TotalMarks
	assignment.UpdatedAt = time.Now().UTC()

	if err := s.assignmentRepo.Update(ctx, assignment); err != nil {
		return nil, err
	}
	return toAssignmentResponse(assignment), nil
}

// ListTeacherAssignmentSubmissions returns a paginated grading queue for a teacher-owned assignment.
func (s *service) ListTeacherAssignmentSubmissions(ctx context.Context, assignmentID, teacherID uuid.UUID, page, limit int) (*TeacherAssignmentSubmissionListResponse, error) {
	assignment, err := s.assignmentRepo.FindByID(ctx, assignmentID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ASSIGNMENT_NOT_FOUND", "assignment not found")
	}

	course, err := s.courseRepo.FindByID(ctx, assignment.CourseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}
	if !course.IsOwnedBy(teacherID) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to grade submissions for this assignment")
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	submissions, total, err := s.submissionRepo.FindByAssignmentID(ctx, assignmentID, page, limit)
	if err != nil {
		return nil, err
	}

	response := &TeacherAssignmentSubmissionListResponse{
		Assignment:  *toAssignmentResponse(assignment),
		Submissions: make([]AssignmentSubmissionResponse, 0, len(submissions)),
	}
	response.Meta.Page = page
	response.Meta.Limit = limit
	response.Meta.Total = total
	response.Meta.TotalPages = (total + limit - 1) / limit

	for _, submission := range submissions {
		submissionResponse, err := s.buildAssignmentSubmissionResponse(ctx, submission)
		if err != nil {
			return nil, err
		}
		response.Submissions = append(response.Submissions, *submissionResponse)
	}

	return response, nil
}

// GetTeacherAssignmentSubmission returns a single teacher-visible submission detail.
func (s *service) GetTeacherAssignmentSubmission(ctx context.Context, assignmentID, submissionID, teacherID uuid.UUID) (*AssignmentSubmissionResponse, error) {
	assignment, err := s.assignmentRepo.FindByID(ctx, assignmentID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ASSIGNMENT_NOT_FOUND", "assignment not found")
	}

	course, err := s.courseRepo.FindByID(ctx, assignment.CourseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}
	if !course.IsOwnedBy(teacherID) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to grade this submission")
	}

	submission, err := s.submissionRepo.FindByID(ctx, submissionID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("SUBMISSION_NOT_FOUND", "submission not found")
	}
	if submission.AssignmentID != assignmentID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "submission does not belong to this assignment")
	}

	return s.buildAssignmentSubmissionResponse(ctx, submission)
}

// ListStudentAssignments returns assignments across a student's enrolled courses with submission status.
func (s *service) ListStudentAssignments(ctx context.Context, studentID uuid.UUID) ([]StudentAssignmentDetailResponse, error) {
	courseIDs, err := s.listStudentCourseIDs(ctx, studentID)
	if err != nil {
		return nil, err
	}

	responses := make([]StudentAssignmentDetailResponse, 0)
	for _, courseID := range courseIDs {
		assignments, err := s.assignmentRepo.FindByCourseID(ctx, courseID)
		if err != nil {
			return nil, err
		}

		for _, assignment := range assignments {
			detail, err := s.buildStudentAssignmentDetail(ctx, assignment, studentID)
			if err != nil {
				return nil, err
			}
			responses = append(responses, *detail)
		}
	}

	return responses, nil
}

// GetStudentAssignmentDetail returns a single assignment plus the student's latest submission.
func (s *service) GetStudentAssignmentDetail(ctx context.Context, assignmentID, studentID uuid.UUID) (*StudentAssignmentDetailResponse, error) {
	assignment, err := s.assignmentRepo.FindByID(ctx, assignmentID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ASSIGNMENT_NOT_FOUND", "assignment not found")
	}

	if err := s.ensureStudentEnrolledInCourse(ctx, studentID, assignment.CourseID); err != nil {
		return nil, err
	}

	return s.buildStudentAssignmentDetail(ctx, assignment, studentID)
}

// SubmitAssignment submits an assignment (draft or final)
func (s *service) SubmitAssignment(ctx context.Context, cmd SubmitAssignmentCommand) (*AssignmentSubmissionResponse, error) {
	// Fetch assignment
	assignment, err := s.assignmentRepo.FindByID(ctx, cmd.AssignmentID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ASSIGNMENT_NOT_FOUND", "assignment not found")
	}

	if err := s.ensureStudentEnrolledInCourse(ctx, cmd.StudentID, assignment.CourseID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	isLate := assignment.IsPastDeadline(now)

	// Check deadline if not a draft and not allowing late submissions
	if !cmd.IsDraft && isLate && !assignment.CanSubmitLate() {
		return nil, apperrors.NewSimpleValidationError("PAST_DEADLINE", "assignment deadline has passed and late submissions are not allowed")
	}

	// Validate submission type
	hasFiles := len(cmd.Files) > 0
	cmd.TextContent = strings.TrimSpace(cmd.TextContent)
	hasText := cmd.TextContent != ""

	if !assignment.AcceptsFileSubmissions() && hasFiles {
		return nil, apperrors.NewSimpleValidationError("FILES_NOT_ALLOWED", "this assignment does not accept file submissions")
	}

	if !assignment.AcceptsTextSubmissions() && hasText {
		return nil, apperrors.NewSimpleValidationError("TEXT_NOT_ALLOWED", "this assignment does not accept text submissions")
	}

	// Validate file count (max 5 files)
	if len(cmd.Files) > 5 {
		return nil, apperrors.NewSimpleValidationError("TOO_MANY_FILES", "maximum 5 files allowed per submission")
	}

	// Validate file sizes
	for _, file := range cmd.Files {
		if !assignment.ValidateFileSize(file.SizeBytes) {
			return nil, apperrors.NewSimpleValidationError("FILE_TOO_LARGE", "one or more files exceed the maximum allowed size")
		}
	}

	// Check for an existing draft or submission before deciding whether this is
	// an update, a finalization, or a revision.
	var submission *assessments.AssignmentSubmission
	existing, _ := s.submissionRepo.FindByAssignmentAndStudent(ctx, cmd.AssignmentID, cmd.StudentID)
	if existing != nil {
		if !existing.IsDraft() && !existing.CanResubmit() {
			return nil, apperrors.NewSimpleValidationError("ALREADY_SUBMITTED", "assignment already submitted and revision not requested")
		}

		submission = existing
		submission.TextContent = cmd.TextContent
		submission.UpdatedAt = now

		if !cmd.IsDraft {
			submission.Status = assessments.AssignmentSubmissionStatusSubmitted
			submission.SubmittedAt = &now
			submission.IsLate = isLate
		}
	}

	hasExistingFiles := false
	if submission != nil && !hasFiles {
		existingFiles, err := s.submissionFileRepo.FindBySubmissionID(ctx, submission.ID)
		if err != nil {
			return nil, err
		}
		hasExistingFiles = len(existingFiles) > 0
	}
	if !cmd.IsDraft && !hasText && !hasFiles && !hasExistingFiles {
		return nil, apperrors.NewSimpleValidationError("EMPTY_SUBMISSION", "add a written response or at least one file before submitting")
	}

	// Create new submission if none exists
	if submission == nil {
		status := assessments.AssignmentSubmissionStatusDraft
		var submittedAt *time.Time
		if !cmd.IsDraft {
			status = assessments.AssignmentSubmissionStatusSubmitted
			submittedAt = &now
		}

		submission = &assessments.AssignmentSubmission{
			ID:           uuid.New(),
			AssignmentID: cmd.AssignmentID,
			StudentID:    cmd.StudentID,
			Status:       status,
			TextContent:  cmd.TextContent,
			SubmittedAt:  submittedAt,
			IsLate:       isLate && !cmd.IsDraft,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := s.submissionRepo.Create(ctx, submission); err != nil {
			return nil, err
		}
	} else {
		if err := s.submissionRepo.Update(ctx, submission); err != nil {
			return nil, err
		}
	}

	// Upload files to RustFS if provided
	var submissionFiles []*assessments.SubmissionFile
	if hasFiles {
		for _, fileCmd := range cmd.Files {
			// Generate unique key for RustFS
			fileKey := fmt.Sprintf("assignments/%s/%s/%s", cmd.AssignmentID.String(), submission.ID.String(), uuid.New().String())

			// Upload to RustFS
			reader := bytes.NewReader(fileCmd.Content)
			if err := s.storageClient.PutObject(ctx, s.filesBucket, fileKey, reader, fileCmd.SizeBytes, fileCmd.MimeType); err != nil {
				return nil, fmt.Errorf("failed to upload file: %w", err)
			}

			// Create submission file record
			submissionFile := &assessments.SubmissionFile{
				ID:               uuid.New(),
				SubmissionID:     submission.ID,
				RustFSKey:        fileKey,
				OriginalFilename: fileCmd.OriginalFilename,
				MimeType:         fileCmd.MimeType,
				SizeBytes:        fileCmd.SizeBytes,
			}
			submissionFiles = append(submissionFiles, submissionFile)
		}

		// Save all files in batch
		if err := s.submissionFileRepo.CreateBatch(ctx, submissionFiles); err != nil {
			return nil, err
		}
	}

	return s.buildAssignmentSubmissionResponse(ctx, submission)
}

// GradeSubmission grades an assignment submission
func (s *service) GradeSubmission(ctx context.Context, cmd GradeSubmissionCommand) (*SubmissionGradeResponse, error) {
	// Fetch submission
	submission, err := s.submissionRepo.FindByID(ctx, cmd.SubmissionID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("SUBMISSION_NOT_FOUND", "submission not found")
	}

	// Fetch assignment to validate score
	assignment, err := s.assignmentRepo.FindByID(ctx, submission.AssignmentID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ASSIGNMENT_NOT_FOUND", "assignment not found")
	}

	// Validate score
	if cmd.Score < 0 || cmd.Score > assignment.TotalMarks {
		return nil, apperrors.NewSimpleValidationError("INVALID_SCORE", "score must be between 0 and total_marks")
	}

	// Verify grader is authorized (teacher owns the course)
	course, err := s.courseRepo.FindByID(ctx, assignment.CourseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}

	if !course.IsOwnedBy(cmd.GradedBy) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to grade this submission")
	}

	now := time.Now().UTC()

	// Create grade record (append-only)
	grade := &assessments.SubmissionGrade{
		ID:                uuid.New(),
		SubmissionID:      cmd.SubmissionID,
		GradedBy:          cmd.GradedBy,
		Score:             cmd.Score,
		Feedback:          cmd.Feedback,
		RevisionRequested: cmd.RevisionRequested,
		RevisionNotes:     cmd.RevisionNotes,
		GradedAt:          now,
	}

	if err := s.gradeRepo.Create(ctx, grade); err != nil {
		return nil, err
	}

	// Update submission status
	if cmd.RevisionRequested {
		submission.Status = assessments.AssignmentSubmissionStatusRevisionRequested
	} else {
		submission.Status = assessments.AssignmentSubmissionStatusGraded
	}
	submission.UpdatedAt = now

	if err := s.submissionRepo.Update(ctx, submission); err != nil {
		return nil, err
	}

	// Send notification to student
	notificationTitle := "Assignment Graded"
	notificationBody := fmt.Sprintf("Your submission for '%s' has been graded. Score: %.2f/%.2f", assignment.Title, cmd.Score, assignment.TotalMarks)
	if cmd.RevisionRequested {
		notificationTitle = "Revision Requested"
		notificationBody = fmt.Sprintf("Your submission for '%s' requires revision. Please review the feedback and resubmit.", assignment.Title)
	}

	if err := s.notificationQueue.EnqueueNotification(ctx, submission.StudentID, "assignment_graded", notificationTitle, notificationBody); err != nil {
		// Log error but don't fail the request
		// In production, use proper logger
	}

	return &SubmissionGradeResponse{
		ID:                grade.ID,
		Score:             grade.Score,
		Feedback:          grade.Feedback,
		RevisionRequested: grade.RevisionRequested,
		RevisionNotes:     grade.RevisionNotes,
		GradedBy:          grade.GradedBy,
		GradedAt:          grade.GradedAt,
	}, nil
}

func (s *service) listStudentCourseIDs(ctx context.Context, studentID uuid.UUID) ([]uuid.UUID, error) {
	page := 1
	limit := 100
	courseIDs := make([]uuid.UUID, 0)

	for {
		enrollmentList, total, err := s.enrollmentRepo.FindByStudentID(ctx, studentID, page, limit)
		if err != nil {
			return nil, err
		}

		for _, enrollment := range enrollmentList {
			if enrollment != nil && enrollment.Status == domainenrollments.EnrollmentStatusActive {
				courseIDs = append(courseIDs, enrollment.CourseID)
			}
		}

		if page*limit >= total || len(enrollmentList) == 0 {
			break
		}
		page++
	}

	return courseIDs, nil
}

func (s *service) ensureStudentEnrolledInCourse(ctx context.Context, studentID, courseID uuid.UUID) error {
	if s.enrollmentRepo == nil {
		return nil
	}
	enrollment, err := s.enrollmentRepo.FindByStudentAndCourse(ctx, studentID, courseID)
	if err != nil || enrollment == nil {
		return apperrors.NewForbiddenError("NOT_ENROLLED", "you are not enrolled in this course")
	}
	if enrollment.Status != domainenrollments.EnrollmentStatusActive {
		return apperrors.NewForbiddenError("NOT_ENROLLED", "you are not enrolled in this course")
	}
	return nil
}

func (s *service) ensureStudentCanAccessQuiz(ctx context.Context, studentID uuid.UUID, quiz *assessments.Quiz) error {
	if quiz == nil {
		return apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if quiz.IsFree && quiz.IsPublished {
		return nil
	}
	return s.ensureStudentEnrolledInCourse(ctx, studentID, quiz.CourseID)
}

func firstAttempt(attempts []*assessments.QuizAttempt) *assessments.QuizAttempt {
	if len(attempts) == 0 {
		return nil
	}
	return attempts[0]
}

func toStudentAttemptResult(attempt *assessments.QuizAttempt) *StudentAttemptResult {
	if attempt == nil {
		return nil
	}

	return &StudentAttemptResult{
		ID:               attempt.ID,
		QuizID:           attempt.QuizID,
		Status:           string(attempt.Status),
		StartedAt:        attempt.StartedAt,
		SubmittedAt:      attempt.SubmittedAt,
		ScorePercent:     attempt.ScorePercent,
		Passed:           attempt.Passed,
		TimeTakenSeconds: attempt.TimeTakenSeconds,
		PointsAwarded:    attempt.PointsAwarded,
	}
}

func decodeDraftAnswers(raw []byte) map[string]interface{} {
	if len(raw) == 0 {
		return nil
	}
	var answers map[string]interface{}
	if err := json.Unmarshal(raw, &answers); err != nil {
		return nil
	}
	return answers
}

func toAssignmentResponse(assignment *assessments.Assignment) *AssignmentResponse {
	if assignment == nil {
		return nil
	}

	return &AssignmentResponse{
		ID:                  assignment.ID,
		CourseID:            assignment.CourseID,
		Title:               assignment.Title,
		Description:         assignment.Description,
		DueAt:               assignment.DueAt,
		SubmissionType:      string(assignment.SubmissionType),
		MaxFileSizeMB:       assignment.MaxFileSizeMB,
		AllowLateSubmission: assignment.AllowLateSubmission,
		TotalMarks:          assignment.TotalMarks,
		CreatedAt:           assignment.CreatedAt,
		UpdatedAt:           assignment.UpdatedAt,
	}
}

func (s *service) buildStudentAssignmentDetail(ctx context.Context, assignment *assessments.Assignment, studentID uuid.UUID) (*StudentAssignmentDetailResponse, error) {
	submission, err := s.submissionRepo.FindByAssignmentAndStudent(ctx, assignment.ID, studentID)
	if err != nil {
		return nil, err
	}

	var submissionResponse *AssignmentSubmissionResponse
	if submission != nil {
		submissionResponse, err = s.buildAssignmentSubmissionResponse(ctx, submission)
		if err != nil {
			return nil, err
		}
	}

	return &StudentAssignmentDetailResponse{
		AssignmentResponse: *toAssignmentResponse(assignment),
		Submission:         submissionResponse,
	}, nil
}

func (s *service) buildAssignmentSubmissionResponse(ctx context.Context, submission *assessments.AssignmentSubmission) (*AssignmentSubmissionResponse, error) {
	allFiles, err := s.submissionFileRepo.FindBySubmissionID(ctx, submission.ID)
	if err != nil {
		return nil, err
	}

	fileResponses := make([]SubmissionFileResponse, 0, len(allFiles))
	for _, file := range allFiles {
		downloadURL := ""
		if s.storageClient != nil && s.filesBucket != "" {
			presignedURL, err := s.storageClient.PresignGetURL(ctx, s.filesBucket, file.RustFSKey, 2*time.Hour)
			if err != nil {
				return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
			}
			downloadURL = presignedURL
		}

		fileResponses = append(fileResponses, SubmissionFileResponse{
			ID:               file.ID,
			OriginalFilename: file.OriginalFilename,
			MimeType:         file.MimeType,
			SizeBytes:        file.SizeBytes,
			DownloadURL:      downloadURL,
		})
	}

	var latestGradeResponse *SubmissionGradeResponse
	latestGrade, err := s.gradeRepo.GetLatestGrade(ctx, submission.ID)
	if err == nil && latestGrade != nil {
		latestGradeResponse = &SubmissionGradeResponse{
			ID:                latestGrade.ID,
			Score:             latestGrade.Score,
			Feedback:          latestGrade.Feedback,
			RevisionRequested: latestGrade.RevisionRequested,
			RevisionNotes:     latestGrade.RevisionNotes,
			GradedBy:          latestGrade.GradedBy,
			GradedAt:          latestGrade.GradedAt,
		}
	}

	return &AssignmentSubmissionResponse{
		ID:           submission.ID,
		AssignmentID: submission.AssignmentID,
		StudentID:    submission.StudentID,
		Status:       string(submission.Status),
		TextContent:  submission.TextContent,
		Files:        fileResponses,
		LatestGrade:  latestGradeResponse,
		SubmittedAt:  submission.SubmittedAt,
		IsLate:       submission.IsLate,
		CreatedAt:    submission.CreatedAt,
		UpdatedAt:    submission.UpdatedAt,
	}, nil
}
