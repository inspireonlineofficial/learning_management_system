package assessments

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"lms-backend/internal/domain/assessments"
	"lms-backend/internal/domain/courses"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Grading engine (pure)

func TestEngine_ParseNumericAnswer(t *testing.T) {
	cases := []struct {
		in, unit string
		want     float64
		ok       bool
	}{
		{"42", "", 42, true},
		{"-3.5", "", -3.5, true},
		{"−3.5", "", -3.5, true}, // Unicode minus
		{"6.02e23", "", 6.02e23, true},
		{"1.6E-19", "", 1.6e-19, true},
		{"3.0×10^8", "", 3.0e8, true},
		{"3.0x10^-4", "", 3.0e-4, true},
		{"3.0*10^(5)", "", 3.0e5, true},
		{"9.81 m/s^2", "m/s^2", 9.81, true},
		{"9.81m/s^2", "m/s^2", 9.81, true},
		{"1 000", "", 1000, true},
		{"+7", "", 7, true},
		{"1,5", "", 0, false}, // ambiguous comma rejected
		{"abc", "", 0, false},
		{"NaN", "", 0, false},
		{"Inf", "", 0, false},
		{"", "", 0, false},
	}
	for _, c := range cases {
		got, ok := ParseNumericAnswer(c.in, c.unit)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("ParseNumericAnswer(%q,%q) = %v,%v want %v,%v", c.in, c.unit, got, ok, c.want, c.ok)
		}
	}
}

func numericQuestion(answer, tol float64, tolType, unit string) *assessments.Question {
	return &assessments.Question{
		ID: uuid.New(), Type: assessments.QuestionTypeNumeric, Body: "g?", Marks: 2,
		Settings: assessments.QuestionSettings{NumericAnswer: &answer, Tolerance: tol, ToleranceType: tolType, Unit: unit},
	}
}

func TestEngine_NumericTolerance(t *testing.T) {
	q := numericQuestion(9.81, 0.05, "absolute", "m/s^2")
	for in, want := range map[string]string{
		"9.81": OutcomeCorrect, "9.85": OutcomeCorrect, "9.76": OutcomeCorrect, "9.9": OutcomeIncorrect,
		"9.81 m/s^2": OutcomeCorrect, "": OutcomeUnanswered, "ten": OutcomeIncorrect,
	} {
		got := gradeQuestion(q, nil, AnswerInput{Text: in}, false)
		if got.Status != want {
			t.Errorf("abs %q: status %s want %s", in, got.Status, want)
		}
		if want == OutcomeCorrect && got.Awarded != 2 {
			t.Errorf("abs %q: awarded %v want 2", in, got.Awarded)
		}
	}
	p := numericQuestion(6.022e23, 1, "percent", "")
	if gradeQuestion(p, nil, AnswerInput{Text: "6.0e23"}, false).Status != OutcomeCorrect {
		t.Error("6.0e23 is within 1% of 6.022e23")
	}
	if gradeQuestion(p, nil, AnswerInput{Text: "5.9e23"}, false).Status != OutcomeIncorrect {
		t.Error("5.9e23 is outside 1% of 6.022e23")
	}
	exact := numericQuestion(0.3, 0, "absolute", "")
	if gradeQuestion(exact, nil, AnswerInput{Text: "0.30"}, false).Status != OutcomeCorrect {
		t.Error("exact numeric match with zero tolerance must be correct")
	}
}

func choiceQuestion(t assessments.QuestionType, marks float64, correct, wrong int) (*assessments.Question, []*assessments.QuestionOption) {
	q := &assessments.Question{ID: uuid.New(), Type: t, Body: "pick", Marks: marks}
	var opts []*assessments.QuestionOption
	for i := 0; i < correct; i++ {
		opts = append(opts, &assessments.QuestionOption{ID: uuid.New(), QuestionID: q.ID, Body: "c", IsCorrect: true, Position: len(opts) + 1})
	}
	for i := 0; i < wrong; i++ {
		opts = append(opts, &assessments.QuestionOption{ID: uuid.New(), QuestionID: q.ID, Body: "w", Position: len(opts) + 1})
	}
	return q, opts
}

func TestEngine_MultipleSelectPartialCredit(t *testing.T) {
	q, opts := choiceQuestion(assessments.QuestionTypeMultiple, 4, 2, 2)
	c1, c2, w1 := opts[0].ID, opts[1].ID, opts[2].ID

	all := gradeQuestion(q, opts, AnswerInput{SelectedOptionIDs: []uuid.UUID{c1, c2}}, true)
	if all.Status != OutcomeCorrect || all.Awarded != 4 {
		t.Fatalf("all correct: %+v", all)
	}
	half := gradeQuestion(q, opts, AnswerInput{SelectedOptionIDs: []uuid.UUID{c1}}, true)
	if half.Status != OutcomePartial || half.Awarded != 2 {
		t.Fatalf("one of two correct with partial credit: %s %v", half.Status, half.Awarded)
	}
	cancel := gradeQuestion(q, opts, AnswerInput{SelectedOptionIDs: []uuid.UUID{c1, w1}}, true)
	if cancel.Awarded != 0 || cancel.Status != OutcomeIncorrect {
		t.Fatalf("a wrong pick cancels a right pick: %s %v", cancel.Status, cancel.Awarded)
	}
	strict := gradeQuestion(q, opts, AnswerInput{SelectedOptionIDs: []uuid.UUID{c1}}, false)
	if strict.Awarded != 0 {
		t.Fatalf("without partial credit an incomplete answer earns nothing")
	}
}

func TestEngine_ChoiceIgnoresForeignAndMultiplePicks(t *testing.T) {
	q, opts := choiceQuestion(assessments.QuestionTypeSingle, 1, 1, 2)
	foreign := gradeQuestion(q, opts, AnswerInput{SelectedOptionIDs: []uuid.UUID{uuid.New()}}, false)
	if foreign.Status != OutcomeUnanswered || len(foreign.SelectedOptionIDs) != 0 {
		t.Fatalf("an option from another question must be ignored: %+v", foreign)
	}
	both := gradeQuestion(q, opts, AnswerInput{SelectedOptionIDs: []uuid.UUID{opts[0].ID, opts[1].ID}}, false)
	if both.Status != OutcomeIncorrect || both.Awarded != 0 {
		t.Fatalf("picking several options on a single-answer question is never correct")
	}
	dup := gradeQuestion(q, opts, AnswerInput{SelectedOptionIDs: []uuid.UUID{opts[0].ID, opts[0].ID}}, false)
	if dup.Status != OutcomeCorrect {
		t.Fatalf("duplicate submission of the same option counts once")
	}
}

func TestEngine_ShortAnswerAndLongAnswer(t *testing.T) {
	q := &assessments.Question{ID: uuid.New(), Type: assessments.QuestionTypeShort, Marks: 1}
	opts := []*assessments.QuestionOption{{ID: uuid.New(), Body: "H2SO4", IsCorrect: true}, {ID: uuid.New(), Body: "sulfuric acid", IsCorrect: true}}
	if gradeQuestion(q, opts, AnswerInput{Text: "  Sulfuric   ACID "}, false).Status != OutcomeCorrect {
		t.Error("short answers match case-insensitively with normalised spaces")
	}
	q.Settings.CaseSensitive = true
	if gradeQuestion(q, opts, AnswerInput{Text: "h2so4"}, false).Status != OutcomeIncorrect {
		t.Error("case-sensitive keys (chemical formulae) must respect case")
	}
	long := &assessments.Question{ID: uuid.New(), Type: assessments.QuestionTypeLong, Marks: 10}
	pending := gradeQuestion(long, nil, AnswerInput{Text: "An essay"}, false)
	if pending.Status != OutcomePending || pending.Awarded != 0 {
		t.Fatalf("long answers wait for manual grading: %+v", pending)
	}
	if gradeQuestion(long, nil, AnswerInput{Text: "   "}, false).Status != OutcomeUnanswered {
		t.Error("a blank long answer is unanswered, not pending")
	}
	awarded, max, isPending := gradeTotals([]QuestionOutcome{pending, {Marks: 2, Awarded: 2, Status: OutcomeCorrect}})
	if awarded != 2 || max != 12 || !isPending {
		t.Fatalf("totals: %v/%v pending=%v", awarded, max, isPending)
	}
}

// ---------------------------------------------------------------------------
// Service-level behaviour

type engineFixture struct {
	svc       *service
	quizzes   *mockQuizRepo
	questions *mockQuestionRepo
	options   *mockQuestionOptionRepo
	attempts  *mockQuizAttemptRepo
	quiz      *assessments.Quiz
	teacherID uuid.UUID
	studentID uuid.UUID
	single    *assessments.Question
	right     uuid.UUID
	wrong     uuid.UUID
	long      *assessments.Question
}

func newEngineFixture(t *testing.T, mutate func(q *assessments.Quiz)) *engineFixture {
	t.Helper()
	ctx := context.Background()
	f := &engineFixture{
		quizzes: newMockQuizRepo(), questions: newMockQuestionRepo(), options: newMockQuestionOptionRepo(),
		attempts: newMockQuizAttemptRepo(), teacherID: uuid.New(), studentID: uuid.New(),
	}
	courseRepo := newMockCourseRepo()
	courseID := uuid.New()
	courseRepo.Create(ctx, &courses.Course{ID: courseID, TeacherID: f.teacherID, Title: "Physics"})
	f.quiz = &assessments.Quiz{
		ID: uuid.New(), CourseID: courseID, Title: "Mechanics", IsPublished: true, IsFree: true,
		PassingScorePercent: 50, Mode: assessments.QuizModeExam,
	}
	if mutate != nil {
		mutate(f.quiz)
	}
	f.quizzes.Create(ctx, f.quiz)
	f.single = &assessments.Question{ID: uuid.New(), QuizID: f.quiz.ID, Type: assessments.QuestionTypeSingle, Body: "F = ?", Marks: 2, Position: 1, Hint: "Newton II", Explanation: "F = ma"}
	f.long = &assessments.Question{ID: uuid.New(), QuizID: f.quiz.ID, Type: assessments.QuestionTypeLong, Body: "Explain inertia", Marks: 8, Position: 2}
	f.questions.Create(ctx, f.single)
	f.questions.Create(ctx, f.long)
	f.right, f.wrong = uuid.New(), uuid.New()
	f.options.Create(ctx, &assessments.QuestionOption{ID: f.right, QuestionID: f.single.ID, Body: "ma", IsCorrect: true, Position: 1})
	f.options.Create(ctx, &assessments.QuestionOption{ID: f.wrong, QuestionID: f.single.ID, Body: "mv", Position: 2})
	svc := NewService(f.quizzes, f.questions, f.options, f.attempts, newMockAssignmentRepo(), newMockSubmissionRepo(),
		newMockSubmissionFileRepo(), newMockGradeRepo(), courseRepo, &mockStorageClientProp{}, &mockNotificationQueueProp{}, "bucket")
	f.svc = svc.(*service)
	return f
}

type countingNotifier struct{ calls int }

func (n *countingNotifier) QuizPassed(ctx context.Context, studentID, quizID uuid.UUID, scorePercent float64) error {
	n.calls++
	return nil
}

func TestEngine_SubmitIsIdempotentAndPendingUntilGraded(t *testing.T) {
	f := newEngineFixture(t, nil)
	notifier := &countingNotifier{}
	f.svc.SetQuizPassNotifier(notifier)
	ctx := context.Background()

	attempt, err := f.svc.StartAttempt(ctx, StartAttemptCommand{QuizID: f.quiz.ID, StudentID: f.studentID})
	if err != nil {
		t.Fatal(err)
	}
	answers := []QuizAnswerCommand{
		{QuestionID: f.single.ID, SelectedOptions: []uuid.UUID{f.right}},
		{QuestionID: f.long.ID, TextAnswer: "Objects resist changes in motion."},
	}
	first, err := f.svc.SubmitAttempt(ctx, SubmitAttemptCommand{AttemptID: attempt.ID, StudentID: f.studentID, Answers: answers})
	if err != nil {
		t.Fatal(err)
	}
	if first.GradingStatus != "pending" || first.ScorePoints != 2 || first.MaxPoints != 10 || first.Passed {
		t.Fatalf("written answer must leave the attempt pending: %+v", first)
	}
	stored, _ := f.attempts.FindByID(ctx, attempt.ID)
	if stored.ScorePercent != nil || stored.Passed != nil {
		t.Fatal("a pending attempt must not store a final percentage or pass/fail")
	}

	// Replay with different answers: the recorded result is returned unchanged.
	replay, err := f.svc.SubmitAttempt(ctx, SubmitAttemptCommand{AttemptID: attempt.ID, StudentID: f.studentID,
		Answers: []QuizAnswerCommand{{QuestionID: f.single.ID, SelectedOptions: []uuid.UUID{f.wrong}}}})
	if err != nil {
		t.Fatalf("a replayed submit must not error: %v", err)
	}
	if replay.ScorePoints != first.ScorePoints || replay.AttemptID != first.AttemptID {
		t.Fatalf("replay changed the result: %+v vs %+v", replay, first)
	}

	// Manual grading: over-max rejected, then valid marks finalise the attempt.
	if _, err := f.svc.GradeAttempt(ctx, GradeAttemptCommand{QuizID: f.quiz.ID, AttemptID: attempt.ID, TeacherID: f.teacherID,
		Grades: []ManualGradeCommand{{QuestionID: f.long.ID, Awarded: 9}}}); errCode(err) != "INVALID_MARKS" {
		t.Fatalf("awarding more than the question's marks must fail, got %v", err)
	}
	if _, err := f.svc.GradeAttempt(ctx, GradeAttemptCommand{QuizID: f.quiz.ID, AttemptID: attempt.ID, TeacherID: f.teacherID,
		Grades: []ManualGradeCommand{{QuestionID: f.single.ID, Awarded: 0}}}); errCode(err) != "NOT_MANUALLY_GRADABLE" {
		t.Fatalf("auto-graded choice questions cannot be re-marked by hand, got %v", err)
	}
	if _, err := f.svc.GradeAttempt(ctx, GradeAttemptCommand{QuizID: f.quiz.ID, AttemptID: attempt.ID, TeacherID: uuid.New(),
		Grades: []ManualGradeCommand{{QuestionID: f.long.ID, Awarded: 5}}}); errCode(err) != "FORBIDDEN" {
		t.Fatalf("another teacher must not grade this course, got %v", err)
	}
	detail, err := f.svc.GradeAttempt(ctx, GradeAttemptCommand{QuizID: f.quiz.ID, AttemptID: attempt.ID, TeacherID: f.teacherID,
		Grades: []ManualGradeCommand{{QuestionID: f.long.ID, Awarded: 6, Feedback: "Good"}}})
	if err != nil {
		t.Fatal(err)
	}
	if detail.GradingStatus != "graded" || detail.ScorePercent == nil || *detail.ScorePercent != 80 || detail.Passed == nil || !*detail.Passed {
		t.Fatalf("after grading: %+v", detail.TeacherAttemptRow)
	}
	if notifier.calls != 1 {
		t.Fatalf("points notifier calls = %d, want exactly 1", notifier.calls)
	}
}

func TestEngine_AutosaveRevisionsAndSanitising(t *testing.T) {
	f := newEngineFixture(t, nil)
	ctx := context.Background()
	attempt, _ := f.svc.StartAttempt(ctx, StartAttemptCommand{QuizID: f.quiz.ID, StudentID: f.studentID})

	newer, err := f.svc.SaveAttemptAnswers(ctx, SaveAttemptAnswersCommand{AttemptID: attempt.ID, StudentID: f.studentID, Revision: 5,
		Answers: map[string]interface{}{f.single.ID.String(): f.right.String(), uuid.New().String(): "junk"}})
	if err != nil || newer.Saved == nil || !*newer.Saved {
		t.Fatalf("revision 5 must save: %v %+v", err, newer)
	}
	stale, err := f.svc.SaveAttemptAnswers(ctx, SaveAttemptAnswersCommand{AttemptID: attempt.ID, StudentID: f.studentID, Revision: 3,
		Answers: map[string]interface{}{f.single.ID.String(): f.wrong.String()}})
	if err != nil || stale.Saved == nil || *stale.Saved {
		t.Fatalf("an older revision arriving late must be ignored: %v %+v", err, stale)
	}
	stored, _ := f.attempts.FindByID(ctx, attempt.ID)
	var draft map[string]interface{}
	_ = json.Unmarshal(stored.DraftAnswers, &draft)
	if draft[f.single.ID.String()] != f.right.String() || len(draft) != 1 {
		t.Fatalf("draft = %v; want only the newest valid answer", draft)
	}
	if _, err := f.svc.SaveAttemptAnswers(ctx, SaveAttemptAnswersCommand{AttemptID: attempt.ID, StudentID: uuid.New(), Revision: 9,
		Answers: map[string]interface{}{}}); errCode(err) != "FORBIDDEN" {
		t.Fatalf("another student must not write this attempt, got %v", err)
	}
}

func TestEngine_AvailabilityWindowAndResume(t *testing.T) {
	future := time.Now().Add(time.Hour)
	f := newEngineFixture(t, func(q *assessments.Quiz) { q.AvailableFrom = &future })
	if _, err := f.svc.StartAttempt(context.Background(), StartAttemptCommand{QuizID: f.quiz.ID, StudentID: f.studentID}); errCode(err) != "ASSESSMENT_NOT_OPEN" {
		t.Fatalf("got %v", err)
	}
	past := time.Now().Add(-time.Hour)
	g := newEngineFixture(t, func(q *assessments.Quiz) { q.AvailableUntil = &past })
	if _, err := g.svc.StartAttempt(context.Background(), StartAttemptCommand{QuizID: g.quiz.ID, StudentID: g.studentID}); errCode(err) != "ASSESSMENT_CLOSED" {
		t.Fatalf("got %v", err)
	}

	h := newEngineFixture(t, func(q *assessments.Quiz) { q.MaxAttempts = 1; q.TimeLimitSeconds = 600 })
	a1, err := h.svc.StartAttempt(context.Background(), StartAttemptCommand{QuizID: h.quiz.ID, StudentID: h.studentID})
	if err != nil {
		t.Fatal(err)
	}
	a2, err := h.svc.StartAttempt(context.Background(), StartAttemptCommand{QuizID: h.quiz.ID, StudentID: h.studentID})
	if err != nil || a2.ID != a1.ID {
		t.Fatalf("a second start must resume the same attempt, not use another one: %v", err)
	}
	if a1.ExpiresAt == nil || a1.ServerNow.IsZero() {
		t.Fatal("timed attempts carry a server-authoritative deadline and server time")
	}
}

func TestEngine_ShuffleIsStablePerAttempt(t *testing.T) {
	f := newEngineFixture(t, func(q *assessments.Quiz) { q.ShuffleQuestions = true; q.ShuffleOptions = true })
	for i := 0; i < 8; i++ {
		q := &assessments.Question{ID: uuid.New(), QuizID: f.quiz.ID, Type: assessments.QuestionTypeTrueFalse, Body: "tf", Marks: 1, Position: 10 + i}
		f.questions.Create(context.Background(), q)
	}
	attempt, _ := f.svc.StartAttempt(context.Background(), StartAttemptCommand{QuizID: f.quiz.ID, StudentID: f.studentID})
	again, _ := f.svc.StartAttempt(context.Background(), StartAttemptCommand{QuizID: f.quiz.ID, StudentID: f.studentID})
	for i := range attempt.Questions {
		if attempt.Questions[i].ID != again.Questions[i].ID {
			t.Fatal("question order changed after resuming the attempt")
		}
	}
}

func TestEngine_PracticeHintsAndExamSecrecy(t *testing.T) {
	exam := newEngineFixture(t, nil)
	ctx := context.Background()
	attempt, _ := exam.svc.StartAttempt(ctx, StartAttemptCommand{QuizID: exam.quiz.ID, StudentID: exam.studentID})
	for _, q := range attempt.Questions {
		if q.Hint != "" {
			t.Fatal("exam mode must not send hints")
		}
	}
	if _, err := exam.svc.CheckAnswer(ctx, CheckAnswerCommand{AttemptID: attempt.ID, StudentID: exam.studentID, QuestionID: exam.single.ID, Answer: exam.right.String()}); errCode(err) != "CHECK_NOT_ALLOWED" {
		t.Fatalf("exam mode must not check answers, got %v", err)
	}
	result, _ := exam.svc.SubmitAttempt(ctx, SubmitAttemptCommand{AttemptID: attempt.ID, StudentID: exam.studentID,
		Answers: []QuizAnswerCommand{{QuestionID: exam.single.ID, SelectedOptions: []uuid.UUID{exam.wrong}}}})
	for _, r := range result.Review {
		if r.Awarded != nil || len(r.CorrectOptionIDs) > 0 || r.Explanation != "" {
			t.Fatalf("unreleased review leaked the answer key: %+v", r)
		}
		for _, o := range r.Options {
			if o.IsCorrect {
				t.Fatal("unreleased review marked the correct option")
			}
		}
	}

	practice := newEngineFixture(t, func(q *assessments.Quiz) { q.Mode = assessments.QuizModePractice })
	pa, _ := practice.svc.StartAttempt(ctx, StartAttemptCommand{QuizID: practice.quiz.ID, StudentID: practice.studentID})
	hinted := false
	for _, q := range pa.Questions {
		hinted = hinted || q.Hint == "Newton II"
	}
	if !hinted {
		t.Fatal("practice mode sends hints")
	}
	check, err := practice.svc.CheckAnswer(ctx, CheckAnswerCommand{AttemptID: pa.ID, StudentID: practice.studentID, QuestionID: practice.single.ID, Answer: practice.right.String()})
	if err != nil || !check.Correct || check.Explanation != "F = ma" {
		t.Fatalf("practice check: %v %+v", err, check)
	}
}

func TestEngine_UpdateQuizKeepsQuestionIDsAndHistory(t *testing.T) {
	f := newEngineFixture(t, nil)
	ctx := context.Background()
	attempt, _ := f.svc.StartAttempt(ctx, StartAttemptCommand{QuizID: f.quiz.ID, StudentID: f.studentID})
	_, _ = f.svc.SubmitAttempt(ctx, SubmitAttemptCommand{AttemptID: attempt.ID, StudentID: f.studentID,
		Answers: []QuizAnswerCommand{{QuestionID: f.single.ID, SelectedOptions: []uuid.UUID{f.right}}}})

	id := f.single.ID
	right, wrong := f.right, f.wrong
	updated, err := f.svc.UpdateQuiz(ctx, UpdateQuizCommand{
		QuizID: f.quiz.ID, TeacherID: f.teacherID, Title: "Mechanics v2", PassingScorePercent: 50, IsPublished: true, IsFree: true,
		Questions: []CreateQuestionCommand{{ID: &id, Type: "single", Body: "Force equals?", Marks: 5, Options: []CreateQuestionOptionCommand{
			{ID: &right, Body: "m·a", IsCorrect: true}, {ID: &wrong, Body: "m·v"},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Questions) != 1 || updated.Questions[0].ID != id || updated.Questions[0].Options[0].ID != right {
		t.Fatalf("question/option IDs must survive an edit: %+v", updated.Questions)
	}
	// The submitted attempt still shows the original wording and marks.
	result, err := f.svc.GetStudentQuizAttemptResult(ctx, f.quiz.ID, attempt.ID, f.studentID)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Review) != 2 || result.Review[0].Body != "F = ?" || result.Review[0].Marks != 2 {
		t.Fatalf("historical attempt changed after editing the quiz: %+v", result.Review)
	}
}

func TestEngine_DraftQuizzesHiddenFromStudentList(t *testing.T) {
	f := newEngineFixture(t, func(q *assessments.Quiz) { q.IsPublished = false })
	if err := f.svc.ensureStudentCanAccessQuiz(context.Background(), f.studentID, f.quiz); errCode(err) != "QUIZ_NOT_FOUND" {
		t.Fatalf("draft quiz must be invisible, got %v", err)
	}
}

func TestEngine_DeleteQuizWithAttemptsRefused(t *testing.T) {
	f := newEngineFixture(t, nil)
	_, _ = f.svc.StartAttempt(context.Background(), StartAttemptCommand{QuizID: f.quiz.ID, StudentID: f.studentID})
	if err := f.svc.DeleteQuiz(context.Background(), f.quiz.ID, f.teacherID); errCode(err) != "QUIZ_HAS_ATTEMPTS" {
		t.Fatalf("deleting a quiz with attempts would destroy grades, got %v", err)
	}
}

func TestEngine_ExpiredAttemptsAreAutoSubmitted(t *testing.T) {
	f := newEngineFixture(t, func(q *assessments.Quiz) { q.TimeLimitSeconds = 60 })
	raw, _ := json.Marshal(map[string]interface{}{f.single.ID.String(): f.right.String()})
	id := uuid.New()
	f.attempts.Create(context.Background(), &assessments.QuizAttempt{ID: id, QuizID: f.quiz.ID, StudentID: f.studentID,
		StartedAt: time.Now().Add(-2 * time.Minute), Status: assessments.QuizAttemptStatusInProgress, DraftAnswers: raw})
	n, err := f.svc.AutoSubmitExpiredAttempts(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("auto-submitted %d, err %v", n, err)
	}
	stored, _ := f.attempts.FindByID(context.Background(), id)
	if stored.Status != assessments.QuizAttemptStatusAutoSubmitted || stored.ScorePoints == nil || *stored.ScorePoints != 2 {
		t.Fatalf("expired attempt must be graded from its draft: %+v", stored)
	}
}

func TestEngine_SingleChoiceOptionIDSentAsText(t *testing.T) {
	q, opts := choiceQuestion(assessments.QuestionTypeSingle, 2, 1, 1)
	got := gradeQuestion(q, opts, AnswerInput{Text: opts[0].ID.String()}, false)
	if got.Status != OutcomeCorrect || got.Awarded != 2 {
		t.Fatalf("an option ID sent as text must be graded as the selection: %+v", got)
	}
}
