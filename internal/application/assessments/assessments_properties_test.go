package assessments

import (
	"context"
	"io"
	"testing"
	"time"

	"lms-backend/internal/domain/assessments"
	"lms-backend/internal/domain/courses"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// ─── Mock implementations ────────────────────────────────────────────────────

// mockAssignmentRepo implements assessments.AssignmentRepository
type mockAssignmentRepo struct {
	assignments map[uuid.UUID]*assessments.Assignment
}

func newMockAssignmentRepo() *mockAssignmentRepo {
	return &mockAssignmentRepo{assignments: make(map[uuid.UUID]*assessments.Assignment)}
}

func (m *mockAssignmentRepo) Create(ctx context.Context, a *assessments.Assignment) error {
	m.assignments[a.ID] = a
	return nil
}

func (m *mockAssignmentRepo) FindByID(ctx context.Context, id uuid.UUID) (*assessments.Assignment, error) {
	if a, ok := m.assignments[id]; ok {
		return a, nil
	}
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockAssignmentRepo) FindByCourseID(ctx context.Context, courseID uuid.UUID) ([]*assessments.Assignment, error) {
	var result []*assessments.Assignment
	for _, a := range m.assignments {
		if a.CourseID == courseID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockAssignmentRepo) Update(ctx context.Context, a *assessments.Assignment) error {
	m.assignments[a.ID] = a
	return nil
}

func (m *mockAssignmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.assignments, id)
	return nil
}

// mockSubmissionRepo implements assessments.AssignmentSubmissionRepository
type mockSubmissionRepo struct {
	submissions map[uuid.UUID]*assessments.AssignmentSubmission
}

func newMockSubmissionRepo() *mockSubmissionRepo {
	return &mockSubmissionRepo{submissions: make(map[uuid.UUID]*assessments.AssignmentSubmission)}
}

func (m *mockSubmissionRepo) Create(ctx context.Context, s *assessments.AssignmentSubmission) error {
	m.submissions[s.ID] = s
	return nil
}

func (m *mockSubmissionRepo) FindByID(ctx context.Context, id uuid.UUID) (*assessments.AssignmentSubmission, error) {
	if s, ok := m.submissions[id]; ok {
		return s, nil
	}
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockSubmissionRepo) FindByAssignmentAndStudent(ctx context.Context, assignmentID, studentID uuid.UUID) (*assessments.AssignmentSubmission, error) {
	for _, s := range m.submissions {
		if s.AssignmentID == assignmentID && s.StudentID == studentID && s.Status != assessments.AssignmentSubmissionStatusDraft {
			return s, nil
		}
	}
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockSubmissionRepo) FindByAssignmentID(ctx context.Context, assignmentID uuid.UUID, page, limit int) ([]*assessments.AssignmentSubmission, int, error) {
	var result []*assessments.AssignmentSubmission
	for _, s := range m.submissions {
		if s.AssignmentID == assignmentID {
			result = append(result, s)
		}
	}
	return result, len(result), nil
}

func (m *mockSubmissionRepo) FindDraftByAssignmentAndStudent(ctx context.Context, assignmentID, studentID uuid.UUID) (*assessments.AssignmentSubmission, error) {
	for _, s := range m.submissions {
		if s.AssignmentID == assignmentID && s.StudentID == studentID && s.Status == assessments.AssignmentSubmissionStatusDraft {
			return s, nil
		}
	}
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockSubmissionRepo) Update(ctx context.Context, s *assessments.AssignmentSubmission) error {
	m.submissions[s.ID] = s
	return nil
}

func (m *mockSubmissionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.submissions, id)
	return nil
}

// mockSubmissionFileRepo implements assessments.SubmissionFileRepository
type mockSubmissionFileRepo struct {
	files map[uuid.UUID]*assessments.SubmissionFile
}

func newMockSubmissionFileRepo() *mockSubmissionFileRepo {
	return &mockSubmissionFileRepo{files: make(map[uuid.UUID]*assessments.SubmissionFile)}
}

func (m *mockSubmissionFileRepo) Create(ctx context.Context, f *assessments.SubmissionFile) error {
	m.files[f.ID] = f
	return nil
}

func (m *mockSubmissionFileRepo) CreateBatch(ctx context.Context, files []*assessments.SubmissionFile) error {
	for _, f := range files {
		m.files[f.ID] = f
	}
	return nil
}

func (m *mockSubmissionFileRepo) FindBySubmissionID(ctx context.Context, submissionID uuid.UUID) ([]*assessments.SubmissionFile, error) {
	var result []*assessments.SubmissionFile
	for _, f := range m.files {
		if f.SubmissionID == submissionID {
			result = append(result, f)
		}
	}
	return result, nil
}

func (m *mockSubmissionFileRepo) CountBySubmissionID(ctx context.Context, submissionID uuid.UUID) (int, error) {
	count := 0
	for _, f := range m.files {
		if f.SubmissionID == submissionID {
			count++
		}
	}
	return count, nil
}

func (m *mockSubmissionFileRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.files, id)
	return nil
}

// mockGradeRepo implements assessments.SubmissionGradeRepository (append-only)
type mockGradeRepo struct {
	grades []*assessments.SubmissionGrade
}

func newMockGradeRepo() *mockGradeRepo {
	return &mockGradeRepo{grades: make([]*assessments.SubmissionGrade, 0)}
}

func (m *mockGradeRepo) Create(ctx context.Context, g *assessments.SubmissionGrade) error {
	// Append-only: always add a new record, never overwrite
	m.grades = append(m.grades, g)
	return nil
}

func (m *mockGradeRepo) FindByID(ctx context.Context, id uuid.UUID) (*assessments.SubmissionGrade, error) {
	for _, g := range m.grades {
		if g.ID == id {
			return g, nil
		}
	}
	return nil, &apperrors.AppError{Code: "NOT_FOUND"}
}

func (m *mockGradeRepo) FindBySubmissionID(ctx context.Context, submissionID uuid.UUID) ([]*assessments.SubmissionGrade, error) {
	var result []*assessments.SubmissionGrade
	for _, g := range m.grades {
		if g.SubmissionID == submissionID {
			result = append(result, g)
		}
	}
	return result, nil
}

func (m *mockGradeRepo) GetLatestGrade(ctx context.Context, submissionID uuid.UUID) (*assessments.SubmissionGrade, error) {
	var latest *assessments.SubmissionGrade
	for _, g := range m.grades {
		if g.SubmissionID == submissionID {
			if latest == nil || g.GradedAt.After(latest.GradedAt) {
				latest = g
			}
		}
	}
	if latest == nil {
		return nil, &apperrors.AppError{Code: "NOT_FOUND"}
	}
	return latest, nil
}

// mockStorageClient implements StorageClient
type mockStorageClientProp struct{}

func (m *mockStorageClientProp) PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) error {
	return nil
}

func (m *mockStorageClientProp) PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	return "https://storage.example.com/presigned/" + key, nil
}

func (m *mockStorageClientProp) DeleteObject(ctx context.Context, bucket, key string) error {
	return nil
}

// mockNotificationQueueProp implements NotificationQueue
type mockNotificationQueueProp struct{}

func (m *mockNotificationQueueProp) EnqueueNotification(ctx context.Context, userID uuid.UUID, notificationType, title, body string) error {
	return nil
}

// ─── Helper: build a full service with all mocks ─────────────────────────────

type propTestDeps struct {
	quizRepo       *mockQuizRepo
	questionRepo   *mockQuestionRepo
	optionRepo     *mockQuestionOptionRepo
	attemptRepo    *mockQuizAttemptRepo
	assignmentRepo *mockAssignmentRepo
	submissionRepo *mockSubmissionRepo
	fileRepo       *mockSubmissionFileRepo
	gradeRepo      *mockGradeRepo
	courseRepo     *mockCourseRepo
}

func newPropTestDeps() *propTestDeps {
	return &propTestDeps{
		quizRepo:       newMockQuizRepo(),
		questionRepo:   newMockQuestionRepo(),
		optionRepo:     newMockQuestionOptionRepo(),
		attemptRepo:    newMockQuizAttemptRepo(),
		assignmentRepo: newMockAssignmentRepo(),
		submissionRepo: newMockSubmissionRepo(),
		fileRepo:       newMockSubmissionFileRepo(),
		gradeRepo:      newMockGradeRepo(),
		courseRepo:     newMockCourseRepo(),
	}
}

func (d *propTestDeps) service() Service {
	return NewService(
		d.quizRepo,
		d.questionRepo,
		d.optionRepo,
		d.attemptRepo,
		d.assignmentRepo,
		d.submissionRepo,
		d.fileRepo,
		d.gradeRepo,
		d.courseRepo,
		&mockStorageClientProp{},
		&mockNotificationQueueProp{},
		"test-bucket",
	)
}

// seedQuizWithQuestion creates a quiz, one question, and one correct + one wrong option.
// Returns (quizID, questionID, correctOptionID, wrongOptionID).
func seedQuizWithQuestion(deps *propTestDeps, teacherID uuid.UUID, passingScore float64, showAnswers bool) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	courseID := uuid.New()
	deps.courseRepo.courses[courseID] = &courses.Course{
		ID:        courseID,
		TeacherID: teacherID,
		Title:     "Test Course",
		Status:    courses.CourseStatusPublished,
	}

	quizID := uuid.New()
	deps.quizRepo.quizzes[quizID] = &assessments.Quiz{
		ID:                         quizID,
		CourseID:                   courseID,
		Title:                      "Test Quiz",
		PassingScorePercent:        passingScore,
		MaxAttempts:                0, // unlimited
		ShowAnswersAfterSubmission: showAnswers,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}

	questionID := uuid.New()
	deps.questionRepo.questions[questionID] = &assessments.Question{
		ID:          questionID,
		QuizID:      quizID,
		Body:        "What is 2+2?",
		Type:        assessments.QuestionTypeSingle,
		Position:    1,
		Explanation: "Basic arithmetic — teacher only",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	correctOptID := uuid.New()
	wrongOptID := uuid.New()
	deps.optionRepo.options[correctOptID] = &assessments.QuestionOption{
		ID:         correctOptID,
		QuestionID: questionID,
		Body:       "4",
		IsCorrect:  true,
		Position:   1,
	}
	deps.optionRepo.options[wrongOptID] = &assessments.QuestionOption{
		ID:         wrongOptID,
		QuestionID: questionID,
		Body:       "5",
		IsCorrect:  false,
		Position:   2,
	}

	return quizID, questionID, correctOptID, wrongOptID
}

// ─── Property 44 ─────────────────────────────────────────────────────────────

// TestProperty44_AnswerKeyFieldsNeverInStudentResponse verifies that
// is_correct, correct_option_ids, and explanation are never present in
// student-facing quiz responses (StartAttempt).
//
// Validates: Requirements 14.2
func TestProperty44_AnswerKeyFieldsNeverInStudentResponse(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		deps := newPropTestDeps()
		svc := deps.service()

		studentID := uuid.New()
		teacherID := uuid.New()

		// Generate random quiz settings
		passingScore := rapid.Float64Range(1.0, 100.0).Draw(t, "passing_score")
		showAnswers := rapid.Bool().Draw(t, "show_answers")

		quizID, _, _, _ := seedQuizWithQuestion(deps, teacherID, passingScore, showAnswers)

		// Start an attempt as a student
		resp, err := svc.StartAttempt(context.Background(), StartAttemptCommand{
			QuizID:    quizID,
			StudentID: studentID,
		})
		if err != nil {
			t.Fatalf("StartAttempt failed: %v", err)
		}

		// Property: QuestionStudentResponse must NOT contain answer key fields.
		// The struct type itself enforces this at compile time — QuestionStudentResponse
		// has no Explanation, IsCorrect, or CorrectOptionIDs fields.
		// We additionally verify the options have no IsCorrect field.
		for _, q := range resp.Questions {
			// QuestionStudentResponse has no Explanation field — verified by type
			// QuestionStudentResponse has no CorrectOptionIDs field — verified by type

			for _, opt := range q.Options {
				// QuestionOptionStudentResponse has no IsCorrect field — verified by type
				// Verify the option body is non-empty (it's a real option, not a stub)
				if opt.Body == "" {
					t.Fatal("option body should not be empty in student response")
				}
				// Verify option has an ID
				if opt.ID == uuid.Nil {
					t.Fatal("option ID should not be nil in student response")
				}
			}
		}

		// Verify the response type is QuizAttemptResponse (student type), not QuizTeacherResponse
		// This is enforced at compile time by the return type of StartAttempt.
		// The QuizAttemptResponse.Questions field is []QuestionStudentResponse,
		// which structurally cannot contain IsCorrect, Explanation, or CorrectOptionIDs.
		if resp.QuizID != quizID {
			t.Fatalf("expected quiz ID %v, got %v", quizID, resp.QuizID)
		}

		// Verify questions are present
		if len(resp.Questions) == 0 {
			t.Fatal("expected at least one question in student response")
		}
	})
}

// ─── Property 45 ─────────────────────────────────────────────────────────────

// TestProperty45_GradebookRecordsOnlyHighestScore verifies that when a student
// submits multiple attempts, the gradebook (GetHighestScore) always reflects
// the highest score achieved, never a lower subsequent score.
//
// Validates: Requirements 14.7
func TestProperty45_GradebookRecordsOnlyHighestScore(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		deps := newPropTestDeps()
		svc := deps.service()

		studentID := uuid.New()
		teacherID := uuid.New()

		// Use a passing score of 50% so we can control pass/fail
		quizID, questionID, correctOptID, wrongOptID := seedQuizWithQuestion(deps, teacherID, 50.0, false)

		// Generate a sequence of 2–5 attempts with random correct/wrong answers
		numAttempts := rapid.IntRange(2, 5).Draw(t, "num_attempts")
		answers := make([]bool, numAttempts) // true = correct answer
		for i := range answers {
			answers[i] = rapid.Bool().Draw(t, "correct")
		}

		var highestScore float64 = -1.0

		for i, correct := range answers {
			// Start attempt
			startResp, err := svc.StartAttempt(context.Background(), StartAttemptCommand{
				QuizID:    quizID,
				StudentID: studentID,
			})
			if err != nil {
				t.Fatalf("StartAttempt %d failed: %v", i, err)
			}

			// Choose answer
			selectedOpt := wrongOptID
			expectedScore := 0.0
			if correct {
				selectedOpt = correctOptID
				expectedScore = 100.0
			}

			// Submit attempt
			submitResp, err := svc.SubmitAttempt(context.Background(), SubmitAttemptCommand{
				AttemptID: startResp.ID,
				StudentID: studentID,
				Answers: []QuizAnswerCommand{
					{QuestionID: questionID, SelectedOptions: []uuid.UUID{selectedOpt}},
				},
			})
			if err != nil {
				t.Fatalf("SubmitAttempt %d failed: %v", i, err)
			}

			if submitResp.ScorePercent != expectedScore {
				t.Fatalf("attempt %d: expected score %.1f, got %.1f", i, expectedScore, submitResp.ScorePercent)
			}

			if expectedScore > highestScore {
				highestScore = expectedScore
			}
		}

		// Property: GetHighestScore must equal the highest score across all attempts
		storedHighest, err := deps.attemptRepo.GetHighestScore(context.Background(), quizID, studentID)
		if err != nil {
			t.Fatalf("GetHighestScore failed: %v", err)
		}

		if storedHighest == nil {
			t.Fatal("expected a highest score to be recorded")
		}

		if *storedHighest != highestScore {
			t.Fatalf("gradebook highest score: expected %.1f, got %.1f", highestScore, *storedHighest)
		}

		// Property: No individual attempt's score should exceed the recorded highest
		for _, attempt := range deps.attemptRepo.attempts {
			if attempt.QuizID == quizID && attempt.StudentID == studentID && attempt.ScorePercent != nil {
				if *attempt.ScorePercent > *storedHighest {
					t.Fatalf("attempt score %.1f exceeds recorded highest %.1f", *attempt.ScorePercent, *storedHighest)
				}
			}
		}
	})
}

// ─── Property 46 ─────────────────────────────────────────────────────────────

// TestProperty46_PointsAwardedOnlyOnFirstPassingAttempt verifies that quiz
// points are awarded exactly once — on the first passing attempt — and never
// re-awarded on subsequent passing attempts.
//
// Validates: Requirements 14.8, 17.6
func TestProperty46_PointsAwardedOnlyOnFirstPassingAttempt(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		deps := newPropTestDeps()
		svc := deps.service()

		studentID := uuid.New()
		teacherID := uuid.New()

		// Use a passing score of 50% so a correct answer (100%) always passes
		quizID, questionID, correctOptID, _ := seedQuizWithQuestion(deps, teacherID, 50.0, false)

		// Generate 2–4 passing attempts
		numPassingAttempts := rapid.IntRange(2, 4).Draw(t, "num_passing_attempts")

		totalPointsAwarded := 0
		firstPassPoints := 0

		for i := 0; i < numPassingAttempts; i++ {
			startResp, err := svc.StartAttempt(context.Background(), StartAttemptCommand{
				QuizID:    quizID,
				StudentID: studentID,
			})
			if err != nil {
				t.Fatalf("StartAttempt %d failed: %v", i, err)
			}

			submitResp, err := svc.SubmitAttempt(context.Background(), SubmitAttemptCommand{
				AttemptID: startResp.ID,
				StudentID: studentID,
				Answers: []QuizAnswerCommand{
					{QuestionID: questionID, SelectedOptions: []uuid.UUID{correctOptID}},
				},
			})
			if err != nil {
				t.Fatalf("SubmitAttempt %d failed: %v", i, err)
			}

			if !submitResp.Passed {
				t.Fatalf("attempt %d should have passed (100%% score)", i)
			}

			totalPointsAwarded += submitResp.PointsAwarded

			if i == 0 {
				// First passing attempt must award points
				if submitResp.PointsAwarded == 0 {
					t.Fatal("first passing attempt must award points")
				}
				firstPassPoints = submitResp.PointsAwarded
			} else {
				// Subsequent passing attempts must NOT award points
				if submitResp.PointsAwarded != 0 {
					t.Fatalf("attempt %d (re-pass) should award 0 points, got %d", i, submitResp.PointsAwarded)
				}
			}
		}

		// Property: total points awarded across all attempts equals points from first pass only
		if totalPointsAwarded != firstPassPoints {
			t.Fatalf("total points awarded %d != first pass points %d", totalPointsAwarded, firstPassPoints)
		}

		// Property: verify in the attempt records that only the first submitted attempt has points
		pointsInRepo := 0
		for _, attempt := range deps.attemptRepo.attempts {
			if attempt.QuizID == quizID && attempt.StudentID == studentID {
				pointsInRepo += attempt.PointsAwarded
			}
		}

		if pointsInRepo != firstPassPoints {
			t.Fatalf("points in repo %d != first pass points %d", pointsInRepo, firstPassPoints)
		}
	})
}

// ─── Property 47 ─────────────────────────────────────────────────────────────

// TestProperty47_GradeRevisionHistoryIsAppendOnly verifies that each call to
// GradeSubmission appends a new SubmissionGrade record rather than overwriting
// the previous one, and that the history is monotonically growing.
//
// Validates: Requirements 15.8
func TestProperty47_GradeRevisionHistoryIsAppendOnly(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		deps := newPropTestDeps()
		svc := deps.service()

		teacherID := uuid.New()
		studentID := uuid.New()

		// Create a course owned by the teacher
		courseID := uuid.New()
		deps.courseRepo.courses[courseID] = &courses.Course{
			ID:        courseID,
			TeacherID: teacherID,
			Title:     "Test Course",
			Status:    courses.CourseStatusPublished,
		}

		// Create an assignment
		assignmentID := uuid.New()
		totalMarks := rapid.Float64Range(10.0, 100.0).Draw(t, "total_marks")
		deps.assignmentRepo.assignments[assignmentID] = &assessments.Assignment{
			ID:             assignmentID,
			CourseID:       courseID,
			Title:          "Test Assignment",
			DueAt:          time.Now().Add(24 * time.Hour), // not past deadline
			SubmissionType: assessments.SubmissionTypeBoth,
			MaxFileSizeMB:  10,
			TotalMarks:     totalMarks,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// Create a submitted submission
		submissionID := uuid.New()
		now := time.Now()
		deps.submissionRepo.submissions[submissionID] = &assessments.AssignmentSubmission{
			ID:           submissionID,
			AssignmentID: assignmentID,
			StudentID:    studentID,
			Status:       assessments.AssignmentSubmissionStatusSubmitted,
			SubmittedAt:  &now,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		// Generate 2–5 grading actions
		numGrades := rapid.IntRange(2, 5).Draw(t, "num_grades")
		scores := make([]float64, numGrades)
		for i := range scores {
			scores[i] = rapid.Float64Range(0.0, totalMarks).Draw(t, "score")
		}

		gradeIDs := make([]uuid.UUID, 0, numGrades)

		for i, score := range scores {
			revisionRequested := i < numGrades-1 // request revision on all but last

			resp, err := svc.GradeSubmission(context.Background(), GradeSubmissionCommand{
				SubmissionID:      submissionID,
				GradedBy:          teacherID,
				Score:             score,
				Feedback:          "Feedback round " + string(rune('A'+i)),
				RevisionRequested: revisionRequested,
			})
			if err != nil {
				t.Fatalf("GradeSubmission %d failed: %v", i, err)
			}

			gradeIDs = append(gradeIDs, resp.ID)

			// After each grade, verify the total count in the repo equals i+1
			allGrades, err := deps.gradeRepo.FindBySubmissionID(context.Background(), submissionID)
			if err != nil {
				t.Fatalf("FindBySubmissionID failed after grade %d: %v", i, err)
			}

			// Property: grade count must grow by exactly 1 with each grading action
			if len(allGrades) != i+1 {
				t.Fatalf("after %d grading actions, expected %d grade records, got %d", i+1, i+1, len(allGrades))
			}

			// Property: all previously created grade IDs must still exist (no overwrites)
			existingIDs := make(map[uuid.UUID]bool)
			for _, g := range allGrades {
				existingIDs[g.ID] = true
			}
			for _, prevID := range gradeIDs {
				if !existingIDs[prevID] {
					t.Fatalf("grade record %v was overwritten or deleted (append-only violation)", prevID)
				}
			}
		}

		// Final property: total grade records == numGrades
		finalGrades, err := deps.gradeRepo.FindBySubmissionID(context.Background(), submissionID)
		if err != nil {
			t.Fatalf("final FindBySubmissionID failed: %v", err)
		}

		if len(finalGrades) != numGrades {
			t.Fatalf("expected %d total grade records, got %d", numGrades, len(finalGrades))
		}

		// Property: all grade IDs are unique (no duplicates)
		seen := make(map[uuid.UUID]bool)
		for _, g := range finalGrades {
			if seen[g.ID] {
				t.Fatalf("duplicate grade ID %v found — append-only violated", g.ID)
			}
			seen[g.ID] = true
		}
	})
}
