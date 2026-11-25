package assessments

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"lms-backend/internal/domain/assessments"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

type quizFixture struct {
	svc        Service
	attempts   *mockQuizAttemptRepo
	quizID     uuid.UUID
	questionID uuid.UUID
	correctID  uuid.UUID
	wrongID    uuid.UUID
	studentID  uuid.UUID
}

func newQuizFixture(t *testing.T, published bool, timeLimit int) *quizFixture {
	t.Helper()
	quizRepo := newMockQuizRepo()
	questionRepo := newMockQuestionRepo()
	optionRepo := newMockQuestionOptionRepo()
	attemptRepo := newMockQuizAttemptRepo()
	f := &quizFixture{
		attempts: attemptRepo, quizID: uuid.New(), questionID: uuid.New(),
		correctID: uuid.New(), wrongID: uuid.New(), studentID: uuid.New(),
	}
	quizRepo.Create(context.Background(), &assessments.Quiz{
		ID: f.quizID, CourseID: uuid.New(), Title: "Q", IsPublished: published,
		PassingScorePercent: 50, TimeLimitSeconds: timeLimit,
	})
	questionRepo.Create(context.Background(), &assessments.Question{ID: f.questionID, QuizID: f.quizID, Type: assessments.QuestionTypeSingle, Body: "2+2"})
	optionRepo.Create(context.Background(), &assessments.QuestionOption{ID: f.correctID, QuestionID: f.questionID, Body: "4", IsCorrect: true})
	optionRepo.Create(context.Background(), &assessments.QuestionOption{ID: f.wrongID, QuestionID: f.questionID, Body: "5"})
	f.svc = NewService(quizRepo, questionRepo, optionRepo, attemptRepo, newMockAssignmentRepo(), newMockSubmissionRepo(), newMockSubmissionFileRepo(), newMockGradeRepo(), newMockCourseRepo(), &mockStorageClientProp{}, &mockNotificationQueueProp{}, "bucket")
	return f
}

func (f *quizFixture) addAttempt(startedAgo time.Duration, draft map[string]interface{}) uuid.UUID {
	id := uuid.New()
	raw, _ := json.Marshal(draft)
	f.attempts.Create(context.Background(), &assessments.QuizAttempt{
		ID: id, QuizID: f.quizID, StudentID: f.studentID, StartedAt: time.Now().Add(-startedAgo),
		Status: assessments.QuizAttemptStatusInProgress, DraftAnswers: raw,
	})
	return id
}

func errCode(err error) string {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return ""
}

func TestAudit_DraftQuizNotAvailableToStudents(t *testing.T) {
	f := newQuizFixture(t, false, 0)
	_, err := f.svc.StartAttempt(context.Background(), StartAttemptCommand{QuizID: f.quizID, StudentID: f.studentID})
	if errCode(err) != "QUIZ_NOT_FOUND" {
		t.Fatalf("expected QUIZ_NOT_FOUND for unpublished quiz, got %v", err)
	}
}

func TestAudit_LateSubmissionIsScoredFromDraft(t *testing.T) {
	f := newQuizFixture(t, true, 60) // 1 minute limit
	// Draft saved during the attempt had the wrong answer; the student keeps
	// the attempt open for 10 minutes and then submits the right answer.
	attemptID := f.addAttempt(10*time.Minute, map[string]interface{}{f.questionID.String(): f.wrongID.String()})

	resp, err := f.svc.SubmitAttempt(context.Background(), SubmitAttemptCommand{
		AttemptID: attemptID, StudentID: f.studentID,
		Answers: []QuizAnswerCommand{{QuestionID: f.questionID, SelectedOptions: []uuid.UUID{f.correctID}}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ScorePercent != 0 || resp.Passed {
		t.Fatalf("answers submitted after the time limit must not be scored; got %.0f%%", resp.ScorePercent)
	}
}

func TestAudit_SubmissionWithinGraceUsesSubmittedAnswers(t *testing.T) {
	f := newQuizFixture(t, true, 60)
	attemptID := f.addAttempt(70*time.Second, nil) // 10s after the limit, inside the 30s grace
	resp, err := f.svc.SubmitAttempt(context.Background(), SubmitAttemptCommand{
		AttemptID: attemptID, StudentID: f.studentID,
		Answers: []QuizAnswerCommand{{QuestionID: f.questionID, SelectedOptions: []uuid.UUID{f.correctID}}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ScorePercent != 100 {
		t.Fatalf("submission within the grace period should be scored normally, got %.0f%%", resp.ScorePercent)
	}
}

func TestAudit_AutoSubmitScoresSavedDraft(t *testing.T) {
	f := newQuizFixture(t, true, 60)
	attemptID := f.addAttempt(5*time.Minute, map[string]interface{}{f.questionID.String(): f.correctID.String()})
	resp, err := f.svc.AutoSubmitAttempt(context.Background(), attemptID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ScorePercent != 100 {
		t.Fatalf("auto-submit must score the saved draft, got %.0f%%", resp.ScorePercent)
	}
}

func TestAudit_DraftAnswerParsing(t *testing.T) {
	q1, q2, q3 := uuid.New(), uuid.New(), uuid.New()
	o1, o2 := uuid.New(), uuid.New()
	raw, _ := json.Marshal(map[string]interface{}{
		q1.String():  o1.String(),
		q2.String():  []string{o1.String(), o2.String()},
		q3.String():  map[string]string{"text": "Paris"},
		"not-a-uuid": "x",
	})
	got := map[uuid.UUID]QuizAnswerCommand{}
	for _, a := range draftAnswersToCommands(raw) {
		got[a.QuestionID] = a
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 parsed answers, got %d", len(got))
	}
	if len(got[q1].SelectedOptions) != 1 || len(got[q2].SelectedOptions) != 2 || got[q3].TextAnswer != "Paris" {
		t.Fatalf("unexpected parse result: %+v", got)
	}
	if len(draftAnswersToCommands([]byte("{bad"))) != 0 {
		t.Fatalf("malformed draft should yield no answers")
	}
}
