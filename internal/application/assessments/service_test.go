package assessments

import (
	"context"
	"testing"
	"time"

	"lms-backend/internal/domain/assessments"
	"lms-backend/internal/domain/courses"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// Mock repositories for testing

type mockQuizRepo struct {
	quizzes map[uuid.UUID]*assessments.Quiz
}

func newMockQuizRepo() *mockQuizRepo {
	return &mockQuizRepo{
		quizzes: make(map[uuid.UUID]*assessments.Quiz),
	}
}

func (m *mockQuizRepo) Create(ctx context.Context, quiz *assessments.Quiz) error {
	m.quizzes[quiz.ID] = quiz
	return nil
}

func (m *mockQuizRepo) FindByID(ctx context.Context, id uuid.UUID) (*assessments.Quiz, error) {
	if quiz, exists := m.quizzes[id]; exists {
		return quiz, nil
	}
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockQuizRepo) FindByCourseID(ctx context.Context, courseID uuid.UUID) ([]*assessments.Quiz, error) {
	var result []*assessments.Quiz
	for _, q := range m.quizzes {
		if q.CourseID == courseID {
			result = append(result, q)
		}
	}
	return result, nil
}

func (m *mockQuizRepo) FindByLessonID(ctx context.Context, lessonID uuid.UUID) ([]*assessments.Quiz, error) {
	var result []*assessments.Quiz
	for _, q := range m.quizzes {
		if q.LessonID != nil && *q.LessonID == lessonID {
			result = append(result, q)
		}
	}
	return result, nil
}

func (m *mockQuizRepo) Update(ctx context.Context, quiz *assessments.Quiz) error {
	if _, exists := m.quizzes[quiz.ID]; !exists {
		return &apperrors.AppError{Code: "NOT_FOUND"}
	}
	m.quizzes[quiz.ID] = quiz
	return nil
}

func (m *mockQuizRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.quizzes, id)
	return nil
}

type mockQuestionRepo struct {
	questions map[uuid.UUID]*assessments.Question
}

func newMockQuestionRepo() *mockQuestionRepo {
	return &mockQuestionRepo{
		questions: make(map[uuid.UUID]*assessments.Question),
	}
}

func (m *mockQuestionRepo) Create(ctx context.Context, question *assessments.Question) error {
	m.questions[question.ID] = question
	return nil
}

func (m *mockQuestionRepo) CreateBatch(ctx context.Context, questions []*assessments.Question) error {
	for _, q := range questions {
		m.questions[q.ID] = q
	}
	return nil
}

func (m *mockQuestionRepo) FindByID(ctx context.Context, id uuid.UUID) (*assessments.Question, error) {
	if q, exists := m.questions[id]; exists {
		return q, nil
	}
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockQuestionRepo) FindByQuizID(ctx context.Context, quizID uuid.UUID) ([]*assessments.Question, error) {
	var result []*assessments.Question
	for _, q := range m.questions {
		if q.QuizID == quizID {
			result = append(result, q)
		}
	}
	return result, nil
}

func (m *mockQuestionRepo) Update(ctx context.Context, question *assessments.Question) error {
	if _, exists := m.questions[question.ID]; !exists {
		return &apperrors.AppError{Code: "NOT_FOUND"}
	}
	m.questions[question.ID] = question
	return nil
}

func (m *mockQuestionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.questions, id)
	return nil
}

type mockQuestionOptionRepo struct {
	options map[uuid.UUID]*assessments.QuestionOption
}

func newMockQuestionOptionRepo() *mockQuestionOptionRepo {
	return &mockQuestionOptionRepo{
		options: make(map[uuid.UUID]*assessments.QuestionOption),
	}
}

func (m *mockQuestionOptionRepo) Create(ctx context.Context, option *assessments.QuestionOption) error {
	m.options[option.ID] = option
	return nil
}

func (m *mockQuestionOptionRepo) CreateBatch(ctx context.Context, options []*assessments.QuestionOption) error {
	for _, opt := range options {
		m.options[opt.ID] = opt
	}
	return nil
}

func (m *mockQuestionOptionRepo) FindByQuestionID(ctx context.Context, questionID uuid.UUID) ([]*assessments.QuestionOption, error) {
	var result []*assessments.QuestionOption
	for _, opt := range m.options {
		if opt.QuestionID == questionID {
			result = append(result, opt)
		}
	}
	return result, nil
}

func (m *mockQuestionOptionRepo) FindByQuestionIDs(ctx context.Context, questionIDs []uuid.UUID) (map[uuid.UUID][]*assessments.QuestionOption, error) {
	result := make(map[uuid.UUID][]*assessments.QuestionOption)
	for _, qID := range questionIDs {
		for _, opt := range m.options {
			if opt.QuestionID == qID {
				result[qID] = append(result[qID], opt)
			}
		}
	}
	return result, nil
}

func (m *mockQuestionOptionRepo) Update(ctx context.Context, option *assessments.QuestionOption) error {
	if _, exists := m.options[option.ID]; !exists {
		return &apperrors.AppError{Code: "NOT_FOUND"}
	}
	m.options[option.ID] = option
	return nil
}

func (m *mockQuestionOptionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.options, id)
	return nil
}

type mockCourseRepo struct {
	courses map[uuid.UUID]*courses.Course
}

func newMockCourseRepo() *mockCourseRepo {
	return &mockCourseRepo{
		courses: make(map[uuid.UUID]*courses.Course),
	}
}

func (m *mockCourseRepo) Create(ctx context.Context, course *courses.Course) error {
	m.courses[course.ID] = course
	return nil
}

func (m *mockCourseRepo) FindByID(ctx context.Context, id uuid.UUID) (*courses.Course, error) {
	if c, exists := m.courses[id]; exists {
		return c, nil
	}
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockCourseRepo) FindBySlug(ctx context.Context, slug string) (*courses.Course, error) {
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockCourseRepo) FindByTeacherID(ctx context.Context, teacherID uuid.UUID, page, limit int) ([]*courses.Course, int, error) {
	return nil, 0, nil
}

func (m *mockCourseRepo) Update(ctx context.Context, course *courses.Course) error {
	return nil
}

func (m *mockCourseRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockCourseRepo) List(ctx context.Context, filters courses.CourseFilters, page, limit int) ([]*courses.Course, int, error) {
	return nil, 0, nil
}

func (m *mockCourseRepo) CountPublishedLessons(ctx context.Context, courseID uuid.UUID) (int, error) {
	return 0, nil
}

type mockQuizAttemptRepo struct {
	attempts map[uuid.UUID]*assessments.QuizAttempt
}

func newMockQuizAttemptRepo() *mockQuizAttemptRepo {
	return &mockQuizAttemptRepo{
		attempts: make(map[uuid.UUID]*assessments.QuizAttempt),
	}
}

func (m *mockQuizAttemptRepo) Create(ctx context.Context, attempt *assessments.QuizAttempt) error {
	m.attempts[attempt.ID] = attempt
	return nil
}

func (m *mockQuizAttemptRepo) FindByID(ctx context.Context, id uuid.UUID) (*assessments.QuizAttempt, error) {
	if a, exists := m.attempts[id]; exists {
		return a, nil
	}
	return nil, &apperrors.AppError{Code: "ATTEMPT_NOT_FOUND"}
}

func (m *mockQuizAttemptRepo) FindByQuizAndStudent(ctx context.Context, quizID, studentID uuid.UUID) ([]*assessments.QuizAttempt, error) {
	var result []*assessments.QuizAttempt
	for _, a := range m.attempts {
		if a.QuizID == quizID && a.StudentID == studentID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockQuizAttemptRepo) CountAttempts(ctx context.Context, quizID, studentID uuid.UUID) (int, error) {
	count := 0
	for _, a := range m.attempts {
		if a.QuizID == quizID && a.StudentID == studentID {
			count++
		}
	}
	return count, nil
}

func (m *mockQuizAttemptRepo) GetHighestScore(ctx context.Context, quizID, studentID uuid.UUID) (*float64, error) {
	var highest *float64
	for _, a := range m.attempts {
		if a.QuizID == quizID && a.StudentID == studentID && a.ScorePercent != nil {
			if highest == nil || *a.ScorePercent > *highest {
				score := *a.ScorePercent
				highest = &score
			}
		}
	}
	return highest, nil
}

func (m *mockQuizAttemptRepo) Update(ctx context.Context, attempt *assessments.QuizAttempt) error {
	if _, exists := m.attempts[attempt.ID]; !exists {
		return &apperrors.AppError{Code: "ATTEMPT_NOT_FOUND"}
	}
	m.attempts[attempt.ID] = attempt
	return nil
}

func (m *mockQuizAttemptRepo) SaveDraftAnswers(ctx context.Context, attemptID uuid.UUID, draftAnswers []byte) error {
	attempt, exists := m.attempts[attemptID]
	if !exists {
		return &apperrors.AppError{Code: "ATTEMPT_NOT_FOUND"}
	}
	attempt.DraftAnswers = draftAnswers
	return nil
}

func (m *mockQuizAttemptRepo) FindInProgressAttempts(ctx context.Context) ([]*assessments.QuizAttempt, error) {
	var result []*assessments.QuizAttempt
	for _, a := range m.attempts {
		if a.Status == assessments.QuizAttemptStatusInProgress {
			result = append(result, a)
		}
	}
	return result, nil
}

// Tests

func TestCreateQuiz_Success(t *testing.T) {
	quizRepo := newMockQuizRepo()
	questionRepo := newMockQuestionRepo()
	optionRepo := newMockQuestionOptionRepo()
	attemptRepo := newMockQuizAttemptRepo()
	courseRepo := newMockCourseRepo()

	teacherID := uuid.New()
	courseID := uuid.New()

	// Setup: create a course owned by teacher
	course := &courses.Course{
		ID:        courseID,
		TeacherID: teacherID,
		Title:     "Test Course",
		Status:    courses.CourseStatusDraft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	courseRepo.Create(context.Background(), course)

	svc := NewService(quizRepo, questionRepo, optionRepo, attemptRepo, newMockAssignmentRepo(), newMockSubmissionRepo(), newMockSubmissionFileRepo(), newMockGradeRepo(), courseRepo, &mockStorageClientProp{}, &mockNotificationQueueProp{}, "test-bucket")

	cmd := CreateQuizCommand{
		CourseID:                   courseID,
		TeacherID:                  teacherID,
		Title:                      "Test Quiz",
		TimeLimitSeconds:           1800,
		MaxAttempts:                3,
		PassingScorePercent:        60.0,
		ShuffleQuestions:           true,
		ShowAnswersAfterSubmission: true,
		Questions: []CreateQuestionCommand{
			{
				Body:        "What is 2+2?",
				Type:        "single",
				Position:    1,
				Explanation: "Basic arithmetic",
				Options: []CreateQuestionOptionCommand{
					{Body: "3", IsCorrect: false, Position: 1},
					{Body: "4", IsCorrect: true, Position: 2},
					{Body: "5", IsCorrect: false, Position: 3},
				},
			},
		},
	}

	resp, err := svc.CreateQuiz(context.Background(), cmd)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.Title != "Test Quiz" {
		t.Errorf("Expected title 'Test Quiz', got %s", resp.Title)
	}

	if resp.TimeLimitSeconds != 1800 {
		t.Errorf("Expected time limit 1800, got %d", resp.TimeLimitSeconds)
	}

	if resp.PassingScorePercent != 60.0 {
		t.Errorf("Expected passing score 60.0, got %f", resp.PassingScorePercent)
	}
}

func TestCreateQuiz_CourseNotFound(t *testing.T) {
	quizRepo := newMockQuizRepo()
	questionRepo := newMockQuestionRepo()
	optionRepo := newMockQuestionOptionRepo()
	attemptRepo := newMockQuizAttemptRepo()
	courseRepo := newMockCourseRepo()

	svc := NewService(quizRepo, questionRepo, optionRepo, attemptRepo, newMockAssignmentRepo(), newMockSubmissionRepo(), newMockSubmissionFileRepo(), newMockGradeRepo(), courseRepo, &mockStorageClientProp{}, &mockNotificationQueueProp{}, "test-bucket")

	cmd := CreateQuizCommand{
		CourseID:  uuid.New(),
		TeacherID: uuid.New(),
		Title:     "Test Quiz",
	}

	_, err := svc.CreateQuiz(context.Background(), cmd)

	if err == nil {
		t.Fatal("Expected error for non-existent course")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok || appErr.Code != "COURSE_NOT_FOUND" {
		t.Errorf("Expected COURSE_NOT_FOUND error, got %v", err)
	}
}

func TestCreateQuiz_NotAuthorized(t *testing.T) {
	quizRepo := newMockQuizRepo()
	questionRepo := newMockQuestionRepo()
	optionRepo := newMockQuestionOptionRepo()
	attemptRepo := newMockQuizAttemptRepo()
	courseRepo := newMockCourseRepo()

	teacherID := uuid.New()
	otherTeacherID := uuid.New()
	courseID := uuid.New()

	// Setup: create a course owned by different teacher
	course := &courses.Course{
		ID:        courseID,
		TeacherID: teacherID,
		Title:     "Test Course",
		Status:    courses.CourseStatusDraft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	courseRepo.Create(context.Background(), course)

	svc := NewService(quizRepo, questionRepo, optionRepo, attemptRepo, newMockAssignmentRepo(), newMockSubmissionRepo(), newMockSubmissionFileRepo(), newMockGradeRepo(), courseRepo, &mockStorageClientProp{}, &mockNotificationQueueProp{}, "test-bucket")

	cmd := CreateQuizCommand{
		CourseID:  courseID,
		TeacherID: otherTeacherID, // Different teacher
		Title:     "Test Quiz",
	}

	_, err := svc.CreateQuiz(context.Background(), cmd)

	if err == nil {
		t.Fatal("Expected error for unauthorized teacher")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok || appErr.Code != "FORBIDDEN" {
		t.Errorf("Expected FORBIDDEN error, got %v", err)
	}
}

func TestGetTeacherQuizzes_Success(t *testing.T) {
	quizRepo := newMockQuizRepo()
	questionRepo := newMockQuestionRepo()
	optionRepo := newMockQuestionOptionRepo()
	attemptRepo := newMockQuizAttemptRepo()
	courseRepo := newMockCourseRepo()

	teacherID := uuid.New()
	courseID := uuid.New()

	// Setup: create a course owned by teacher
	course := &courses.Course{
		ID:        courseID,
		TeacherID: teacherID,
		Title:     "Test Course",
		Status:    courses.CourseStatusDraft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	courseRepo.Create(context.Background(), course)

	// Create a quiz with questions
	quizID := uuid.New()
	quiz := &assessments.Quiz{
		ID:                  quizID,
		CourseID:            courseID,
		Title:               "Test Quiz",
		TimeLimitSeconds:    1800,
		MaxAttempts:         3,
		PassingScorePercent: 60.0,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	quizRepo.Create(context.Background(), quiz)

	questionID := uuid.New()
	question := &assessments.Question{
		ID:          questionID,
		QuizID:      quizID,
		Body:        "What is 2+2?",
		Type:        assessments.QuestionTypeSingle,
		Position:    1,
		Explanation: "Basic arithmetic",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	questionRepo.Create(context.Background(), question)

	option1 := &assessments.QuestionOption{
		ID:         uuid.New(),
		QuestionID: questionID,
		Body:       "3",
		IsCorrect:  false,
		Position:   1,
	}
	option2 := &assessments.QuestionOption{
		ID:         uuid.New(),
		QuestionID: questionID,
		Body:       "4",
		IsCorrect:  true,
		Position:   2,
	}
	optionRepo.Create(context.Background(), option1)
	optionRepo.Create(context.Background(), option2)

	svc := NewService(quizRepo, questionRepo, optionRepo, attemptRepo, newMockAssignmentRepo(), newMockSubmissionRepo(), newMockSubmissionFileRepo(), newMockGradeRepo(), courseRepo, &mockStorageClientProp{}, &mockNotificationQueueProp{}, "test-bucket")

	quizzes, err := svc.GetTeacherQuizzes(context.Background(), courseID, teacherID)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(quizzes) != 1 {
		t.Fatalf("Expected 1 quiz, got %d", len(quizzes))
	}

	if quizzes[0].Title != "Test Quiz" {
		t.Errorf("Expected title 'Test Quiz', got %s", quizzes[0].Title)
	}

	if len(quizzes[0].Questions) != 1 {
		t.Fatalf("Expected 1 question, got %d", len(quizzes[0].Questions))
	}

	q := quizzes[0].Questions[0]
	if q.Body != "What is 2+2?" {
		t.Errorf("Expected question body 'What is 2+2?', got %s", q.Body)
	}

	if q.Explanation != "Basic arithmetic" {
		t.Errorf("Expected explanation 'Basic arithmetic', got %s", q.Explanation)
	}

	if len(q.Options) != 2 {
		t.Fatalf("Expected 2 options, got %d", len(q.Options))
	}

	if len(q.CorrectOptionIDs) != 1 {
		t.Fatalf("Expected 1 correct option, got %d", len(q.CorrectOptionIDs))
	}

	if q.CorrectOptionIDs[0] != option2.ID {
		t.Errorf("Expected correct option ID %s, got %s", option2.ID, q.CorrectOptionIDs[0])
	}
}

func TestGetTeacherQuizzes_NotAuthorized(t *testing.T) {
	quizRepo := newMockQuizRepo()
	questionRepo := newMockQuestionRepo()
	optionRepo := newMockQuestionOptionRepo()
	attemptRepo := newMockQuizAttemptRepo()
	courseRepo := newMockCourseRepo()

	teacherID := uuid.New()
	otherTeacherID := uuid.New()
	courseID := uuid.New()

	// Setup: create a course owned by different teacher
	course := &courses.Course{
		ID:        courseID,
		TeacherID: teacherID,
		Title:     "Test Course",
		Status:    courses.CourseStatusDraft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	courseRepo.Create(context.Background(), course)

	svc := NewService(quizRepo, questionRepo, optionRepo, attemptRepo, newMockAssignmentRepo(), newMockSubmissionRepo(), newMockSubmissionFileRepo(), newMockGradeRepo(), courseRepo, &mockStorageClientProp{}, &mockNotificationQueueProp{}, "test-bucket")

	_, err := svc.GetTeacherQuizzes(context.Background(), courseID, otherTeacherID)

	if err == nil {
		t.Fatal("Expected error for unauthorized teacher")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok || appErr.Code != "FORBIDDEN" {
		t.Errorf("Expected FORBIDDEN error, got %v", err)
	}
}

// Quiz Attempt Tests

func TestStartAttempt_Success(t *testing.T) {
	quizRepo := newMockQuizRepo()
	questionRepo := newMockQuestionRepo()
	optionRepo := newMockQuestionOptionRepo()
	attemptRepo := newMockQuizAttemptRepo()
	courseRepo := newMockCourseRepo()

	quizID := uuid.New()
	studentID := uuid.New()

	// Setup: create a quiz
	quiz := &assessments.Quiz{
		ID:                  quizID,
		CourseID:            uuid.New(),
		Title:               "Test Quiz",
		TimeLimitSeconds:    1800,
		MaxAttempts:         3,
		PassingScorePercent: 60.0,
		ShuffleQuestions:    false,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	quizRepo.Create(context.Background(), quiz)

	// Create questions
	questionID := uuid.New()
	question := &assessments.Question{
		ID:       questionID,
		QuizID:   quizID,
		Body:     "What is 2+2?",
		Type:     assessments.QuestionTypeSingle,
		Position: 1,
	}
	questionRepo.Create(context.Background(), question)

	// Create options
	option1 := &assessments.QuestionOption{
		ID:         uuid.New(),
		QuestionID: questionID,
		Body:       "4",
		IsCorrect:  true,
		Position:   1,
	}
	optionRepo.Create(context.Background(), option1)

	svc := NewService(quizRepo, questionRepo, optionRepo, attemptRepo, newMockAssignmentRepo(), newMockSubmissionRepo(), newMockSubmissionFileRepo(), newMockGradeRepo(), courseRepo, &mockStorageClientProp{}, &mockNotificationQueueProp{}, "test-bucket")

	cmd := StartAttemptCommand{
		QuizID:    quizID,
		StudentID: studentID,
	}

	resp, err := svc.StartAttempt(context.Background(), cmd)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.QuizID != quizID {
		t.Errorf("Expected quiz ID %s, got %s", quizID, resp.QuizID)
	}

	if len(resp.Questions) != 1 {
		t.Fatalf("Expected 1 question, got %d", len(resp.Questions))
	}

	// Verify no answer keys in student response
	q := resp.Questions[0]
	if q.Body != "What is 2+2?" {
		t.Errorf("Expected question body 'What is 2+2?', got %s", q.Body)
	}

	if len(q.Options) != 1 {
		t.Fatalf("Expected 1 option, got %d", len(q.Options))
	}

	// Verify expires_at is set
	if resp.ExpiresAt == nil {
		t.Error("Expected expires_at to be set")
	}
}

func TestStartAttempt_MaxAttemptsReached(t *testing.T) {
	quizRepo := newMockQuizRepo()
	questionRepo := newMockQuestionRepo()
	optionRepo := newMockQuestionOptionRepo()
	attemptRepo := newMockQuizAttemptRepo()
	courseRepo := newMockCourseRepo()

	quizID := uuid.New()
	studentID := uuid.New()

	// Setup: create a quiz with max 2 attempts
	quiz := &assessments.Quiz{
		ID:                  quizID,
		CourseID:            uuid.New(),
		Title:               "Test Quiz",
		MaxAttempts:         2,
		PassingScorePercent: 60.0,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	quizRepo.Create(context.Background(), quiz)

	// Create 2 existing attempts
	for i := 0; i < 2; i++ {
		attempt := &assessments.QuizAttempt{
			ID:        uuid.New(),
			QuizID:    quizID,
			StudentID: studentID,
			StartedAt: time.Now(),
			Status:    assessments.QuizAttemptStatusSubmitted,
		}
		attemptRepo.Create(context.Background(), attempt)
	}

	svc := NewService(quizRepo, questionRepo, optionRepo, attemptRepo, newMockAssignmentRepo(), newMockSubmissionRepo(), newMockSubmissionFileRepo(), newMockGradeRepo(), courseRepo, &mockStorageClientProp{}, &mockNotificationQueueProp{}, "test-bucket")

	cmd := StartAttemptCommand{
		QuizID:    quizID,
		StudentID: studentID,
	}

	_, err := svc.StartAttempt(context.Background(), cmd)

	if err == nil {
		t.Fatal("Expected error for max attempts reached")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok || appErr.Code != "MAX_ATTEMPTS_REACHED" {
		t.Errorf("Expected MAX_ATTEMPTS_REACHED error, got %v", err)
	}
}

func TestSubmitAttempt_Success(t *testing.T) {
	quizRepo := newMockQuizRepo()
	questionRepo := newMockQuestionRepo()
	optionRepo := newMockQuestionOptionRepo()
	attemptRepo := newMockQuizAttemptRepo()
	courseRepo := newMockCourseRepo()

	quizID := uuid.New()
	studentID := uuid.New()
	attemptID := uuid.New()

	// Setup: create a quiz
	quiz := &assessments.Quiz{
		ID:                         quizID,
		CourseID:                   uuid.New(),
		Title:                      "Test Quiz",
		PassingScorePercent:        60.0,
		ShowAnswersAfterSubmission: true,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}
	quizRepo.Create(context.Background(), quiz)

	// Create question
	questionID := uuid.New()
	question := &assessments.Question{
		ID:          questionID,
		QuizID:      quizID,
		Body:        "What is 2+2?",
		Type:        assessments.QuestionTypeSingle,
		Position:    1,
		Explanation: "Basic math",
	}
	questionRepo.Create(context.Background(), question)

	// Create options
	correctOptionID := uuid.New()
	option1 := &assessments.QuestionOption{
		ID:         correctOptionID,
		QuestionID: questionID,
		Body:       "4",
		IsCorrect:  true,
		Position:   1,
	}
	option2 := &assessments.QuestionOption{
		ID:         uuid.New(),
		QuestionID: questionID,
		Body:       "5",
		IsCorrect:  false,
		Position:   2,
	}
	optionRepo.Create(context.Background(), option1)
	optionRepo.Create(context.Background(), option2)

	// Create attempt
	attempt := &assessments.QuizAttempt{
		ID:        attemptID,
		QuizID:    quizID,
		StudentID: studentID,
		StartedAt: time.Now().Add(-5 * time.Minute),
		Status:    assessments.QuizAttemptStatusInProgress,
	}
	attemptRepo.Create(context.Background(), attempt)

	svc := NewService(quizRepo, questionRepo, optionRepo, attemptRepo, newMockAssignmentRepo(), newMockSubmissionRepo(), newMockSubmissionFileRepo(), newMockGradeRepo(), courseRepo, &mockStorageClientProp{}, &mockNotificationQueueProp{}, "test-bucket")

	cmd := SubmitAttemptCommand{
		AttemptID: attemptID,
		StudentID: studentID,
		Answers: []QuizAnswerCommand{
			{
				QuestionID:      questionID,
				SelectedOptions: []uuid.UUID{correctOptionID},
			},
		},
	}

	resp, err := svc.SubmitAttempt(context.Background(), cmd)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.ScorePercent != 100.0 {
		t.Errorf("Expected score 100.0, got %f", resp.ScorePercent)
	}

	if !resp.Passed {
		t.Error("Expected passed to be true")
	}

	if resp.PointsAwarded != 30 {
		t.Errorf("Expected 30 points (20 + 10 bonus), got %d", resp.PointsAwarded)
	}

	if len(resp.QuestionResults) != 1 {
		t.Fatalf("Expected 1 question result, got %d", len(resp.QuestionResults))
	}

	if !resp.QuestionResults[0].IsCorrect {
		t.Error("Expected question to be marked correct")
	}
}

func TestSubmitAttempt_PartialScore(t *testing.T) {
	quizRepo := newMockQuizRepo()
	questionRepo := newMockQuestionRepo()
	optionRepo := newMockQuestionOptionRepo()
	attemptRepo := newMockQuizAttemptRepo()
	courseRepo := newMockCourseRepo()

	quizID := uuid.New()
	studentID := uuid.New()
	attemptID := uuid.New()

	// Setup: create a quiz
	quiz := &assessments.Quiz{
		ID:                         quizID,
		CourseID:                   uuid.New(),
		Title:                      "Test Quiz",
		PassingScorePercent:        60.0,
		ShowAnswersAfterSubmission: false,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}
	quizRepo.Create(context.Background(), quiz)

	// Create 2 questions
	q1ID := uuid.New()
	q2ID := uuid.New()
	questionRepo.Create(context.Background(), &assessments.Question{
		ID:       q1ID,
		QuizID:   quizID,
		Body:     "Q1",
		Type:     assessments.QuestionTypeSingle,
		Position: 1,
	})
	questionRepo.Create(context.Background(), &assessments.Question{
		ID:       q2ID,
		QuizID:   quizID,
		Body:     "Q2",
		Type:     assessments.QuestionTypeSingle,
		Position: 2,
	})

	// Create options
	correctOpt1 := uuid.New()
	correctOpt2 := uuid.New()
	wrongOpt := uuid.New()

	optionRepo.Create(context.Background(), &assessments.QuestionOption{
		ID:         correctOpt1,
		QuestionID: q1ID,
		Body:       "Correct",
		IsCorrect:  true,
		Position:   1,
	})
	optionRepo.Create(context.Background(), &assessments.QuestionOption{
		ID:         correctOpt2,
		QuestionID: q2ID,
		Body:       "Correct",
		IsCorrect:  true,
		Position:   1,
	})
	optionRepo.Create(context.Background(), &assessments.QuestionOption{
		ID:         wrongOpt,
		QuestionID: q2ID,
		Body:       "Wrong",
		IsCorrect:  false,
		Position:   2,
	})

	// Create attempt
	attempt := &assessments.QuizAttempt{
		ID:        attemptID,
		QuizID:    quizID,
		StudentID: studentID,
		StartedAt: time.Now().Add(-5 * time.Minute),
		Status:    assessments.QuizAttemptStatusInProgress,
	}
	attemptRepo.Create(context.Background(), attempt)

	svc := NewService(quizRepo, questionRepo, optionRepo, attemptRepo, newMockAssignmentRepo(), newMockSubmissionRepo(), newMockSubmissionFileRepo(), newMockGradeRepo(), courseRepo, &mockStorageClientProp{}, &mockNotificationQueueProp{}, "test-bucket")

	// Answer only 1 out of 2 correctly
	cmd := SubmitAttemptCommand{
		AttemptID: attemptID,
		StudentID: studentID,
		Answers: []QuizAnswerCommand{
			{QuestionID: q1ID, SelectedOptions: []uuid.UUID{correctOpt1}},
			{QuestionID: q2ID, SelectedOptions: []uuid.UUID{wrongOpt}},
		},
	}

	resp, err := svc.SubmitAttempt(context.Background(), cmd)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.ScorePercent != 50.0 {
		t.Errorf("Expected score 50.0, got %f", resp.ScorePercent)
	}

	if resp.Passed {
		t.Error("Expected passed to be false (50% < 60%)")
	}

	if resp.PointsAwarded != 0 {
		t.Errorf("Expected 0 points (not passing), got %d", resp.PointsAwarded)
	}

	// Verify no question results when show_answers_after_submission is false
	if len(resp.QuestionResults) != 0 {
		t.Errorf("Expected 0 question results, got %d", len(resp.QuestionResults))
	}
}

func TestSubmitAttempt_HighestScoreOnly(t *testing.T) {
	quizRepo := newMockQuizRepo()
	questionRepo := newMockQuestionRepo()
	optionRepo := newMockQuestionOptionRepo()
	attemptRepo := newMockQuizAttemptRepo()
	courseRepo := newMockCourseRepo()

	quizID := uuid.New()
	studentID := uuid.New()

	// Setup: create a quiz
	quiz := &assessments.Quiz{
		ID:                  quizID,
		CourseID:            uuid.New(),
		Title:               "Test Quiz",
		PassingScorePercent: 60.0,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	quizRepo.Create(context.Background(), quiz)

	// Create question
	questionID := uuid.New()
	questionRepo.Create(context.Background(), &assessments.Question{
		ID:       questionID,
		QuizID:   quizID,
		Body:     "Q1",
		Type:     assessments.QuestionTypeSingle,
		Position: 1,
	})

	correctOpt := uuid.New()
	optionRepo.Create(context.Background(), &assessments.QuestionOption{
		ID:         correctOpt,
		QuestionID: questionID,
		Body:       "Correct",
		IsCorrect:  true,
		Position:   1,
	})

	// Create first attempt with 80% score (passing)
	score80 := 80.0
	passed := true
	attempt1 := &assessments.QuizAttempt{
		ID:           uuid.New(),
		QuizID:       quizID,
		StudentID:    studentID,
		StartedAt:    time.Now().Add(-1 * time.Hour),
		Status:       assessments.QuizAttemptStatusSubmitted,
		ScorePercent: &score80,
		Passed:       &passed,
	}
	attemptRepo.Create(context.Background(), attempt1)

	// Create second attempt (in progress)
	attempt2ID := uuid.New()
	attempt2 := &assessments.QuizAttempt{
		ID:        attempt2ID,
		QuizID:    quizID,
		StudentID: studentID,
		StartedAt: time.Now().Add(-5 * time.Minute),
		Status:    assessments.QuizAttemptStatusInProgress,
	}
	attemptRepo.Create(context.Background(), attempt2)

	svc := NewService(quizRepo, questionRepo, optionRepo, attemptRepo, newMockAssignmentRepo(), newMockSubmissionRepo(), newMockSubmissionFileRepo(), newMockGradeRepo(), courseRepo, &mockStorageClientProp{}, &mockNotificationQueueProp{}, "test-bucket")

	// Submit second attempt with 100% score
	cmd := SubmitAttemptCommand{
		AttemptID: attempt2ID,
		StudentID: studentID,
		Answers: []QuizAnswerCommand{
			{QuestionID: questionID, SelectedOptions: []uuid.UUID{correctOpt}},
		},
	}

	resp, err := svc.SubmitAttempt(context.Background(), cmd)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.ScorePercent != 100.0 {
		t.Errorf("Expected score 100.0, got %f", resp.ScorePercent)
	}

	// Should NOT award points because already passed before
	if resp.PointsAwarded != 0 {
		t.Errorf("Expected 0 points (already passed), got %d", resp.PointsAwarded)
	}
}

func TestSubmitAttempt_NotAuthorized(t *testing.T) {
	quizRepo := newMockQuizRepo()
	questionRepo := newMockQuestionRepo()
	optionRepo := newMockQuestionOptionRepo()
	attemptRepo := newMockQuizAttemptRepo()
	courseRepo := newMockCourseRepo()

	quizID := uuid.New()
	studentID := uuid.New()
	otherStudentID := uuid.New()
	attemptID := uuid.New()

	// Create attempt owned by different student
	attempt := &assessments.QuizAttempt{
		ID:        attemptID,
		QuizID:    quizID,
		StudentID: studentID,
		StartedAt: time.Now(),
		Status:    assessments.QuizAttemptStatusInProgress,
	}
	attemptRepo.Create(context.Background(), attempt)

	svc := NewService(quizRepo, questionRepo, optionRepo, attemptRepo, newMockAssignmentRepo(), newMockSubmissionRepo(), newMockSubmissionFileRepo(), newMockGradeRepo(), courseRepo, &mockStorageClientProp{}, &mockNotificationQueueProp{}, "test-bucket")

	cmd := SubmitAttemptCommand{
		AttemptID: attemptID,
		StudentID: otherStudentID, // Different student
		Answers:   []QuizAnswerCommand{},
	}

	_, err := svc.SubmitAttempt(context.Background(), cmd)

	if err == nil {
		t.Fatal("Expected error for unauthorized student")
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok || appErr.Code != "FORBIDDEN" {
		t.Errorf("Expected FORBIDDEN error, got %v", err)
	}
}

func TestAutoSubmitAttempt_Success(t *testing.T) {
	quizRepo := newMockQuizRepo()
	questionRepo := newMockQuestionRepo()
	optionRepo := newMockQuestionOptionRepo()
	attemptRepo := newMockQuizAttemptRepo()
	courseRepo := newMockCourseRepo()

	quizID := uuid.New()
	studentID := uuid.New()
	attemptID := uuid.New()

	// Setup: create a quiz with time limit
	quiz := &assessments.Quiz{
		ID:                  quizID,
		CourseID:            uuid.New(),
		Title:               "Test Quiz",
		TimeLimitSeconds:    300, // 5 minutes
		PassingScorePercent: 60.0,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	quizRepo.Create(context.Background(), quiz)

	// Create question
	questionID := uuid.New()
	questionRepo.Create(context.Background(), &assessments.Question{
		ID:       questionID,
		QuizID:   quizID,
		Body:     "Q1",
		Type:     assessments.QuestionTypeSingle,
		Position: 1,
	})

	optionRepo.Create(context.Background(), &assessments.QuestionOption{
		ID:         uuid.New(),
		QuestionID: questionID,
		Body:       "Option",
		IsCorrect:  true,
		Position:   1,
	})

	// Create expired attempt (started 10 minutes ago)
	attempt := &assessments.QuizAttempt{
		ID:        attemptID,
		QuizID:    quizID,
		StudentID: studentID,
		StartedAt: time.Now().Add(-10 * time.Minute),
		Status:    assessments.QuizAttemptStatusInProgress,
	}
	attemptRepo.Create(context.Background(), attempt)

	svc := NewService(quizRepo, questionRepo, optionRepo, attemptRepo, newMockAssignmentRepo(), newMockSubmissionRepo(), newMockSubmissionFileRepo(), newMockGradeRepo(), courseRepo, &mockStorageClientProp{}, &mockNotificationQueueProp{}, "test-bucket")

	resp, err := svc.AutoSubmitAttempt(context.Background(), attemptID)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.ScorePercent != 0.0 {
		t.Errorf("Expected score 0.0 (no answers), got %f", resp.ScorePercent)
	}

	if resp.Passed {
		t.Error("Expected passed to be false")
	}

	if resp.PointsAwarded != 0 {
		t.Errorf("Expected 0 points for auto-submit, got %d", resp.PointsAwarded)
	}

	// Verify attempt was updated
	updatedAttempt, _ := attemptRepo.FindByID(context.Background(), attemptID)
	if updatedAttempt.Status != assessments.QuizAttemptStatusAutoSubmitted {
		t.Errorf("Expected status auto_submitted, got %s", updatedAttempt.Status)
	}
}
