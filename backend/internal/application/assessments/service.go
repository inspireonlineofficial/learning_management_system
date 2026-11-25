package assessments

import (
	"bytes"
	"context"
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
	// AutoSubmitExpiredAttempts finalises every in-progress attempt whose
	// deadline has passed (called periodically by a background worker).
	AutoSubmitExpiredAttempts(ctx context.Context) (int, error)
	CheckAnswer(ctx context.Context, cmd CheckAnswerCommand) (*CheckAnswerResponse, error)
	UploadAnswerImage(ctx context.Context, cmd UploadAnswerImageCommand) (*UploadAnswerImageResponse, error)

	// Teacher operations - monitoring, grading and analytics
	ListQuizAttempts(ctx context.Context, quizID, teacherID uuid.UUID, filter AttemptListFilter) (*QuizAttemptsResponse, error)
	GetTeacherAttempt(ctx context.Context, quizID, attemptID, teacherID uuid.UUID) (*TeacherAttemptDetailResponse, error)
	GradeAttempt(ctx context.Context, cmd GradeAttemptCommand) (*TeacherAttemptDetailResponse, error)
	GetQuizAnalytics(ctx context.Context, quizID, teacherID uuid.UUID) (*QuizAnalyticsResponse, error)
	DuplicateQuiz(ctx context.Context, quizID, teacherID uuid.UUID) (*QuizTeacherResponse, error)

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
	txRunner           TxRunner
	passNotifier       QuizPassNotifier
	now                func() time.Time
}

// TxRunner runs fn in a database transaction (see postgres.TxRunner).
type TxRunner interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// AttemptLocker serialises attempt creation per student and quiz so double
// clicks cannot create two attempts or exceed the attempt limit.
type AttemptLocker interface {
	LockStudentQuiz(ctx context.Context, quizID, studentID uuid.UUID) error
}

// QuizPassNotifier is told when a student passes a quiz with final marks
// (it awards gamification points; the points service de-duplicates).
type QuizPassNotifier interface {
	QuizPassed(ctx context.Context, studentID, quizID uuid.UUID, scorePercent float64) error
}

// SetTxRunner enables transactional quiz edits and serialised attempt starts.
func (s *service) SetTxRunner(runner TxRunner) { s.txRunner = runner }

// SetQuizPassNotifier wires quiz passes to the points system.
func (s *service) SetQuizPassNotifier(n QuizPassNotifier) { s.passNotifier = n }

// Configurer exposes the optional setters on the concrete service.
type Configurer interface {
	SetTxRunner(runner TxRunner)
	SetQuizPassNotifier(n QuizPassNotifier)
}

func (s *service) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

// inTx runs fn in a transaction when a TxRunner is configured.
func (s *service) inTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if s.txRunner == nil {
		return fn(ctx)
	}
	return s.txRunner.RunInTx(ctx, fn)
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

	// Drafts are work in progress; they can only be graded once submitted.
	if submission.IsDraft() {
		return nil, apperrors.NewSimpleValidationError("SUBMISSION_NOT_SUBMITTED", "draft submissions cannot be graded")
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
